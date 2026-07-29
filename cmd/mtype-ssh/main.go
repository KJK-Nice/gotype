package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"

	"github.com/kjkusap/monkeytype-clone/internal/ui"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "2222"
	}
	addr := net.JoinHostPort("0.0.0.0", port)

	hostKey, err := hostKeyOption()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mtype-ssh: host key: %v\n", err)
		os.Exit(1)
	}

	s, err := wish.NewServer(
		wish.WithAddress(addr),
		hostKey,
		wish.WithPasswordAuth(func(ssh.Context, string) bool { return true }),
		// Last middleware runs first: logging → activeterm → bubbletea.
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mtype-ssh: %v\n", err)
		os.Exit(1)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Starting SSH server", "addr", addr)
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
		os.Exit(1)
	}
}

func teaHandler(ssh.Session) (tea.Model, []tea.ProgramOption) {
	return ui.New(), []tea.ProgramOption{tea.WithAltScreen()}
}

func hostKeyOption() (ssh.Option, error) {
	if pem := os.Getenv("SSH_HOST_KEY"); pem != "" {
		return wish.WithHostKeyPEM([]byte(pem)), nil
	}
	// Local/dev: auto-generate under a temp subdir (not durable across restarts).
	dir := filepath.Join(os.TempDir(), "mtype-ssh")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return wish.WithHostKeyPath(filepath.Join(dir, "host_ed25519")), nil
}
