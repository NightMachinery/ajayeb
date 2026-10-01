// Command ajayeb is the Ajayeb game server. One process serves the web
// client, the JSON API and the WebSocket game rooms.
//
// Usage:
//
//	ajayeb [flags]
//
// Run ajayeb -help for the flags.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/httpapi"
	"github.com/NightMachinery/ajayeb/server/internal/webdist"
)

// version is the build version, set at link time with
// -ldflags "-X main.version=$(git describe --always --dirty)".
var version = "dev"

const (
	defaultAddr     = "127.0.0.1:38951"
	defaultData     = "./data"
	defaultBotDelay = 400 * time.Millisecond

	// shutdownTimeout bounds how long in-flight requests may take to finish
	// after SIGINT or SIGTERM.
	shutdownTimeout = 10 * time.Second
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// options holds the parsed command line.
type options struct {
	addr         string
	dataDir      string
	staticDir    string
	dev          bool
	devEndpoints bool
	origins      []string
	trustProxy   []netip.Prefix
	botDelay     time.Duration
	seedBase     uint64
	showVersion  bool
}

// errBadFlags reports a flag syntax error that the flag package has already
// printed together with the usage text.
var errBadFlags = errors.New("bad flags")

// run is main without the process exit, so tests can drive it.
func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseFlags(args, stderr)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return 0
	case errors.Is(err, errBadFlags):
		return 2
	case err != nil:
		fmt.Fprintf(stderr, "ajayeb: %v\n", err)
		return 2
	}
	if opts.showVersion {
		fmt.Fprintln(stdout, version)
		return 0
	}

	level := slog.LevelInfo
	if opts.dev {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level}))

	if err := serve(opts, logger); err != nil {
		logger.Error("server failed", "err", err)
		return 1
	}
	return 0
}

func parseFlags(args []string, stderr io.Writer) (options, error) {
	opts := options{}
	var origins, proxies []string

	fset := flag.NewFlagSet("ajayeb", flag.ContinueOnError)
	fset.SetOutput(stderr)
	fset.StringVar(&opts.addr, "addr", defaultAddr, "listen `address` (host:port; port 0 picks a free port)")
	fset.StringVar(&opts.dataDir, "data", defaultData, "data `directory` for room files and the identity secret (created if missing)")
	fset.StringVar(&opts.staticDir, "static", "", "serve the web build from this `directory` instead of the embedded one")
	fset.BoolVar(&opts.dev, "dev", false, "development mode: debug logging and loopback WebSocket origins")
	fset.BoolVar(&opts.devEndpoints, "dev-endpoints", false, "enable the test-only /api/dev/ endpoints (requires -dev)")
	fset.Var(listValue{&origins}, "origin", "allowed site `origin` such as https://example.org (repeatable or comma separated)")
	fset.Var(listValue{&proxies}, "trust-proxy", "trusted reverse proxy `IPs or CIDRs` whose X-Forwarded-For is believed (repeatable or comma separated)")
	fset.DurationVar(&opts.botDelay, "bot-delay", defaultBotDelay, "minimum think `time` before a bot move (0 for tests)")
	fset.Uint64Var(&opts.seedBase, "seed-base", 0, "derive room seeds from this `base` for reproducible tests (0 means random seeds)")
	fset.BoolVar(&opts.showVersion, "version", false, "print the version and exit")
	fset.Usage = func() {
		fmt.Fprintf(fset.Output(), "Usage: ajayeb [flags]\n\nThe Ajayeb game server.\n\nFlags:\n")
		fset.PrintDefaults()
	}

	if err := fset.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return options{}, err
		}
		return options{}, errBadFlags
	}
	if fset.NArg() > 0 {
		return options{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fset.Args(), " "))
	}
	if opts.devEndpoints && !opts.dev {
		return options{}, errors.New("-dev-endpoints requires -dev; refusing to start")
	}
	if opts.botDelay < 0 {
		return options{}, errors.New("-bot-delay must not be negative")
	}
	if opts.dataDir == "" {
		return options{}, errors.New("-data must not be empty")
	}

	var err error
	if opts.origins, err = parseOrigins(origins); err != nil {
		return options{}, fmt.Errorf("-origin: %w", err)
	}
	if opts.trustProxy, err = parseProxies(proxies); err != nil {
		return options{}, fmt.Errorf("-trust-proxy: %w", err)
	}
	return opts, nil
}

