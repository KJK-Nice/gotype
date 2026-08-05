#!/usr/bin/env python3
"""Run gotype in a PTY and auto-tour the new UI for asciinema."""
import os
import pty
import select
import sys
import time

KEYS = [
    ("c", 1.0),            # claim overlay
    ("\r", 1.0),           # register (list)
    ("demoplayer", 0.4),
    ("\r", 2.5),           # create — wait for claim code screen
    ("\r", 1.2),           # dismiss code
    ("i", 1.6),            # inventory tab
    ("s", 1.6),            # shop
    ("p", 1.8),            # season pass
    ("e", 1.6),            # equip
    ("\x1b", 1.0),        # esc — close progression
    ("m", 1.6),            # multiplayer menu
    ("\x1b", 1.0),
    ("u", 1.0),            # theme
    ("q", 0.6),
]


def relay(master_fd: int) -> None:
    """Copy PTY output to stdout so asciinema records the TUI."""
    while True:
        r, _, _ = select.select([master_fd], [], [], 0.1)
        if not r:
            continue
        try:
            data = os.read(master_fd, 4096)
        except OSError:
            break
        if not data:
            break
        os.write(sys.stdout.fileno(), data)


def main() -> int:
    os.chdir("/workspace")
    os.environ["GOTYPE_DATA_DIR"] = os.environ.get("GOTYPE_DATA_DIR", "/tmp/gotype-demo-cast")
    os.environ.setdefault("TERM", "xterm-256color")

    pid, master_fd = pty.fork()
    if pid == 0:
        os.execvp("go", ["go", "run", "./cmd/gotype"])

    import threading

    t = threading.Thread(target=relay, args=(master_fd,), daemon=True)
    t.start()

    time.sleep(2.5)
    for key, delay in KEYS:
        if len(key) > 1 and key not in ("\r", "\x1b"):
            for ch in key:
                os.write(master_fd, ch.encode("latin-1"))
                time.sleep(0.05)
        else:
            os.write(master_fd, key.encode("latin-1") if isinstance(key, str) else key)
        time.sleep(delay)

    try:
        os.waitpid(pid, 0)
    except ChildProcessError:
        pass
    time.sleep(0.3)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
