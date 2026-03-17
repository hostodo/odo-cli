# Hostodo CLI [WIP]
![Version](https://img.shields.io/badge/version-0.1.0-blue)
![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-green)

## 🚀 Quick Start

### Installation

#### Homebrew (macOS/Linux)

```bash
# Install from Homebrew tap (recommended)
brew tap hostodo/tap
brew install hostodo

# Or install directly from the formula
brew install hostodo/tap/hostodo
```

#### Download Binary

Download pre-built binaries from the [releases page](https://github.com/hostodo/hostodo-cli/releases):

Available for:
- macOS (Intel & Apple Silicon)
- Linux (amd64, arm64)
- Windows (amd64)

```bash
# macOS example (ARM64)
curl -L https://github.com/hostodo/hostodo-cli/releases/latest/download/hostodo_Darwin_arm64.tar.gz | tar xz
sudo mv hostodo /usr/local/bin/

# Linux example (amd64)
curl -L https://github.com/hostodo/hostodo-cli/releases/latest/download/hostodo_Linux_x86_64.tar.gz | tar xz
sudo mv hostodo /usr/local/bin/
```

#### From Source

```bash
# Clone the repository
git clone https://github.com/hostodo/hostodo-cli.git
cd hostodo-cli

# Build and install
make install

# Or build manually
go build -o hostodo .
sudo mv hostodo /usr/local/bin/
```

#### Package Managers

```bash
# Debian/Ubuntu
wget https://github.com/hostodo/hostodo-cli/releases/latest/download/hostodo_Linux_x86_64.deb
sudo dpkg -i hostodo_Linux_x86_64.deb

# RHEL/CentOS/Fedora
wget https://github.com/hostodo/hostodo-cli/releases/latest/download/hostodo_Linux_x86_64.rpm
sudo rpm -i hostodo_Linux_x86_64.rpm
```

### Authentication

Before using the CLI, authenticate with your Hostodo account:

```bash
hostodo login
```

This uses OAuth device flow: a browser window will open where you authorize the CLI. Your access token is securely stored in the OS keychain (macOS Keychain, Linux Secret Service) with an encrypted file fallback.

**Options:**
- `--api-url`: Use a custom API URL (default: https://api.hostodo.com)

### Basic Usage

```bash
# List all instances (interactive TUI)
hostodo list

# List instances as JSON
hostodo list --json

# List instances as simple table
hostodo list --simple

# View detailed information about a specific instance
hostodo status <hostname>

# Power control (by hostname)
hostodo start <hostname>
hostodo stop <hostname>
hostodo restart <hostname>

# SSH into an instance
hostodo ssh <hostname>

# Deploy a new instance
hostodo deploy

# Logout
hostodo logout
```

## 📖 Commands

### Global Flags

All commands support these global flags:

- `--api-url string` - API URL (default: https://api.hostodo.com or $HOSTODO_API_URL)
- `--config string` - Config file path (default: $HOME/.hostodo/config.json)
- `-h, --help` - Show help
- `-v, --version` - Show version

### Authentication Commands

#### `hostodo login`

Authenticate with your Hostodo account using OAuth device flow. Opens a browser for authorization.

**Examples:**
```bash
hostodo login
hostodo login --api-url https://custom-api.example.com
```

#### `hostodo logout`

Clear stored credentials and logout.

**Example:**
```bash
hostodo logout
```

#### `hostodo whoami`

Display information about the currently authenticated user.

**Example:**
```bash
hostodo whoami
```

#### `hostodo auth sessions`

List your active CLI sessions.

**Example:**
```bash
hostodo auth sessions
```

### Instance Commands

All instance commands accept **hostnames** as the primary identifier. Hostname resolution supports exact match, unambiguous prefix match, and instance ID fallback. Tab completion is available for hostnames.

#### `hostodo list`

List all your VPS instances with various output formats. Aliases: `ls`, `ps`.

**Flags:**
- `--json` - Output as JSON
- `--simple` - Output as simple ASCII table
- `--details` - Show detailed information
- `--limit int` - Maximum number of instances to fetch (default: 100)
- `--offset int` - Offset for pagination (default: 0)

**Examples:**
```bash
# Interactive TUI (default)
hostodo list

# JSON output for scripting
hostodo list --json

# Simple table for quick viewing
hostodo list --simple

# Detailed view with all information
hostodo list --details

# Fetch 50 instances
hostodo list --limit 50
```

**Interactive TUI Controls:**
- `↑/↓` or `j/k` - Navigate through instances
- `Enter` - View detailed information about selected instance
- `q` or `Ctrl+C` or `Esc` - Quit

#### `hostodo status <hostname>`

Get detailed information about a specific instance.

**Flags:**
- `--json` - Output as JSON

**Examples:**
```bash
hostodo status my-server
hostodo status my-server --json
```

**Information Displayed:**
- Basic information (ID, hostname, status, power state)
- Network configuration (IPs, MAC address)
- Resource allocation (RAM, CPU, Disk, Bandwidth usage)
- Plan and template details
- Billing information (amount, cycle, next due date)
- Timeline (created, updated timestamps)

#### `hostodo start <hostname>`

Start a stopped VPS instance.

**Examples:**
```bash
hostodo start my-server
```

The command will:
1. Send the start command to the instance
2. Wait for the instance to boot (up to 30 seconds)
3. Display status updates

#### `hostodo stop <hostname>`

Stop a running VPS instance.

**Flags:**
- `-f, --force` - Force immediate shutdown (without graceful shutdown)

**Examples:**
```bash
# Graceful shutdown
hostodo stop my-server

# Force shutdown
hostodo stop my-server --force
```

The command will:
1. Send the stop command to the instance
2. Wait for the instance to shutdown (up to 60 seconds)
3. Display status updates

#### `hostodo restart <hostname>`

Restart a VPS instance.

**Flags:**
- `-f, --force` - Force immediate restart

**Examples:**
```bash
# Graceful restart
hostodo restart my-server

# Force restart
hostodo restart my-server --force
```

The command will:
1. Send the restart command to the instance
2. Wait for the instance to restart (up to 90 seconds)
3. Display status updates

#### `hostodo ssh <hostname>`

Connect to an instance via SSH using the system ssh binary. Auto-detects the SSH user from the instance template. If key-based auth fails and the instance has a default password, automatically retries with sshpass.

**Flags:**
- `-u, --user string` - SSH user (default: auto-detected from template)

**Examples:**
```bash
hostodo ssh my-server
hostodo ssh my-server --user ubuntu
```

### Deployment

#### `hostodo deploy`

Deploy a new VPS instance with interactive prompts or flags. Aliases: `new`, `create`.

**Flags:**
- `--os string` - OS template name (skips OS prompt)
- `--region string` - Region name (skips region prompt)
- `--plan string` - Plan name (skips plan prompt)
- `--hostname string` - Custom hostname (skips auto-generation)
- `--ssh-key string` - SSH key name to use for authentication
- `-y, --yes` - Skip confirmation prompt
- `--json` - JSON output mode (requires --os, --region, --plan)

**Examples:**
```bash
# Interactive mode (guided prompts)
hostodo deploy

# Skip prompts with flags
hostodo deploy --os "Ubuntu 22.04" --region "Los Angeles" --plan KVM-2G

# Custom hostname
hostodo deploy --hostname my-server

# Skip confirmation
hostodo deploy --yes

# JSON output (requires all selection flags)
hostodo deploy --os "Ubuntu 22.04" --region "Los Angeles" --plan KVM-2G --json
```

### Billing Commands

#### `hostodo invoices`

List your invoices with optional filtering. Alias: `bills`.

**Flags:**
- `--status string` - Filter by status (e.g., `unpaid`)

**Examples:**
```bash
hostodo invoices
hostodo invoices --status=unpaid
```

#### `hostodo pay <invoice-id>`

Pay an invoice.

**Examples:**
```bash
hostodo pay INV-12345
```

### SSH Key Management

#### `hostodo keys list`

List all SSH keys. Alias: `ls`.

#### `hostodo keys add [name] [public-key]`

Add a new SSH key.

**Flags:**
- `-f, --file string` - Read public key from file

**Examples:**
```bash
# Add key inline
hostodo keys add mykey "ssh-rsa AAAAB3NzaC1yc2EAAA... user@host"

# Add key from file
hostodo keys add mykey --file ~/.ssh/id_rsa.pub
```

#### `hostodo keys remove <name>`

Remove an SSH key.

**Examples:**
```bash
hostodo keys remove mykey
```

### Shell Completions

#### `hostodo completion`

Generate shell completion scripts.

**Examples:**
```bash
# Bash
hostodo completion bash > /etc/bash_completion.d/hostodo

# Zsh
hostodo completion zsh > "${fpath[1]}/_hostodo"

# Fish
hostodo completion fish > ~/.config/fish/completions/hostodo.fish
```

## 🔧 Configuration

### Configuration File

Configuration is stored in `~/.hostodo/config.json`:

```json
{
  "api_url": "https://api.hostodo.com",
  "device_id": "a1b2c3d4-..."
}
```

Access tokens are stored separately in the OS keychain (macOS Keychain, Linux Secret Service), with an AES-encrypted file fallback at `~/.hostodo/token.enc` when no keychain is available.

**Security:**
- Config file permissions are automatically set to `0600` (owner read/write only)
- Directory permissions are set to `0700` (owner read/write/execute only)
- Tokens are stored in the OS keychain, not in config files

### Environment Variables

- `HOSTODO_API_URL` - Override the default API URL
- `HOME` - Used to locate the config directory

**Example:**
```bash
export HOSTODO_API_URL=http://localdev.hostodo.com:8000
hostodo login
```

## 🎨 Output Formats

### Interactive TUI

The default output format provides a beautiful, interactive table:

- Keyboard navigation with arrow keys or vim keys (j/k)
- Color-coded status indicators
- Press Enter to view detailed instance information
- Responsive layout that adapts to terminal size

### JSON Output

Perfect for scripting and automation:

```bash
hostodo list --json | jq '.[] | select(.status == "running")'
```

### Simple Table

Clean ASCII table for quick viewing or piping:

```
ID            HOSTNAME                   IP ADDRESS        STATUS          POWER       RAM (MB)    CPU  DISK (GB)
abc123        server1.hostodo.com        192.168.1.100     provisioned     running         4096      2        50
def456        server2.hostodo.com        192.168.1.101     provisioned     stopped         2048      1        25
```

### Detailed View

Comprehensive information in a readable format:

```
Instance: abc123
  Hostname:     server1.hostodo.com
  IP Address:   192.168.1.100
  Status:       provisioned
  Power:        running
  Resources:    4096 MB RAM, 2 CPU, 50 GB Disk
  Bandwidth:    45.20 / 1000 GB
  Plan:         Starter VPS
  Template:     Ubuntu 22.04
  Region:       US-East
  Billing:      $5.00 / monthly
  Next Due:     2025-12-08
```

## Claude Code

Manage your Hostodo instances directly from [Claude Code](https://claude.ai/code) using natural language.

### Install

```bash
make install-skill
```

This copies the skill file to `~/.claude/commands/hostodo.md`.

### Usage

In any Claude Code session, use the `/hostodo` command:

```
/hostodo list my instances
/hostodo status my-server
/hostodo deploy ubuntu in LA
/hostodo stop my-server
/hostodo show unpaid invoices
```

The skill translates your intent into non-interactive CLI commands, confirms destructive actions, and formats output as readable markdown.

## 🏗️ Development

### Prerequisites

- Go 1.24 or higher
- Git

### Building from Source

```bash
# Clone the repository
git clone https://github.com/hostodo/hostodo-cli.git
cd hostodo-cli

# Install dependencies
go mod download

# Build
go build -o hostodo .

# Run
./hostodo --help
```

### Project Structure

```
hostodo-cli/
├── cmd/
│   ├── root.go              # Root command and command registration
│   ├── list.go              # List instances command
│   ├── status.go            # Instance status/details command
│   ├── start.go             # Start instance command
│   ├── stop.go              # Stop instance command
│   ├── restart.go           # Restart instance command
│   ├── ssh.go               # SSH into instance command
│   ├── deploy.go            # Deploy new instance wizard
│   ├── invoices.go          # List invoices command
│   ├── pay.go               # Pay invoice command
│   ├── keys.go              # SSH key management commands
│   ├── completion.go        # Shell completion command
│   └── auth/
│       ├── auth.go          # Auth parent command
│       ├── login.go         # OAuth device flow login
│       ├── logout.go        # Logout command
│       ├── whoami.go        # Current user info
│       └── sessions.go      # CLI session management
├── pkg/
│   ├── api/
│   │   ├── client.go        # HTTP API client with Bearer auth
│   │   ├── auth.go          # Authentication endpoints
│   │   ├── instances.go     # Instance endpoints
│   │   ├── deploy.go        # Deployment endpoints
│   │   ├── invoices.go      # Billing endpoints
│   │   ├── sshkeys.go       # SSH key endpoints
│   │   ├── sessions.go      # Session endpoints
│   │   └── models.go        # Request/response structs
│   ├── auth/
│   │   ├── keychain.go      # Token storage (keychain + encrypted fallback)
│   │   └── oauth.go         # OAuth device flow client
│   ├── config/
│   │   └── config.go        # Config file management
│   ├── resolver/
│   │   └── resolver.go      # Hostname → instance resolution + completions
│   ├── deploy/
│   │   └── hostname.go      # Hostname generation (adjective-noun combos)
│   ├── ui/
│   │   ├── table.go         # Interactive Bubble Tea table component
│   │   ├── styles.go        # Lipgloss styles
│   │   └── formatters.go    # Output formatters (JSON/simple/details)
│   └── utils/
│       └── ssh.go           # SSH fingerprint utilities
├── main.go
├── go.mod
├── Makefile
└── README.md
```

### Dependencies

Core libraries:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling library
- [Survey](https://github.com/AlecAivazis/survey) - Interactive prompts
- [go-keyring](https://github.com/zalando/go-keyring) - OS keychain access
- [go-figure](https://github.com/common-nighthawk/go-figure) - ASCII art text

### Testing

```bash
# Run all tests
make test

# Test against local development API
export HOSTODO_API_URL=http://localdev.hostodo.com:8000
hostodo login
hostodo list
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📝 License

MIT License - see LICENSE file for details

## 🔗 Links

- [Hostodo Website](https://hostodo.com)
- [Hostodo Console](https://console.hostodo.com)
- [API Documentation](https://console.hostodo.com/api/docs)
- [GitHub Repository](https://github.com/hostodo/hostodo)

## 💬 Support

- Email: support@hostodo.com
- Discord: [Join our community](https://discord.gg/hostodo)
- Documentation: [docs.hostodo.com](https://docs.hostodo.com)

## 🎯 Roadmap

### Completed
- [x] SSH key management commands
- [x] Instance deployment wizard
- [x] Billing and invoice management
- [x] SSH connectivity
- [x] Shell auto-completion for hostnames
- [x] OAuth device flow authentication

### Planned
- [ ] Bandwidth monitoring and alerts
- [ ] Backup management
- [ ] Network (reverse DNS) management
- [ ] Interactive dashboard with real-time metrics
- [ ] Log streaming and viewing
- [ ] Custom script execution
- [ ] Bulk operations
- [ ] Configuration templates

---

**Built with ❤️ by the Hostodo team**
