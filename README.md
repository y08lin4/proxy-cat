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

Phase 0 is frozen in `docs/`. Phase 1 has a code-level MVP for the Go backend, Wails shell, and React control panel.

## Development

Prerequisites:

- Go 1.22+
- Node.js
- pnpm
- Wails v2 CLI
- `mihomo.exe` available on `PATH` or beside the app runtime

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
