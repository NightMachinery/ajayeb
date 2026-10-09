package main

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// rssInterval is how often --server-pid samples the server's resident
// memory while games are in flight.
const rssInterval = 100 * time.Millisecond

// rssSampler records the peak resident memory of one process, read with
// ps(1) (whose rss column is in KiB on both macOS and Linux), for the
// memory figure of a --concurrency run. The server must run on this host.
type rssSampler struct {
	pid int

	mu      sync.Mutex
	peakKiB int
	samples int
	err     error
}

// readRSS returns pid's resident set size in KiB.
func readRSS(ctx context.Context, pid int) (int, error) {
	out, err := exec.CommandContext(ctx, "ps", "-o", "rss=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0, fmt.Errorf("ps -p %d: %w", pid, err)
	}
	kib, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("ps -p %d: unexpected output %q", pid, out)
	}
	return kib, nil
}

// sample takes one reading. A failed reading is kept (the first one) and
// reported by summary, since a memory figure with gaps is not one to trust.
func (r *rssSampler) sample(ctx context.Context) {
	kib, err := readRSS(ctx, r.pid)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		if r.err == nil && ctx.Err() == nil {
			r.err = err
		}
		return
	}
	r.samples++
	r.peakKiB = max(r.peakKiB, kib)
}

// run samples until ctx is done.
func (r *rssSampler) run(ctx context.Context) {
	ticker := time.NewTicker(rssInterval)
	defer ticker.Stop()
	r.sample(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.sample(ctx)
		}
	}
}

// summary is the one line --server-pid prints after the games, and an
// error if any sample failed.
func (r *rssSampler) summary() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	line := fmt.Sprintf("server pid=%d peak-rss=%.1fMB (%d KiB, %d samples every %s)",
		r.pid, float64(r.peakKiB)/1024, r.peakKiB, r.samples, rssInterval)
	if r.err == nil && r.samples == 0 {
		return line, fmt.Errorf("no RSS sample of pid %d", r.pid)
	}
	return line, r.err
}
