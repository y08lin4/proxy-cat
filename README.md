# Proxy-Cat

Proxy-Cat is a Windows desktop proxy client based on Mihomo, built with Wails, React, and Go.

The project goal is simple: provide a Clash-style client with more stable automatic node selection and fewer manual switches.

## Phases

- Phase 0: product and architecture design freeze
- Phase 1: runnable MVP core
- Phase 2: stable-first automatic node selection
- Phase 3: UI and user experience polish
- Phase 4: reliability and convergence

## Current Status

Phase 0 is frozen in `docs/`. Phases 1-4 have a code-level MVP for the Go backend, Wails shell, React control panel, auto-stable selection, and reliability controls.

Phase 4 covers Mihomo unexpected-exit tracking, lightweight recovery on status refresh, an explicit recovery entry point, auto-stable tick cooldown, and bounded in-memory logs. It is not a full end-to-end proxy acceptance until Wails and Mihomo are available locally.

## Development

Prerequisites:

- Go 1.22+
- Node.js
- pnpm
- Wails v2 CLI
- `mihomo.exe` available on `PATH` or beside the app runtime

The app starts Mihomo as an external process with:

```bash
mihomo.exe -f <app-data>/profiles/active/config.yaml -d <app-data>/mihomo
```

Without both the Wails CLI and a usable `mihomo.exe`, local verification is limited to unit tests and frontend builds. Real desktop smoke testing still needs the Wails shell, and real proxy acceptance still needs Mihomo running with a valid subscription/profile.

Common checks:

```bash
go test ./...
cd frontend
pnpm install --ignore-scripts
pnpm run build
```

Run with Wails:

```bash
wails dev
```

Manual smoke checklist once the required CLIs are available:

1. Start the app with `wails dev`.
2. Load a valid subscription and confirm the active Mihomo config is generated.
3. Start the Mihomo core and confirm connection status/logs update.
4. Toggle the Windows system proxy only when you are ready for local proxy changes.
5. Run one auto-stable tick, then immediately run another and confirm the cooldown path is reported.
6. If Mihomo exits unexpectedly, refresh status or call the recovery path and confirm the core restarts from the last launch config.
