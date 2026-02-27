# ldbeditor – Agent Guide

This document helps AI agents understand and work effectively in the **ldbeditor** project.

## Project Overview

**ldbeditor** is a cross-platform LevelDB editor built with [Wails v3](https://v3.wails.io/). It uses Go for the backend and a SvelteKit frontend. The app runs as a native GUI (webview) on macOS, with support for server mode (no GUI, HTTP only) and cross-platform builds via Docker.

## Tech Stack

| Layer     | Technology                                                 |
| --------- | ---------------------------------------------------------- |
| Framework | Wails v3 (alpha)                                           |
| Backend   | Go 1.25                                                    |
| Frontend  | SvelteKit, Svelte 4, Vite 5                                |
| Build     | Task (go-task)                                             |
| Styling   | Puppertino CSS (optional, in `frontend/public/puppertino`) |

## Project Structure

```
ldbeditor/
├── main.go              # App entry point, window setup, event emission
├── greetservice.go      # Example Go service (GreetService) bound to frontend
├── go.mod / go.sum
├── frontend/
│   ├── src/             # SvelteKit source
│   │   ├── routes/      # Routes (e.g. +page.svelte)
│   │   └── app.html    # Root HTML
│   ├── bindings/        # Generated TypeScript bindings from Go services
│   ├── public/          # Static assets, Puppertino CSS
│   ├── dist/            # Build output (embedded into Go binary)
│   └── package.json
├── build/
│   ├── config.yml       # Wails config (info, dev_mode, fileAssociations)
│   ├── Taskfile.yml     # Common tasks (frontend build, bindings, icons)
│   ├── darwin/          # macOS-specific build (Taskfile, Info.plist, icons)
│   ├── windows/
│   ├── linux/
│   ├── ios/
│   └── android/
├── bin/                 # Built binaries (.app on macOS)
└── Taskfile.yml         # Root task orchestrator
```

## Key Files

- **`main.go`** – Application setup: services, assets, window options, event emission.
- **`greetservice.go`** – Example service; add more Go services here and register them in `main.go` with `application.NewService(&YourService{})`.
- **`build/config.yml`** – Product info, dev mode, file associations. Run `wails3 task common:update:build-assets` after changes.
- **`frontend/bindings/`** – **Generated** by Wails from Go services. Don’t edit manually; use `wails3 generate bindings`.

## Commands

| Task            | Command                                 | Description                                    |
| --------------- | --------------------------------------- | ---------------------------------------------- |
| Dev mode        | `task dev`                              | Hot-reload dev; uses `wails3 dev` + Vite + run |
| Build           | `task build`                            | Build app for current OS                       |
| Package         | `task package`                          | Create packaged `.app` (macOS)                 |
| Run             | `task run`                              | Run built binary                               |
| Server mode     | `task build:server` / `task run:server` | Build/run HTTP-only, no GUI                    |
| Docker server   | `task build:docker` / `task run:docker` | Docker image for server mode                   |
| Cross-compile   | `task setup:docker` then `task build`   | Use Docker to build for other platforms        |
| macOS universal | `task darwin:build:universal`           | ARM64 + AMD64 universal binary                 |

## Conventions for AI Agents

1. **Go services**
   - Add new services as Go structs with exported methods.
   - Register them in `main.go` via `application.NewService(&YourService{})`.
   - Run `wails3 generate bindings -ts` (or use `task dev`) to regenerate frontend bindings.

2. **Frontend bindings**
   - Import bindings from `frontend/bindings/changeme` (or the module name from `go.mod`).
   - Use `@wailsio/runtime` for events: `Events.On('eventName', callback)`.

3. **Frontend build**
   - SvelteKit with `adapter-static`; output goes to `frontend/dist`.
   - `frontend/dist` is embedded in the Go binary via `//go:embed`.

4. **Module name**
   - `go.mod` uses `module changeme`. Update it and `build/config.yml` when renaming the project; bindings path will follow.

5. **Paths**
   - `build/` holds platform-specific Taskfiles and config.
   - `bin/` is ignored in `.gitignore` and contains built executables.

## Dependencies

- **Go**: `go mod tidy` for dependency management.
- **Frontend**: `npm install` in `frontend/` (or via `task dev`).
- **Task**: [go-task](https://taskfile.dev/) for running tasks.

## Events

The app registers a `time` event (string) in `main.go` and emits it every second. Frontend can listen with:

```ts
import { Events } from "@wailsio/runtime";
Events.On("time", (data) => {
  /* data.data is the string */
});
```

To add events: `application.RegisterEvent<YourType>("eventName")` and `app.Event.Emit("eventName", value)`.
