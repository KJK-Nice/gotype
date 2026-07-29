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
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"

	"github.com/kjkusap/monkeytype-clone/internal/multi"
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

	hub := multi.NewHub()

	s, err := wish.NewServer(
		wish.WithAddress(addr),
		hostKey,
		wish.WithPasswordAuth(func(ssh.Context, string) bool { return true }),
		wish.WithMiddleware(
			bubbletea.MiddlewareWithColorProfile(makeTeaHandler(hub), termenv.TrueColor),
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

func makeTeaHandler(hub *multi.Hub) func(ssh.Session) (tea.Model, []tea.ProgramOption) {
	return func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		lipgloss.SetColorProfile(termenv.TrueColor)
		name := sess.User()
		if name == "" {
			name = "player"
		}
		m := ui.NewWithOptions(ui.Options{
			Hub:        hub,
			PlayerName: name,
			PlayerID:   multi.NewPlayerID(),
		})
		return m, []tea.ProgramOption{tea.WithAltScreen()}
	}
}

func hostKeyOption() (ssh.Option, error) {
	if pem := os.Getenv("SSH_HOST_KEY"); pem != "" {
		return wish.WithHostKeyPEM([]byte(pem)), nil
	}
	dir := filepath.Join(os.TempDir(), "mtype-ssh")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return wish.WithHostKeyPath(filepath.Join(dir, "host_ed25519")), nil
}
