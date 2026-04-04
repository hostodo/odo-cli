# Hostodo CLI
![Version](https://img.shields.io/badge/version-2.0.0-blue)
![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-green)

The official CLI for managing Hostodo VPS instances. The binary is called `odo`.

## 🚀 Quick Start

### Installation

#### Homebrew (macOS/Linux)

```bash
brew tap hostodo/tap
brew install hostodo/tap/odo
```

#### Download Binary

Download pre-built binaries from the [releases page](https://github.com/hostodo/cli/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/hostodo/cli/releases/latest/download/odo_Darwin_arm64.tar.gz | tar xz
sudo mv odo /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/hostodo/cli/releases/latest/download/odo_Darwin_x86_64.tar.gz | tar xz
sudo mv odo /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/hostodo/cli/releases/latest/download/odo_Linux_x86_64.tar.gz | tar xz
sudo mv odo /usr/local/bin/
```

#### Package Managers

```bash
# Debian/Ubuntu
wget https://github.com/hostodo/cli/releases/latest/download/odo_Linux_x86_64.deb
sudo dpkg -i odo_Linux_x86_64.deb

# RHEL/CentOS/Fedora
wget https://github.com/hostodo/cli/releases/latest/download/odo_Linux_x86_64.rpm
sudo rpm -i odo_Linux_x86_64.rpm
```

#### From Source

```bash
git clone https://github.com/hostodo/cli.git
cd cli
make install
```

### Authentication

```bash
odo login
```

Uses OAuth device flow — a browser window opens for you to authorize the CLI. Your access token is stored in the OS keychain (macOS Keychain, Linux Secret Service) with an encrypted file fallback.

### Basic Usage

```bash
# List all instances (interactive TUI)
odo instances

# SSH into an instance
odo ssh <hostname>

# Deploy a new instance
odo instances deploy

# Power control
odo instances start <hostname>
odo instances stop <hostname>
odo instances restart <hostname>
```

## 📖 Commands

### Global Flags

- `--api-url string` - API URL (default: https://api.hostodo.com or `$HOSTODO_API_URL`)
- `--config string` - Config file path (default: `$HOME/.odo/config.json`)
- `-h, --help` - Show help
- `-v, --version` - Show version

### Authentication

#### `odo login`
Authenticate with your Hostodo account using OAuth device flow.

```bash
odo login
odo login --api-url https://custom-api.example.com
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
odo instances
odo instances --json
odo instances --simple
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
odo instances start my-server
odo instances stop my-server
odo instances stop my-server --force   # immediate shutdown
odo instances restart my-server
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
odo instances rename my-server new-name
```

#### `odo instances deploy`

Deploy a new Hostodo VPS with interactive prompts or flags. Aliases: `new`, `create`.

**Flags:**
- `--os string` - OS template (skips OS prompt)
- `--region string` - Region (skips region prompt)
- `--plan string` - Plan name (skips plan prompt)
- `--hostname string` - Custom hostname
- `--ssh-key string` - SSH key name
- `--billing-cycle string` - `monthly`, `annually`, etc.
- `-y, --yes` - Skip confirmation
- `--json` - JSON output (requires `--os`, `--region`, `--plan`)

```bash
# Interactive
odo instances deploy

# Non-interactive
odo deploy --os "Ubuntu 25.04" --region DET01 --plan EPYC-2G1C32GN --yes

# JSON output
odo deploy --os "Ubuntu 25.04" --region DET01 --plan EPYC-2G1C32GN --json
```

#### `odo instances reinstall <hostname>`

Reinstall the OS on an instance. *(Coming soon)*

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

Tokens are stored in the OS keychain, not in the config file. Encrypted file fallback at `~/.odo/token.enc`.

### Environment Variables

- `HOSTODO_API_URL` — Override the API URL

```bash
export HOSTODO_API_URL=http://localdev.hostodo.com:8000
odo login
```

## 🏗️ Development

### Building from Source

```bash
git clone https://github.com/hostodo/cli.git
cd cli
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
    ├── root.go          # instances parent command
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
├── auth/                # Token storage (keychain + encrypted fallback)
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
- [Hostodo Console](https://console.hostodo.com)
- [API Documentation](https://console.hostodo.com/api/docs)

## 💬 Support

- Email: support@hostodo.com
- Documentation: [docs.hostodo.com](https://docs.hostodo.com)
