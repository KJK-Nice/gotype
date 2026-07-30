// Package invite builds stable share / challenge strings for SSH play.
package invite

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultHost = "game.gotype.fun"
	defaultUser = "play"
	defaultPort = "58372"
)

// SSHCommand returns the canonical pasteable connect string.
// Override with SHARE_SSH_CMD, or SHARE_HOST / SHARE_PORT / SHARE_USER,
// falling back to Railway TCP proxy port.
func SSHCommand() string {
	if cmd := strings.TrimSpace(os.Getenv("SHARE_SSH_CMD")); cmd != "" {
		return cmd
	}
	user := envOr("SHARE_USER", defaultUser)
	host := envOr("SHARE_HOST", defaultHost)
	port := strings.TrimSpace(os.Getenv("SHARE_PORT"))
	if port == "" {
		port = strings.TrimSpace(os.Getenv("RAILWAY_TCP_PROXY_PORT"))
	}
	if port == "" {
		port = defaultPort
	}
	if port == "22" {
		return fmt.Sprintf("ssh %s@%s", user, host)
	}
	return fmt.Sprintf("ssh %s@%s -p %s", user, host, port)
}

// DemoSSHCommand is the spectate/demo connect string (user "demo").
func DemoSSHCommand() string {
	cmd := SSHCommand()
	return strings.Replace(cmd, "play@", "demo@", 1)
}

// BeatMe is the podium / lobby invite line.
func BeatMe(roomCode string) string {
	roomCode = strings.TrimSpace(strings.ToUpper(roomCode))
	if roomCode == "" {
		return "Beat me: " + SSHCommand()
	}
	return fmt.Sprintf("Beat me: %s · room %s", SSHCommand(), roomCode)
}

// RaceMe is the solo-result challenge line.
func RaceMe(wpm float64) string {
	if wpm < 1 {
		return "Race me: " + SSHCommand()
	}
	return fmt.Sprintf("Race me: %s · beat %.0f wpm", SSHCommand(), wpm)
}

// HostFingerprint returns SHA256 fingerprint hint for first-time SSH (optional env).
func HostFingerprint() string {
	return strings.TrimSpace(os.Getenv("SSH_HOST_FINGERPRINT"))
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
