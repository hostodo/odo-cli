# Hostodo CLI
![Version](https://img.shields.io/badge/version-2.1.0-blue)
![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-green)

The official CLI for managing Hostodo VPS instances. The binary is called `odo`.

## 🚀 Quick Start

### Installation

#### Homebrew (macOS/Linux)

```bash
brew install hostodo/tap/odo
```

#### Download Binary

Download pre-built binaries from the [releases page](https://github.com/hostodo/odo-cli/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/hostodo/odo-cli/releases/latest/download/odo-cli_Darwin_all.tar.gz | tar xz
sudo mv odo /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/hostodo/odo-cli/releases/latest/download/odo-cli_Linux_x86_64.tar.gz | tar xz
sudo mv odo /usr/local/bin/
```

#### Package Managers

```bash
# Debian/Ubuntu
wget https://github.com/hostodo/odo-cli/releases/latest/download/odo_linux_amd64.deb
sudo dpkg -i odo_linux_amd64.deb

# RHEL/CentOS/Fedora
wget https://github.com/hostodo/odo-cli/releases/latest/download/odo_linux_amd64.rpm
sudo rpm -i odo_linux_amd64.rpm
```

#### From Source

Requires Go 1.25 or higher:

```bash
git clone https://github.com/hostodo/odo-cli.git
cd odo-cli
make install
```

### Authentication

```bash
odo login
```

Uses OAuth device flow — a browser window opens for you to authorize the CLI. Your access token is stored in the OS keychain (macOS Keychain, Linux Secret Service). On headless systems without a keychain, a plain token file is stored at `~/.odo/token` with `0600` permissions.

### Basic Usage

```bash
# List all instances (interactive TUI)
odo list

# SSH into an instance
odo ssh <hostname>

# Deploy a new instance (interactive wizard)
odo deploy

# Deploy with a promo code
odo deploy --promo LETCLI

# Power control
odo start <hostname>
odo stop <hostname>
odo restart <hostname>
```

## 📖 Commands

### Global Flags

- `--api-url string` - API URL (default: `https://api.hostodo.com` or `$HOSTODO_API_URL`, must be https://)
- `--config string` - Config file path (default: `$HOME/.odo/config.json`)
- `-h, --help` - Show help
- `-v, --version` - Show version

### Authentication

#### `odo login`
Authenticate with your Hostodo account using OAuth device flow.

```bash
odo login
```

#### `odo logout`
Clear stored credentials.

```bash
odo logout
```

#### `odo whoami`
Show the currently authenticated user.

```bash
odo whoami
```

#### `odo auth sessions`
List active CLI sessions.

```bash
odo auth sessions
```

### Capacity Commands

```bash
odo pools list --simple
odo pools show pool::abc --details
odo pools options
odo pools quote --plan-id 42 --billing-cycle annually --promo SAVE
odo pools purchase --plan-id 42
odo pools purchase --plan-id 42 --payment-method saved_card --payment-method-id pm::123 --idempotency-key my-purchase-1 --yes
odo pools update pool::abc --display-name Production --autorenew=true
odo pools update pool::abc --autorenew=false
odo pools cancel pool::abc --reason "No longer needed"
```

A first purchase fetches and displays a fresh quote with the selected payment
method before asking you to type the server's exact confirmation phrase. Saved
cards also require a separate exact charge phrase showing the card, ending, and
amount. Use `--yes` to explicitly accept both applicable phrases in scripts.
Servers that omit the generic confirmation cannot be used for purchases.
The default payment method is hosted Stripe checkout; a validated HTTPS checkout
URL is printed alone on stdout. Quotes, approvals, and idempotency keys go to stderr.
Reuse the same `--idempotency-key` with the same arguments, API origin, and account
when retrying; rotating the bearer token for that account does not invalidate retries.
Confirmed quote snapshots are saved in `~/.odo/capacity-checkouts-v2/` before checkout,
with owner-only permissions on Unix and inherited profile ACLs on Windows.
Cached retries reuse the exact saved quote and confirmations without fetching a
new quote, even if prices, credits, or Capacity have since changed. An explicit
key without a local record is treated as first use, with a stderr notice.
Keep these records when an outcome is unknown; recovery on another machine also
requires the original authentication key. Corrupt records or changed inputs abort
checkout. Records use HMAC-SHA256 with a stable random 32-byte OS keychain secret;
when the keychain is unavailable at initialization, the secret is saved with 0600
permissions in `~/.odo/capacity-retry-key`, outside the cache directory. That file
otherwise records the keychain storage choice. Inaccessible keys fail closed, as
does a missing key marker when the v2 cache directory exists, even if empty. Legacy
unauthenticated records in `~/.odo/capacity-checkouts/` are preserved but never replayed or migrated,
and do not block key initialization for new purchases. Reconcile any legacy checkout
before using a new key. Key initialization serializes with an OS advisory lock that
releases automatically on process exit; stale directories from the old
`capacity-retry-key.lock` lock are ignored and preserved. Generic confirmations are
encrypted at rest; saved-card phrases are reconstructed from the supplied card ID,
safe cached last four digits, and original amount. No plaintext card IDs, provider
secrets, or credentials are cached. Human output strips terminal control sequences;
successful `--json` payloads are preserved byte for byte.
Human-mode retries show the backend phase, order/invoice status, and invoice URL
on stderr, distinguishing an unresolved outcome from a completed checkout with
an unpaid invoice. Stdout remains a validated HTTPS checkout URL or raw JSON.
Selecting another tier can change existing Capacity; review the quote's mode
and recurring amount.

Cancellation also cancels the member instances and requires typing `CANCEL` or
passing `--yes`. All Capacity commands support `--json` to print the raw API
response as formatted JSON, including fields unknown to the CLI. `--json` does
not bypass confirmation. List/show retain `--simple`, `--details`, and the default
interactive table.

### Instance Commands

All instance commands are under `odo instances` (alias: `odo i`). Flat shortcuts like `odo ssh`, `odo list`, `odo start` etc. also work.

Hostnames support exact match, unambiguous prefix match, and instance ID fallback. Tab completion works for hostnames.

#### `odo instances` / `odo list`

List all your VPS instances. Aliases: `ls`, `ps`.

**Flags:**
- `--json` - Output as JSON
- `--simple` - Output as simple ASCII table
- `--details` - Show detailed information
- `--limit int` - Max instances to fetch (default: 100)
- `--offset int` - Pagination offset

```bash
odo list
odo list --json
odo list --simple
odo list --details
```

**Interactive TUI Controls:**
- `↑/↓` or `j/k` — Navigate
- `Enter` — View instance details
- `q` / `Esc` / `Ctrl+C` — Quit

#### `odo instances status <hostname>`

Show detailed information about an instance.

```bash
odo instances status my-server
odo instances status my-server --json
```

#### `odo instances start/stop/restart <hostname>`

Power control.

```bash
odo start my-server
odo stop my-server
odo stop my-server --force   # immediate shutdown
odo restart my-server
```

#### `odo instances ssh <hostname>`

SSH into an instance. Auto-detects the SSH user from the template. Falls back to sshpass if key auth fails and the instance has a default password.

```bash
odo ssh my-server
odo ssh my-server --user ubuntu
odo ssh my-server -- -L 8080:localhost:8080   # extra ssh flags after --
```

#### `odo instances rename <hostname> <new-hostname>`

```bash
odo rename my-server new-name
```

#### `odo instances reinstall <hostname>`

Wipe and reinstall the OS on an instance. **Destructive — all data will be lost.**

```bash
# Interactive
odo reinstall my-server

# Specify OS
odo reinstall my-server --os "Debian 12"

# Specify OS + SSH key
odo reinstall my-server --os "Ubuntu 22.04" --ssh-key mykey
```

#### `odo instances deploy`

Deploy a new Hostodo VPS with interactive prompts or flags. Aliases: `new`, `create`.

**Flags:**
- `--os string` - OS template (skips OS prompt)
- `--region string` - Region: `DET01`, `LV01`, `TPA01`
- `--plan string` - Plan name (e.g. `EPYC-2G1C32GN`)
- `--hostname string` - Custom hostname
- `--ssh-key string` - SSH key name
- `--billing-cycle string` - `monthly`, `annually`, `semiannually`, `biennially`, `triennially`
- `--promo string` - Promo code for a discount
- `-y, --yes` - Skip confirmation
- `--json` - JSON output (requires `--os`, `--region`, `--plan`)

```bash
# Interactive wizard
odo deploy

# With promo code
odo deploy --promo LETCLI

# Fully non-interactive
odo deploy --os "Ubuntu 22.04" --region DET01 --plan EPYC-2G1C32GN --billing-cycle monthly --promo LETCLI --yes

# JSON output
odo deploy --os "Ubuntu 22.04" --region DET01 --plan EPYC-2G1C32GN --json
```

### Billing

#### `odo invoices`

List invoices. Alias: `bills`.

```bash
odo invoices
odo invoices --status=unpaid
```

#### `odo pay <invoice-id>`

Pay an invoice using your default payment method.

```bash
odo pay INV-12345
odo pay INV-12345 --yes
```

### SSH Keys

```bash
odo keys list
odo keys add mykey "ssh-rsa AAAA..."
odo keys add mykey --file ~/.ssh/id_rsa.pub
odo keys remove mykey
```

### Shell Completions

Homebrew installs completions automatically. For manual installs:

```bash
# Zsh
odo completion zsh > "${fpath[1]}/_odo"

# Bash
odo completion bash > ~/.local/share/bash-completion/completions/odo

# Fish
odo completion fish > ~/.config/fish/completions/odo.fish
```

## 🔧 Configuration

Config is stored in `~/.odo/config.json`:

```json
{
  "api_url": "https://api.hostodo.com",
  "device_id": "a1b2c3d4-..."
}
```

Tokens are stored in the OS keychain. On systems without a keychain, a plain token file is used at `~/.odo/token` with `0600` permissions. The config file never contains credentials.

### Environment Variables

- `HOSTODO_API_URL` — Override the API URL (must be `https://`)

## 🏗️ Development

### Building from Source

```bash
git clone https://github.com/hostodo/odo-cli.git
cd odo-cli
make build   # → ./odo
make test
make dev ARGS="instances list"
```

### Project Structure

```
cmd/
├── root.go              # Root command, registers all subcommands
├── invoices.go          # Billing commands
├── pay.go
├── keys.go              # SSH key management
├── completion.go
├── auth/                # Auth subcommands (login, logout, whoami, sessions)
└── instances/           # Instance subcommands
    ├── root.go
    ├── list.go
    ├── status.go
    ├── start.go
    ├── stop.go
    ├── restart.go
    ├── ssh.go
    ├── rename.go
    ├── deploy.go        # Deployment wizard
    └── reinstall.go
pkg/
├── api/                 # HTTP client + endpoint methods + models
├── auth/                # Token storage (OS keychain + file fallback)
├── config/              # Config file (~/.odo/config.json)
├── resolver/            # Hostname resolution + tab completion
├── deploy/              # Hostname generation
├── ui/                  # Bubble Tea TUI, Lipgloss styles, output formatters
└── utils/               # SSH fingerprint utilities
```

## 📝 License

MIT — see LICENSE file for details.

## 🔗 Links

- [Hostodo Website](https://hostodo.com)
- [CLI Documentation](https://hostodo.com/docs/cli)
- [Hostodo Console](https://console.hostodo.com)

## 💬 Support

- Email: support@hostodo.com
- Open an issue: [github.com/hostodo/odo-cli/issues](https://github.com/hostodo/odo-cli/issues)
