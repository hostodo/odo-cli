# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build          # Build binary → ./odo
make install        # Build + install to /usr/local/bin (uses sudo)
make test           # Run all tests: go test -v ./...
make fmt            # Format code: go fmt + gofmt -s
make lint           # Run golangci-lint
make dev ARGS="instances list"  # Run in dev mode: go run . <args>
```

Run a single test: `go test -v -run TestName ./pkg/api/`

Version info is injected via ldflags at build time (see Makefile `LDFLAGS`). The variables `cmd.Version`, `cmd.Commit`, and `cmd.Date` are set during `make build`.

## Architecture Overview

This is a Go CLI for managing Hostodo VPS instances. Binary name is `odo`. Uses **Cobra** for command structure, **Bubble Tea** for interactive TUI views, and **Lipgloss** for terminal styling.

### Command Structure

Instance commands live under `odo instances` (aliases: `i`, `ins`). Flat shortcuts are registered as hidden root-level commands for backward compat.

```
odo login / logout / whoami         → aliases for auth subcommands
odo instances (default: list)       → list instances (TUI/JSON/simple/details)
odo instances list (ls, ps)
odo instances status <hostname>     → instance details
odo instances start/stop/restart <host>
odo instances ssh <hostname>        → SSH into instance
odo instances deploy (new, create)  → interactive VPS provisioning wizard
odo instances rename <host> <new>
odo instances reinstall <host>      → stub (not yet implemented)
odo invoices (bills)                → list invoices
odo pay <invoice-id>                → pay an invoice
odo keys list/add/remove            → SSH key management
odo auth login/logout/whoami/sessions
odo completion                      → shell completions

# Flat shortcuts (hidden, same as instances subcommands):
odo list / odo ssh / odo deploy / odo start / odo stop / etc.
```

All instance commands accept **hostnames** as the primary identifier. The `pkg/resolver` package resolves hostnames via exact match → prefix match → instance ID fallback.

### Package Layout

- **`cmd/`** — Cobra command definitions. `root.go` registers all commands. Auth subcommands in `cmd/auth/`. Instance subcommands in `cmd/instances/`.
- **`pkg/api/`** — HTTP API client (`client.go`), endpoint methods, and request/response models (`models.go`).
- **`pkg/auth/`** — Token storage (`keychain.go`: OS keychain via go-keyring with AES-encrypted file fallback at `~/.odo/token.enc`) and OAuth device flow client (`oauth.go`).
- **`pkg/config/`** — Config file management (`~/.odo/config.json`). Stores API URL and device ID only. Migrates from `~/.hostodo/` on first run.
- **`pkg/resolver/`** — Hostname-to-instance resolution with caching. Used by all instance commands and for shell tab completion.
- **`pkg/deploy/`** — Hostname generation (adjective-noun combos with uniqueness checking).
- **`pkg/ui/`** — Bubble Tea table component, Lipgloss styles, and output formatters (JSON/simple/details).
- **`pkg/utils/`** — SSH fingerprint utilities.

### Authentication Flow

Login uses **OAuth device flow**: the CLI gets a device code, opens a browser for the user to authorize, then polls for a token. Tokens are stored in the OS keychain (service name: `odo-cli`) with an encrypted file fallback. Falls back to `hostodo-cli` keychain entry for existing users. The config file only stores `api_url` and `device_id`.

### Key Patterns

- **Authenticated commands**: Load config → check `auth.IsAuthenticated()` → create `api.NewClient(cfg)` → use client methods.
- **Instance resolution**: All instance commands use `resolver.ResolveInstance(client, hostname)`.
- **Output modes**: List-style commands support `--json`, `--simple`, `--details` alongside the default interactive TUI.
- **API base URL**: Defaults to `https://api.hostodo.com`, overridable via `HOSTODO_API_URL` env var or `--api-url` flag.
- **Shell completions**: Hostname tab-completion uses a 3-second TTL cache in `pkg/resolver`.
