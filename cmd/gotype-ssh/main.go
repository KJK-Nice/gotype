package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"

	"github.com/kjkusap/monkeytype-clone/internal/invite"
	"github.com/kjkusap/monkeytype-clone/internal/multi"
	"github.com/kjkusap/monkeytype-clone/internal/ui"
)

// sharedApp wires Redis-backed services once per process (stateless replicas).
var sharedApp *ui.App

func main() {
	sshPort := os.Getenv("SSH_PORT")
	if sshPort == "" {
		sshPort = "2222"
	}
	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	hostKey, fp, err := hostKeyOption()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gotype-ssh: host key: %v\n", err)
		os.Exit(1)
	}
	if envFP := invite.HostFingerprint(); envFP != "" {
		fp = envFP
	}

	hub, err := multi.NewHubFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gotype-ssh: hub: %v\n", err)
		os.Exit(1)
	}
	app, err := ui.OpenApp("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "gotype-ssh: data store: %v\n", err)
		os.Exit(1)
	}
	sharedApp = app

	sshAddr := net.JoinHostPort("0.0.0.0", sshPort)
	s, err := wish.NewServer(
		wish.WithAddress(sshAddr),
		hostKey,
		wish.WithPasswordAuth(func(ssh.Context, string) bool { return true }),
		wish.WithMiddleware(
			// ProgramHandler so we can clamp 0×0 PTY sizes — bubbletea v2's
			// cell renderer paints nothing when width/height are zero.
			bubbletea.MiddlewareWithProgramHandler(makeTeaProgram(hub)),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gotype-ssh: %v\n", err)
		os.Exit(1)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Starting SSH server", "addr", sshAddr, "share", invite.SSHCommand())
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start SSH server", "error", err)
			done <- nil
		}
	}()

	httpAddr := net.JoinHostPort("0.0.0.0", httpPort)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		landingHandler(w, r, fp)
	})
	mux.HandleFunc("/demo", func(w http.ResponseWriter, r *http.Request) {
		demoHandler(w, r, fp)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	httpSrv := &http.Server{Addr: httpAddr, Handler: mux}
	log.Info("Starting HTTP landing", "addr", httpAddr)
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("Could not start HTTP server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping servers")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop SSH server", "error", err)
		os.Exit(1)
	}
}

func landingHandler(w http.ResponseWriter, r *http.Request, fingerprint string) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writePlayPage(w, pageContent{
		Title:       "gotype",
		Lede:        "Typing races in your terminal. Multiplayer. Roast or stoic. Over SSH.",
		SSHLine:     invite.SSHCommand(),
		Hint:        "Any username · empty password · Enter",
		Extra:       `<p class="hint"><a href="/demo">spectate / demo</a> · film without friends</p>`,
		Fingerprint: fingerprint,
	})
}

func demoHandler(w http.ResponseWriter, r *http.Request, fingerprint string) {
	writePlayPage(w, pageContent{
		Title:       "gotype demo",
		Lede:        "Spectate a live race. Always something to film.",
		SSHLine:     invite.DemoSSHCommand(),
		Hint:        "Username demo · empty password · Enter",
		Extra:       `<p class="hint"><a href="/">play instead</a> · share a room code from the podium</p>`,
		Fingerprint: fingerprint,
		ShowRace:    true,
	})
}

type pageContent struct {
	Title       string
	Lede        string
	SSHLine     string
	Hint        string
	Extra       string
	Fingerprint string
	ShowRace    bool
}

