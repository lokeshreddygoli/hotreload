# hotreload

I got tired of hitting `Ctrl+C`, `go build`, `./server` fifty times a day. So I built this.

`hotreload` watches your Go project, rebuilds on save, and restarts the server — all within about a second.

```bash
hotreload --root ./myproject --build "go build -o ./bin/server ./cmd/server" --exec "./bin/server"
```
Project Website: https://lokeshreddygoli.github.io/hotreload/
---

## Install

```bash
git clone https://github.com/lokeshreddygoli/hotreload
cd hotreload
make build
# binary lands at ./bin/hotreload
```

Or install directly:

```bash
go install github.com/lokeshreddygoli/hotreload@latest
```

---

## Usage

```
hotreload --root <dir> --build "<build cmd>" --exec "<run cmd>"
```

| Flag | What it does |
|---|---|
| `--root` | Directory to watch (all subdirs included automatically) |
| `--build` | Command to build the project |
| `--exec` | Command to start the server after a successful build |

**Linux / macOS:**
```bash
hotreload \
  --root ./service \
  --build "go build -o ./bin/api ./service/cmd/api" \
  --exec "./bin/api --port 8080"
```

**Windows (PowerShell):**
```powershell
hotreload `
  --root .\service `
  --build "go build -o .\bin\api.exe .\service\cmd\api" `
  --exec ".\bin\api.exe"
```

---

## How it works

A few things I had to get right that aren't obvious:

**Editors don't save cleanly.**
Vim writes a temp file and renames it. VS Code sometimes fires 3–4 events per save. A naive watcher rebuilds 4 times. I debounce events with a 300ms quiet window — the rebuild only fires once things settle.

**Rapid changes during a slow build.**
If you save while a build is already running, the old build is cancelled immediately. Only the latest state gets built. Each build cycle runs under its own `context.Context` that gets cancelled on the next trigger.

**Process tree cleanup.**
On Linux/macOS, the server is launched with `Setpgid=true`. On kill, I send the signal to `-pgid` which takes out the entire process tree — not just the parent. On Windows, I use a Job Object with `KILL_ON_JOB_CLOSE`, falling back to `taskkill /F /T` if needed.

**Crash loops.**
If the server exits within 2s of starting three times in a row, the tool backs off for 5 seconds before retrying. Saves you from a spinning CPU while you fix a boot-time panic.

**New directories at runtime.**
Create a new package folder while the tool is running — it starts watching it immediately. No restart needed.

---

## What gets ignored

- `.git/`, `vendor/`, `node_modules/`
- Editor swap files (`.swp`, `.swo`, `~`, `#file#`)
- Hidden files and directories
- Build artifacts (`.o`, `.a`, `.so`, `.exe`, `.test`)

Only `.go`, `go.mod`, and `go.sum` trigger a rebuild. Configurable in `internal/filter/filter.go`.

---

## Demo

```bash
make demo
```

Starts `hotreload` watching `testserver/` — a minimal HTTP server at `http://localhost:8080`. Open `testserver/main.go`, change the `greeting` constant, save. Watch the terminal.

---

## Tests

```bash
make test
make test-race
```

---

## Project layout

```
hotreload/
├── cmd/root.go                   CLI flags (cobra)
├── internal/
│   ├── engine/engine.go          main loop: watch → debounce → build → run
│   ├── watcher/watcher.go        fsnotify wrapper with dynamic dir watching
│   ├── debounce/debounce.go      300ms quiet-window debouncer
│   ├── filter/filter.go          what to watch, what to ignore
│   └── process/
│       ├── process.go            shared: Start, Kill, Run, Wait
│       ├── process_unix.go       Linux/macOS: Setpgid + SIGTERM/SIGKILL
│       └── process_windows.go    Windows: Job Objects + taskkill fallback
└── testserver/                   demo HTTP server
```

---

## Platform support

Works on Linux, macOS, and Windows. Process management is platform-specific (build tags) but the interface is identical everywhere.

---

## License

MIT — see [LICENSE](LICENSE)
