# wails-launcher

A dev service manager for the Oasis stack: it starts, stops and restarts the backend
services and the frontend, keeps their logs, and exposes a small HTTP control API on
`127.0.0.1:9901`.

## Two shells, one core

All the actual behaviour lives in `pkg/launcher` — service definitions, process
supervision, log capture and the HTTP API. Two shells wrap it:

| Shell | Build | What it is |
|---|---|---|
| GUI | `wails build` (or `wails dev`) | The desktop app. `main.go` embeds `*launcher.App` and supplies the two things only a window can do: forwarding service events to the frontend, and opening a file dialog. |
| Headless | `go build -o bin/launcherd ./cmd/launcherd` | `cmd/launcherd` — no window, no Dock icon, no tray. It serves the same control API and exits cleanly on SIGINT/SIGTERM. `--start-all` brings every service up at boot. |

The core never imports the Wails runtime. It takes an `Emitter` and a `FilePicker`
through `launcher.NewAppWith`; the headless shell passes neither, so `Browse` says
there is no file dialog instead of hanging.

`main.go` keeps the bound type named `main.App` on purpose — the generated frontend
bindings in `frontend/wailsjs/go/main/App` are written against that name.

## Which one runs

`oasis service …` drives the headless daemon: it builds `bin/launcherd` if it is
missing, starts it detached, and talks to the same `127.0.0.1:9901` API. Nothing
appears on screen. The GUI app is for hands-on use.

`bin/` is gitignored and deliberately not `build/bin`, which is `//go:embed`-ed into
the GUI binary.

## Development

- `wails dev` — GUI with frontend hot reload.
- `go build ./... && go vet ./... && go test ./...` — the core and both shells.