func writePlayPage(w http.ResponseWriter, p pageContent) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")

	fpLine := ""
	if p.Fingerprint != "" {
		fpLine = fmt.Sprintf(`<p class="hint">host key · %s</p>`, htmlEscape(p.Fingerprint))
	}
	raceBlock := ""
	if p.ShowRace {
		raceBlock = `
  <pre class="race" aria-hidden="true"><span class="ok">the</span> <span class="ok">quick</span> <span class="cur">b</span><span class="dim">rown fox jumps</span>
neon   92 wpm ████████░░
pixel  78 wpm ██████░░░░</pre>`
	}

	_, _ = fmt.Fprintf(w, `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<meta name="description" content="gotype — typing races in your terminal over SSH"/>
<title>%s</title>
<link rel="icon" href="data:image/svg+xml,%%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'%%3E%%3Crect width='32' height='32' rx='6' fill='%%23141210'%%2F%%3E%%3Ctext x='16' y='23' text-anchor='middle' font-size='20' font-family='IBM%%20Plex%%20Mono%%2Cui-monospace%%2Cmonospace' font-weight='700' fill='%%23e2b714'%%3E%%F0%%9D%%90%%A0%%3C%%2Ftext%%3E%%3C%%2Fsvg%%3E"/>
<link rel="preconnect" href="https://fonts.googleapis.com"/>
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin/>
<link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;600;700&display=swap" rel="stylesheet"/>
<style>
  :root {
    --bg: #141210;
    --fg: #f0e6d2;
    --mute: #8a8070;
    --acc: #e2b714;
    --line: #2c2820;
    --panel: #1c1914;
    --ok: #9a8b4a;
  }
  * { box-sizing: border-box; }
  html, body { height: 100%%; }
  body {
    margin: 0;
    min-height: 100vh;
    display: grid;
    place-items: center;
    font-family: "IBM Plex Mono", ui-monospace, monospace;
    color: var(--fg);
    background:
      radial-gradient(900px 480px at 15%% -10%%, rgba(226,183,20,.16), transparent 55%%),
      radial-gradient(700px 420px at 90%% 110%%, rgba(226,183,20,.08), transparent 50%%),
      linear-gradient(180deg, #1a1712 0%%, var(--bg) 55%%, #100e0c 100%%);
    padding: 2rem 1.25rem;
  }
  main {
    width: min(34rem, 100%%);
    animation: rise .7s ease-out both;
  }
  @keyframes rise {
    from { opacity: 0; transform: translateY(12px); }
    to { opacity: 1; transform: none; }
  }
  @keyframes blink {
    50%% { opacity: 0; }
  }
  @keyframes bar {
    from { clip-path: inset(0 60%% 0 0); }
    to { clip-path: inset(0 0 0 0); }
  }
  .brand {
    font-size: clamp(2.6rem, 11vw, 4.75rem);
    font-weight: 700;
    letter-spacing: -.06em;
    line-height: .95;
    margin: 0 0 .85rem;
    color: var(--acc);
  }
  .brand .cursor {
    display: inline-block;
    width: .55ch;
    height: .9em;
    margin-left: .08em;
    vertical-align: -0.08em;
    background: var(--acc);
    animation: blink 1.1s step-end infinite;
  }
  .lede {
    margin: 0 0 1.75rem;
    color: var(--mute);
    font-size: 1.05rem;
    line-height: 1.55;
    max-width: 32ch;
    animation: rise .7s .08s ease-out both;
  }
  .cmd {
    display: flex;
    align-items: stretch;
    border: 1px solid var(--line);
    background: var(--panel);
    animation: rise .7s .16s ease-out both;
  }
  .cmd code {
    flex: 1;
    padding: 1rem 1.1rem;
    font: inherit;
    font-size: clamp(.78rem, 2.6vw, .95rem);
    color: var(--fg);
    overflow-x: auto;
    white-space: nowrap;
  }
  .cmd button {
    border: 0;
    border-left: 1px solid var(--line);
    background: transparent;
    color: var(--acc);
    font: inherit;
    font-size: .8rem;
    font-weight: 600;
    letter-spacing: .04em;
    text-transform: uppercase;
    padding: 0 1rem;
    cursor: pointer;
    transition: background .15s ease, color .15s ease;
  }
  .cmd button:hover { background: rgba(226,183,20,.12); }
  .cmd button.ok { color: var(--fg); }
  .hint {
    margin: .9rem 0 0;
    color: var(--mute);
    font-size: .82rem;
    line-height: 1.45;
    animation: rise .7s .24s ease-out both;
  }
  .hint a { color: var(--acc); text-decoration: none; }
  .hint a:hover { text-decoration: underline; }
  .race {
    margin: 1.4rem 0 0;
    padding: 1rem 1.1rem;
    border: 1px solid var(--line);
    background: var(--panel);
    color: var(--mute);
    font: inherit;
    font-size: .85rem;
    line-height: 1.55;
    overflow: hidden;
    animation: rise .7s .2s ease-out both, bar 4.5s ease-in-out infinite alternate;
  }
  .race .ok { color: var(--ok); }
  .race .cur {
    color: var(--bg);
    background: var(--acc);
  }
  .race .dim { color: var(--mute); }
</style>
</head>
<body>
<main>
  <h1 class="brand">gotype<span class="cursor" aria-hidden="true"></span></h1>
  <p class="lede">%s</p>
  <div class="cmd">
    <code id="ssh">%s</code>
    <button type="button" id="copy" aria-label="Copy SSH command">copy</button>
  </div>
  <p class="hint">%s</p>
  %s
  %s
  %s
</main>
<script>
(() => {
  const btn = document.getElementById("copy");
  const el = document.getElementById("ssh");
  btn.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText(el.textContent.trim());
      btn.textContent = "copied";
      btn.classList.add("ok");
      setTimeout(() => { btn.textContent = "copy"; btn.classList.remove("ok"); }, 1400);
    } catch (_) {
      const r = document.createRange();
      r.selectNodeContents(el);
      const s = window.getSelection();
      s.removeAllRanges();
      s.addRange(r);
    }
  });
})();
</script>
</body>
</html>
`, htmlEscape(p.Title), htmlEscape(p.Lede), htmlEscape(p.SSHLine), htmlEscape(p.Hint), raceBlock, p.Extra, fpLine)
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func makeTeaProgram(hub *multi.Hub) bubbletea.ProgramHandler {
	return func(sess ssh.Session) *tea.Program {
		name := sess.User()
		if name == "" {
			name = "player"
		}
		auto := strings.EqualFold(name, "demo")
		if auto {
			name = "viewer"
		}
		remote := ""
		if addr := sess.RemoteAddr(); addr != nil {
			remote = addr.String()
		}
		pid := multi.NewPlayerID()
		m := ui.NewWithOptions(ui.Options{
			Hub:          hub,
			PlayerName:   name,
			PlayerID:     pid,
			AutoSpectate: auto,
			App:          sharedApp,
			SessionID:    pid,
			RemoteIP:     remote,
		})

		w, h := 80, 24
		if pty, _, ok := sess.Pty(); ok {
			if pty.Window.Width > 0 {
				w = pty.Window.Width
			}
			if pty.Window.Height > 0 {
				h = pty.Window.Height
			}
		}

		opts := append(bubbletea.MakeOptions(sess),
			tea.WithWindowSize(w, h),
			tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
				switch msg := msg.(type) {
				case tea.SuspendMsg:
					return tea.ResumeMsg{}
				case tea.WindowSizeMsg:
					if msg.Width < 1 {
						msg.Width = 80
					}
					if msg.Height < 1 {
						msg.Height = 24
					}
					return msg
				}
				return msg
			}),
		)
		return tea.NewProgram(m, opts...)
	}
}

func hostKeyOption() (ssh.Option, string, error) {
	if pem := os.Getenv("SSH_HOST_KEY"); pem != "" {
		signer, err := gossh.ParsePrivateKey([]byte(pem))
		if err != nil {
			return nil, "", err
		}
		return wish.WithHostKeyPEM([]byte(pem)), gossh.FingerprintSHA256(signer.PublicKey()), nil
	}
	dir := filepath.Join(os.TempDir(), "gotype-ssh")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, "", err
	}
	path := filepath.Join(dir, "host_ed25519")
	opt := wish.WithHostKeyPath(path)
	// Fingerprint after wish materializes the key on first listen; try read now.
	fp := fingerprintFile(path)
	return opt, fp, nil
}

func fingerprintFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	signer, err := gossh.ParsePrivateKey(b)
	if err != nil {
		return ""
	}
	return gossh.FingerprintSHA256(signer.PublicKey())
}
