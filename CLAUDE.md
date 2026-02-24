# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build          # Build binary → ./hostodo
make install        # Build + install to /usr/local/bin (uses sudo)
make test           # Run all tests: go test -v ./...
make fmt            # Format code: go fmt + gofmt -s
make lint           # Run golangci-lint
make dev ARGS="list"  # Run in dev mode: go run . <args>
make run ARGS="list"  # Same as dev
```

Run a single test: `go test -v -run TestName ./pkg/api/`

Version info is injected via ldflags at build time (see Makefile `LDFLAGS`). The variables `cmd.Version`, `cmd.Commit`, and `cmd.Date` are set during `make build`.

## Architecture Overview

This is a Go CLI for managing Hostodo VPS instances. It uses **Cobra** for command structure, **Bubble Tea** for interactive TUI views, and **Lipgloss** for terminal styling.

### Command Structure

Commands are flat at the root level (not nested under `instances`):

```
hostodo login / logout / whoami     → aliases for auth subcommands
hostodo list (ls, ps)               → list instances (TUI/JSON/simple/details)
hostodo status <hostname>           → instance details
hostodo start/stop/restart <host>   → power control
hostodo ssh <hostname>              → SSH into instance
hostodo deploy (new, create)        → interactive VPS provisioning wizard
hostodo invoices (bills)            → list invoices
hostodo pay <invoice-id>            → pay an invoice
hostodo keys list/add/remove        → SSH key management
hostodo auth login/logout/whoami/sessions → auth subcommand group
hostodo completion                  → shell completions
```

All instance commands accept **hostnames** (not instance IDs) as the primary identifier. The `pkg/resolver` package resolves hostnames via exact match → prefix match → instance ID fallback.

### Package Layout

- **`cmd/`** — Cobra command definitions. `root.go` registers all commands. Auth subcommands live in `cmd/auth/`. All other commands are top-level files in `cmd/`.
- **`pkg/api/`** — HTTP API client (`client.go`), endpoint methods (`instances.go`, `auth.go`, `deploy.go`, `invoices.go`, `sshkeys.go`, `sessions.go`), and request/response models (`models.go`).
- **`pkg/auth/`** — Token storage (`keychain.go`: OS keychain via go-keyring with AES-encrypted file fallback) and OAuth device flow client (`oauth.go`).
- **`pkg/config/`** — Config file management (`~/.hostodo/config.json`). Stores API URL and device ID only — tokens are in the keychain.
- **`pkg/resolver/`** — Hostname-to-instance resolution with caching. Used by all instance commands and for shell tab completion.
- **`pkg/deploy/`** — Hostname generation (adjective-noun combos with uniqueness checking).
- **`pkg/ui/`** — Bubble Tea table component, Lipgloss styles, and output formatters (JSON/simple/details).
- **`pkg/utils/`** — SSH fingerprint utilities.

### Authentication Flow

Login uses **OAuth device flow** (not email/password): the CLI gets a device code, opens a browser for the user to authorize, then polls for a token. Tokens are stored in the OS keychain (macOS Keychain, Linux Secret Service) with an encrypted file fallback. The config file (`~/.hostodo/config.json`) only stores `api_url` and `device_id`.

### Key Patterns

- **Authenticated commands**: Load config → check `auth.IsAuthenticated()` → create `api.NewClient(cfg)` → use client methods.
- **Instance resolution**: All instance commands use `resolver.ResolveInstance(client, hostname)` which returns a `ResolveResult` with the matched instance and match type (exact/prefix/id).
- **Output modes**: List-style commands support `--json`, `--simple`, `--details` flags alongside the default interactive TUI.
- **Deploy command**: Interactive wizard using `survey` prompts (OS → region → plan → hostname → SSH key → quote → confirm). Supports `--json` mode with required flags for scripting.
- **API base URL**: Defaults to `https://api.hostodo.com`, overridable via `HOSTODO_API_URL` env var or `--api-url` flag.
- **Shell completions**: Hostname tab-completion uses a 3-second TTL cache in `pkg/resolver`.
