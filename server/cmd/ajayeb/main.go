// Command ajayeb is the Ajayeb game server. One process serves the web
// client, the JSON API and the WebSocket game rooms.
//
// Usage:
//
//	ajayeb [flags]
//
// Run ajayeb -help for the flags.
//
// On SIGINT or SIGTERM the server stops accepting connections, closes every
// WebSocket with status 1012 (service restart, so clients reconnect with
// backoff), stops the rooms (which are already on disk; a room whose last
// write failed tries once more) and exits.
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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/httpapi"
	"github.com/NightMachinery/ajayeb/server/internal/ident"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/webdist"
	"github.com/NightMachinery/ajayeb/server/internal/ws"
)

// version is the build version, set at link time with
// -ldflags "-X main.version=$(git describe --always --dirty)".
var version = "dev"

const (
	defaultAddr        = "127.0.0.1:38951"
	defaultData        = "./data"
	defaultBotDelay    = room.DefaultBotDelay
	defaultBotSearches = 2

	// shutdownTimeout bounds how long the HTTP requests and the WebSocket
	// closing handshakes may take after SIGINT or SIGTERM.
	shutdownTimeout = 10 * time.Second

	// reapEvery is how often old room files are swept (store.Reaper).
	reapEvery = time.Hour
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
	botSearches  int
	botPerAddr   int
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// Once a signal has arrived, restore the default handling, so a second
	// Ctrl-C kills the process at once if shutdown hangs.
	context.AfterFunc(ctx, stop)

	if err := serve(ctx, opts, logger, nil); err != nil {
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
	fset.IntVar(&opts.botSearches, "bot-searches", defaultBotSearches, "how many bot searches may run at once across all rooms")
	fset.IntVar(&opts.botPerAddr, "bot-searches-per-address", ai.DefaultPerKey, "how many of those one client address (an IPv4 address or IPv6 /64) may hold at once; its further bot turns wait without holding up other addresses")
	fset.Uint64Var(&opts.seedBase, "seed-base", 0, "derive room seeds from this `base` for reproducible tests (requires -dev; 0 means random seeds)")
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
	// Seeds from a counter make every deal predictable: whoever recognizes
	// one game's deal knows the hidden cards of every later one.
	if opts.seedBase != 0 && !opts.dev {
		return options{}, errors.New("-seed-base requires -dev; refusing to start")
	}
	if opts.botDelay < 0 {
		return options{}, errors.New("-bot-delay must not be negative")
	}
	if opts.botSearches < 1 {
		return options{}, errors.New("-bot-searches must be at least 1")
	}
	if opts.botPerAddr < 1 {
		return options{}, errors.New("-bot-searches-per-address must be at least 1")
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

// seedSource returns the seeds of new games: base, base+1, ... when base is
// set (reproducible test runs), or nil for the default random seeds.
func seedSource(base uint64) func() uint64 {
	if base == 0 {
		return nil
	}
	var n atomic.Uint64
	return func() uint64 { return base + n.Add(1) - 1 }
}

// serve runs the server until ctx is done, then shuts it down gracefully.
// ready, when set, receives the listening address once connections are
// accepted (port 0 picks a free one).
func serve(ctx context.Context, opts options, logger *slog.Logger, ready func(addr string)) error {
	if err := os.MkdirAll(opts.dataDir, 0o700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	secret, err := ident.LoadSecretLogged(opts.dataDir, logger)
	if err != nil {
		return err
	}
	st, err := store.Open(opts.dataDir)
	if err != nil {
		return err
	}
	static, staticSource, err := staticFS(opts, logger)
	if err != nil {
		return err
	}

	seed := seedSource(opts.seedBase)
	rooms, err := room.NewManager(room.Config{
		Store:    st,
		Logger:   logger,
		BotPool:  ai.NewPoolPerKey(opts.botSearches, opts.botPerAddr),
		BotDelay: opts.botDelay,
		Seed:     seed,
		Claims:   secret,
	})
	if err != nil {
		return err
	}
	defer rooms.Close() // idempotent; the normal path closes it in order below

	sockets, err := ws.New(ws.Config{
		Rooms:          rooms,
		Identity:       secret,
		Version:        version,
		Origins:        opts.origins,
		Dev:            opts.dev,
		TrustedProxies: opts.trustProxy,
		Logger:         logger,
	})
	if err != nil {
		return err
	}
	handler, err := httpapi.New(httpapi.Config{
		Version:        version,
		Dev:            opts.dev,
		DevEndpoints:   opts.devEndpoints,
		WebSocket:      sockets,
		DevRooms:       rooms,
		Seed:           seed,
		Logs:           st,
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

	reapCtx, stopReaper := context.WithCancel(context.Background())
	reaperDone := make(chan struct{})
	go func() {
		defer close(reaperDone)
		runReaper(reapCtx, store.NewReaper(st, nil), logger)
	}()

	seedLabel := "random"
	if opts.seedBase != 0 {
		seedLabel = strconv.FormatUint(opts.seedBase, 10)
	}
	logger.Info("listening",
		"addr", ln.Addr().String(),
		"version", version,
		"static", staticSource,
		"data", opts.dataDir,
		"dev", opts.dev,
		"devEndpoints", opts.devEndpoints,
		"origins", strings.Join(opts.origins, ","),
		"trustedProxies", len(opts.trustProxy),
		"botDelay", opts.botDelay,
		"botSearches", opts.botSearches,
		"botSearchesPerAddress", opts.botPerAddr,
		"seedBase", seedLabel,
	)

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	if ready != nil {
		ready(ln.Addr().String())
	}

	var failed error
	select {
	case err := <-serveErr:
		failed = err // Serve stopped on its own: shut the rest down too
	case <-ctx.Done():
		logger.Info("shutting down", "timeout", shutdownTimeout)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	// Stop accepting connections and finish the plain HTTP requests. Upgraded
	// sockets are not the http.Server's any more.
	httpErr := srv.Shutdown(shutdownCtx)
	// Close every socket with 1012 and detach its client from its room.
	wsErr := sockets.Shutdown(shutdownCtx)
	// Stop the rooms and their bots; a room whose last write failed tries
	// once more.
	rooms.Close()
	stopReaper()
	<-reaperDone

	if failed == nil {
		failed = <-serveErr
	}
	if errors.Is(failed, http.ErrServerClosed) {
		failed = nil
	}
	if err := errors.Join(failed, httpErr, wsErr); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	logger.Info("stopped")
	return nil
}

// runReaper deletes old room files now and then every reapEvery, until ctx
// is done.
func runReaper(ctx context.Context, r *store.Reaper, logger *slog.Logger) {
	sweep := func() {
		removed, err := r.Sweep()
		if err != nil {
			logger.Warn("sweeping old rooms failed", "err", err)
		}
		if len(removed) > 0 {
			logger.Info("removed old rooms", "count", len(removed))
		}
	}
	sweep()
	tick := time.NewTicker(reapEvery)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			sweep()
		case <-ctx.Done():
			return
		}
	}
}
