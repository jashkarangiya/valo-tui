// Command valo-tui-ssh serves the TUI over SSH with Wish. Each connection gets
// its own tea.Program and model — no shared state. Run it, then from another
// terminal: `ssh -p 23234 localhost`.
//
// Bind/host-key are configurable via flags or env so the same binary serves a
// local dev port (23234) and a public deploy on :22 (bare `ssh host`):
//
//	valo-tui-ssh --port 22 --host-key /var/lib/valo-tui/.ssh/id_ed25519
package main

import (
	"context"
	"errors"
	"flag"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"github.com/charmbracelet/ssh"

	"github.com/jashkarangiya/valo-tui/internal/app"
)

// maxSessions caps concurrent SSH sessions. Past this point a connection flood
// would push the container into the OOM-killer, which drops *everyone* and
// triggers a reconnect stampede; refusing the surplus instead keeps every
// already-connected viewer alive. Sized for a multi-GB container — lower it if
// you run on the default 512 MB CT.
const maxSessions = 512

func main() {
	host := flag.String("host", envOr("VALO_TUI_SSH_HOST", "0.0.0.0"),
		"interface to bind (env VALO_TUI_SSH_HOST)")
	port := flag.String("port", envOr("VALO_TUI_SSH_PORT", "23234"),
		"port to listen on; use 22 for bare `ssh host` (env VALO_TUI_SSH_PORT)")
	hostKey := flag.String("host-key", envOr("VALO_TUI_SSH_HOST_KEY", ".ssh/id_ed25519"),
		"SSH host key path (env VALO_TUI_SSH_HOST_KEY)")
	flag.Parse()

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(*host, *port)),
		wish.WithHostKeyPath(*hostKey),
		// Drop idle/abandoned connections so the public front door can't be tied
		// up: the TUI is read-only, so the only abuse vector is connection count.
		wish.WithIdleTimeout(15*time.Minute),
		// Middleware runs last-listed-first, so logging still records every
		// attempt and the session cap gates before the PTY/program is set up.
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(), // Bubble Tea needs a PTY
			limitSessions(maxSessions),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Fatal("could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("starting SSH server", "host", *host, "port", *port)
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("server crashed", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("shutdown failed", "error", err)
	}
}

// envOr returns the environment value for key, or def when it is unset/empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// limitSessions returns middleware that admits at most limit concurrent
// sessions. The surplus get a one-line "at capacity" notice and a clean
// disconnect instead of dragging the whole server down with them.
func limitSessions(limit int) wish.Middleware {
	var active int64
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			n := atomic.AddInt64(&active, 1)
			defer atomic.AddInt64(&active, -1)
			if n > int64(limit) {
				wish.Printf(sess, "valo-tui is at capacity (%d viewers). Please try again in a minute.\r\n", limit)
				log.Info("session refused: at capacity", "limit", limit, "remote", sess.RemoteAddr())
				return
			}
			next(sess)
		}
	}
}

// teaHandler is invoked once per SSH connection.
func teaHandler(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := sess.Pty()
	m := app.New(pty.Window.Width, pty.Window.Height)
	return m, nil
}
