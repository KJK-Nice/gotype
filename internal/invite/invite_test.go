package invite

import "testing"

func TestSSHCommandDefault(t *testing.T) {
	t.Setenv("SHARE_SSH_CMD", "")
	t.Setenv("SHARE_HOST", "")
	t.Setenv("SHARE_PORT", "")
	t.Setenv("SHARE_USER", "")
	t.Setenv("RAILWAY_TCP_PROXY_PORT", "58372")
	got := SSHCommand()
	want := "ssh play@game.gotype.fun -p 58372"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSSHCommandPort22(t *testing.T) {
	t.Setenv("SHARE_SSH_CMD", "")
	t.Setenv("SHARE_HOST", "game.gotype.fun")
	t.Setenv("SHARE_PORT", "22")
	t.Setenv("SHARE_USER", "play")
	got := SSHCommand()
	want := "ssh play@game.gotype.fun"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBeatMe(t *testing.T) {
	t.Setenv("SHARE_SSH_CMD", "ssh play@game.gotype.fun -p 58372")
	got := BeatMe("ab12")
	want := "Beat me: ssh play@game.gotype.fun -p 58372 · room AB12"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDemoSSHCommand(t *testing.T) {
	t.Setenv("SHARE_SSH_CMD", "ssh play@game.gotype.fun -p 58372")
	got := DemoSSHCommand()
	want := "ssh demo@game.gotype.fun -p 58372"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
