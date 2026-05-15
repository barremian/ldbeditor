# LevelDB Editor — Agent Instructions

## Cursor Cloud specific instructions

### Overview

LevelDB Editor is a cross-platform desktop app for viewing/editing LevelDB databases, built with **Go 1.25** (Wails v3) backend and **SvelteKit** frontend. See `README.md` for run/build commands (`task dev`, `task build`).

### Runtime requirements

- **Go 1.25+** — required by `go.mod`. The update script installs it to `/usr/local/go`.
- **Node.js 22** — pre-installed; used for the SvelteKit frontend in `frontend/`.
- **go-task** (`task`) — task runner; installed to `/usr/local/bin/task`.
- **wails3** — Wails v3 CLI installed at `~/go/bin/wails3`. Version must match the vendored dependency (`v3.0.0-alpha.74` currently in `go.mod`).
- **Linux GUI libs** — `libgtk-3-dev` and `libwebkit2gtk-4.1-dev` are required for native build and `task dev`.

### Key commands

| Action | Command |
|---|---|
| Dev mode (full app) | `task dev` |
| Build native binary | `task build` |
| Frontend only dev | `cd frontend && npm run dev` |
| Frontend typecheck | `cd frontend && npm run check` |
| Frontend tests | `cd frontend && npm run test` |
| Frontend format check | `cd frontend && npm run format:check` |
| Go vet | `go vet ./...` (requires `frontend/dist` to exist for embed) |
| Generate bindings | `wails3 generate bindings -f '-buildvcs=false' -clean=true -ts` |

### Gotchas

- `go vet ./...` and `go build` both fail if `frontend/dist/` doesn't exist, because `main.go` uses `//go:embed all:frontend/dist`. Build the frontend first (`cd frontend && npm run build:dev`) or run `task dev` which handles it.
- The `server` build tag (`go build -tags server`) has a pre-existing compile error in the vendored Wails code (`BrowserWindow` missing `AttachModal` method). The native build (default, no server tag) works fine.
- The wails3 CLI version should match the vendored dependency version. Install the correct version with `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.74`.
- Go dependencies are vendored in `vendor/`. No need to run `go mod download`.
- Prettier format check (`npm run format:check`) reports pre-existing formatting issues in 28 files — these are not regressions.
- `task dev` starts three processes: builds the Go binary, runs the Vite frontend dev server on port 9245, and launches the native app window. In headless Cloud Agent VMs, the app window opens but requires X11 (which is available).