// listValue is a flag.Value that collects comma separated items across
// repeated uses of a flag.
type listValue struct{ items *[]string }

func (l listValue) String() string {
	if l.items == nil {
		return ""
	}
	return strings.Join(*l.items, ",")
}

func (l listValue) Set(s string) error {
	for item := range strings.SplitSeq(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			*l.items = append(*l.items, item)
		}
	}
	return nil
}

// parseOrigins validates site origins and normalises them to scheme://host
// with a lower-case scheme and host, the form browsers send in the Origin
// header.
func parseOrigins(raw []string) ([]string, error) {
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		u, err := url.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", s, err)
		}
		scheme := strings.ToLower(u.Scheme)
		switch {
		case scheme != "http" && scheme != "https":
			return nil, fmt.Errorf("%q: scheme must be http or https", s)
		case u.Host == "" || u.Hostname() == "":
			return nil, fmt.Errorf("%q: missing host", s)
		case u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "":
			return nil, fmt.Errorf("%q: an origin is only scheme://host[:port]", s)
		}
		out = append(out, scheme+"://"+strings.ToLower(u.Host))
	}
	return out, nil
}

// parseProxies parses IP addresses and CIDR prefixes. A bare address becomes
// a single-address prefix.
func parseProxies(raw []string) ([]netip.Prefix, error) {
	out := make([]netip.Prefix, 0, len(raw))
	for _, s := range raw {
		if strings.Contains(s, "/") {
			p, err := netip.ParsePrefix(s)
			if err != nil {
				return nil, err
			}
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(s)
		if err != nil {
			return nil, err
		}
		a = a.Unmap()
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}

// staticFS picks the web build to serve: the --static directory, else the
// embedded build, else the placeholder page. It also returns a short label
// for the startup log.
func staticFS(opts options, logger *slog.Logger) (fs.FS, string, error) {
	if opts.staticDir != "" {
		dir, err := filepath.Abs(opts.staticDir)
		if err != nil {
			return nil, "", fmt.Errorf("-static: %w", err)
		}
		info, err := os.Stat(dir)
		if err != nil {
			return nil, "", fmt.Errorf("-static: %w", err)
		}
		if !info.IsDir() {
			return nil, "", fmt.Errorf("-static: %s is not a directory", opts.staticDir)
		}
		fsys := os.DirFS(dir)
		if _, err := fs.Stat(fsys, "index.html"); err != nil {
			logger.Warn("static directory has no index.html; app routes will return 404", "dir", opts.staticDir)
		}
		return fsys, "dir", nil
	}
	if fsys, ok := webdist.FS(); ok {
		return fsys, "embedded", nil
	}
	if webdist.Embedded {
		logger.Warn("built with -tags embedweb but the embedded web build has no index.html")
	}
	logger.Warn("no web build available; serving the placeholder page (use -static DIR or build with -tags embedweb)")
	return webdist.Placeholder(), "placeholder", nil
}

func serve(opts options, logger *slog.Logger) error {
	if err := os.MkdirAll(opts.dataDir, 0o700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	static, staticSource, err := staticFS(opts, logger)
	if err != nil {
		return err
	}

	handler, err := httpapi.New(httpapi.Config{
		Version:        version,
		Dev:            opts.dev,
		DevEndpoints:   opts.devEndpoints,
		Origins:        opts.origins,
		TrustedProxies: opts.trustProxy,
	}, static)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", opts.addr)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
		// No WriteTimeout: it would cut long-lived WebSocket connections.
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	seedBase := "random"
	if opts.seedBase != 0 {
		seedBase = strconv.FormatUint(opts.seedBase, 10)
	}
	// botDelay and seedBase are validated and logged now; the room package
	// consumes them once it exists (milestone M4).
	logger.Info("listening",
		"addr", ln.Addr().String(),
		"version", version,
		"static", staticSource,
		"dev", opts.dev,
		"devEndpoints", opts.devEndpoints,
		"origins", strings.Join(opts.origins, ","),
		"trustedProxies", len(opts.trustProxy),
		"botDelay", opts.botDelay,
		"seedBase", seedBase,
	)

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	// Restore default signal handling, so a second Ctrl-C kills the process
	// at once if shutdown hangs.
	stop()
	logger.Info("shutting down", "timeout", shutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("stopped")
	return nil
}
