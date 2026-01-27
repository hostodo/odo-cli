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

You'll be prompted for your email and password. Credentials are securely stored in `~/.hostodo/config.json` with restricted permissions (0600).

**Options:**
- `--email` / `-e`: Provide email address directly
- `--remember-me` / `-r`: Extend session to 7 days (default: 24 hours)
- `--api-url`: Use a custom API URL (default: https://console.hostodo.com)

### Basic Usage

```bash
# List all instances (interactive TUI)
hostodo instances list

# List instances as JSON
hostodo instances list --json

# List instances as simple table
hostodo instances list --simple

# View detailed information
hostodo instances list --details

# Get details about a specific instance
hostodo instances get <instance-id>

# Power control
hostodo instances start <instance-id>
hostodo instances stop <instance-id>
hostodo instances reboot <instance-id>

# Logout
hostodo logout
```

## 📖 Commands

### Global Flags

All commands support these global flags:

- `--api-url string` - API URL (default: https://console.hostodo.com or $HOSTODO_API_URL)
- `--config string` - Config file path (default: $HOME/.hostodo/config.json)
- `-h, --help` - Show help
- `-v, --version` - Show version

### Authentication Commands

#### `hostodo login`

Authenticate with your Hostodo account and store credentials.

**Flags:**
- `-e, --email string` - Email address
- `-r, --remember-me` - Remember me (extend session to 7 days)

**Examples:**
```bash
hostodo login
hostodo login --email user@example.com
hostodo login --remember-me
```

#### `hostodo logout`

Clear stored credentials and logout.

**Example:**
```bash
hostodo logout
```

### Instance Commands

#### `hostodo instances list`

List all your VPS instances with various output formats.

**Flags:**
- `--json` - Output as JSON
- `--simple` - Output as simple ASCII table
- `--details` - Show detailed information
- `--limit int` - Maximum number of instances to fetch (default: 100)
- `--offset int` - Offset for pagination (default: 0)

**Examples:**
```bash
# Interactive TUI (default)
hostodo instances list

# JSON output for scripting
hostodo instances list --json

# Simple table for quick viewing
hostodo instances list --simple

# Detailed view with all information
hostodo instances list --details

# Fetch 50 instances
hostodo instances list --limit 50
```

**Interactive TUI Controls:**
- `↑/↓` or `j/k` - Navigate through instances
- `Enter` - View detailed information about selected instance
- `q` or `Ctrl+C` or `Esc` - Quit

#### `hostodo instances get <instance-id>`

Get detailed information about a specific instance.

**Flags:**
- `--json` - Output as JSON

**Examples:**
```bash
hostodo instances get abc123
hostodo instances get abc123 --json
```

**Information Displayed:**
- Basic information (ID, hostname, status, power state)
- Network configuration (IPs, MAC address)
- Resource allocation (RAM, CPU, Disk, Bandwidth usage)
- Plan and template details
- Billing information (amount, cycle, next due date)
- Timeline (created, updated timestamps)

#### `hostodo instances start <instance-id>`

Start a stopped VPS instance.

**Examples:**
```bash
hostodo instances start abc123
```

The command will:
1. Send the start command to the instance
2. Wait for the instance to boot (up to 30 seconds)
3. Display status updates

#### `hostodo instances stop <instance-id>`

Stop a running VPS instance.

**Flags:**
- `-f, --force` - Force immediate shutdown (without graceful shutdown)

**Examples:**
```bash
# Graceful shutdown
hostodo instances stop abc123

# Force shutdown
hostodo instances stop abc123 --force
```

The command will:
1. Send the stop command to the instance
2. Wait for the instance to shutdown (up to 60 seconds)
3. Display status updates

#### `hostodo instances reboot <instance-id>`

Reboot a VPS instance.

**Flags:**
- `-f, --force` - Force immediate reboot

**Examples:**
```bash
# Graceful reboot
hostodo instances reboot abc123

# Force reboot
hostodo instances reboot abc123 --force
```

The command will:
1. Send the reboot command to the instance
2. Wait for the instance to restart (up to 90 seconds)
3. Display status updates

## 🔧 Configuration

### Configuration File

Credentials are stored in `~/.hostodo/config.json`:

```json
{
  "api_url": "https://console.hostodo.com",
  "access_token": "...",
  "refresh_token": "...",
  "email": "user@example.com"
}
```

**Security:**
- File permissions are automatically set to `0600` (owner read/write only)
- Directory permissions are set to `0700` (owner read/write/execute only)
- Tokens are automatically refreshed when expired

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
hostodo instances list --json | jq '.[] | select(.status == "running")'
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

## 🏗️ Development

### Prerequisites

- Go 1.24 or higher
- Git

### Building from Source

```bash
# Clone the repository
git clone https://github.com/hostodo/hostodo.git
cd hostodo/hostodo-cli

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
│   ├── root.go              # Root command
│   ├── login.go             # Login command
│   ├── logout.go            # Logout command
│   └── instances/
│       ├── instances.go     # Instances parent command
│       ├── list.go          # List command
│       ├── get.go           # Get command
│       ├── start.go         # Start command
│       ├── stop.go          # Stop command
│       └── reboot.go        # Reboot command
├── pkg/
│   ├── api/
│   │   ├── client.go        # API client with JWT handling
│   │   ├── auth.go          # Authentication endpoints
│   │   ├── instances.go     # Instance endpoints
│   │   └── models.go        # Response/request structs
│   ├── config/
│   │   └── config.go        # Config file management
│   └── ui/
│       ├── table.go         # Interactive table component
│       ├── styles.go        # Lipgloss styles
│       └── formatters.go    # Output formatters
├── main.go
├── go.mod
└── README.md
```

### Dependencies

Core libraries:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling library
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering

### Testing

```bash
# Test against local development API
export HOSTODO_API_URL=http://localdev.hostodo.com:8000
./hostodo login
./hostodo instances list
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

### Phase 2 (Planned)
- [ ] SSH key management commands
- [ ] Bandwidth monitoring and alerts
- [ ] Instance deployment wizard
- [ ] Backup management
- [ ] Network (reverse DNS) management
- [ ] Billing and invoice management

### Phase 3 (Future)
- [ ] Interactive dashboard with real-time metrics
- [ ] Log streaming and viewing
- [ ] Custom script execution
- [ ] Bulk operations
- [ ] Auto-completion for instance IDs
- [ ] Configuration templates

---

**Built with ❤️ by the Hostodo team**
