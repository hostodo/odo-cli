# Homebrew Distribution Setup Guide

This guide explains how to set up and maintain Homebrew distribution for the hostodo-cli.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Initial Setup](#initial-setup)
3. [Creating a Homebrew Tap](#creating-a-homebrew-tap)
4. [Release Process](#release-process)
5. [Testing](#testing)
6. [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Tools
- Go 1.24+
- Git
- GitHub account with repository access
- GoReleaser (`brew install goreleaser`)
- Homebrew (for testing)

### Repository Setup
Ensure your repository is set up correctly:
```bash
# Verify repository URL in go.mod
cat go.mod | grep module
# Should show: module github.com/hostodo/hostodo-cli

# Ensure you're on GitHub
git remote -v
```

## Initial Setup

### 1. Install GoReleaser

```bash
brew install goreleaser
```

### 2. Test GoReleaser Configuration

```bash
# Check if configuration is valid
goreleaser check

# Create a test/snapshot release (doesn't push to GitHub)
goreleaser release --snapshot --clean

# Check the generated files
ls -la dist/
```

### 3. Add Version Information to CLI

Update `cmd/root.go` to include version information:

```go
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
)

var (
    Version = "dev"
    Commit  = "none"
    Date    = "unknown"
)

var rootCmd = &cobra.Command{
    Use:   "hostodo",
    Short: "Official CLI for managing Hostodo VPS instances",
    Long:  `Hostodo CLI allows you to manage your VPS instances from the command line.`,
    Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date),
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    // Add version template
    rootCmd.SetVersionTemplate(`{{.Version}}`)
}
```

## Creating a Homebrew Tap

### Option 1: Use the Main Repository (Simpler)

GoReleaser can create a formula in the same repository:

1. Update `.goreleaser.yml`:
```yaml
brews:
  - name: hostodo
    repository:
      owner: hostodo
      name: hostodo-cli
    directory: Formula
    # ... rest of config
```

2. Users install with:
```bash
brew install hostodo/hostodo-cli/hostodo
```

### Option 2: Create a Separate Tap Repository (Recommended)

This is the standard Homebrew practice:

1. **Create a new repository** on GitHub named `homebrew-tap`:
   - Go to https://github.com/hostodo
   - Click "New repository"
   - Name: `homebrew-tap` (must start with `homebrew-`)
   - Public visibility
   - Initialize with README

2. **Set up repository access**:
   - Create a Personal Access Token (PAT) with `repo` scope
   - Go to: GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
   - Generate new token with `repo` permissions
   - Save the token securely

3. **Add token to repository secrets**:
   - Go to hostodo-cli repository → Settings → Secrets and variables → Actions
   - Click "New repository secret"
   - Name: `TAP_GITHUB_TOKEN`
   - Value: Your PAT
   - Click "Add secret"

4. **Update `.goreleaser.yml`**:
```yaml
brews:
  - name: hostodo
    repository:
      owner: hostodo
      name: homebrew-tap
      token: "{{ .Env.TAP_GITHUB_TOKEN }}"
    # ... rest stays the same
```

5. **Update GitHub Actions workflow** (`.github/workflows/release.yml`):
```yaml
env:
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}
```

6. **Users install with**:
```bash
brew tap hostodo/tap
brew install hostodo
```

## Release Process

### Semantic Versioning

Follow [Semantic Versioning](https://semver.org/):
- `vX.Y.Z` (e.g., v1.2.3)
- MAJOR version for incompatible API changes
- MINOR version for new features (backwards compatible)
- PATCH version for bug fixes

### Creating a Release

#### Method 1: Using Git Tags (Automated)

```bash
# Ensure you're on main branch and up to date
git checkout main
git pull origin main

# Create and push a tag
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0

# GitHub Actions will automatically:
# 1. Build binaries for all platforms
# 2. Create a GitHub release
# 3. Upload binaries
# 4. Update Homebrew formula
```

#### Method 2: Manual Release with GoReleaser

```bash
# Set GitHub token
export GITHUB_TOKEN="your_github_token"
export TAP_GITHUB_TOKEN="your_tap_token"  # If using separate tap

# Create a tag
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0

# Run GoReleaser
goreleaser release --clean
```

### Release Checklist

Before creating a release:

- [ ] Update version in README badges
- [ ] Update CHANGELOG.md
- [ ] Test all commands locally
- [ ] Run tests: `go test ./...`
- [ ] Build locally: `make build`
- [ ] Commit all changes
- [ ] Push to main branch

After creating a release:

- [ ] Verify GitHub release was created
- [ ] Download and test binaries
- [ ] Test Homebrew installation
- [ ] Announce release (if applicable)

## Testing

### Test Locally Before Release

```bash
# Create a snapshot release (local only)
goreleaser release --snapshot --clean

# Test the binary
./dist/hostodo_darwin_arm64/hostodo --version
./dist/hostodo_darwin_arm64/hostodo --help
```

### Test Homebrew Formula Locally

```bash
# Test formula syntax
brew audit --formula Formula/hostodo.rb

# Test installation from local formula
brew install --build-from-source Formula/hostodo.rb

# Verify installation
which hostodo
hostodo --version

# Clean up
brew uninstall hostodo
```

### Test Homebrew Installation After Release

```bash
# If using separate tap
brew tap hostodo/tap
brew install hostodo

# Or directly
brew install hostodo/tap/hostodo

# Test
hostodo --version
hostodo --help

# Uninstall
brew uninstall hostodo
brew untap hostodo/tap
```

## Troubleshooting

### GoReleaser Fails

**Error: "Git is in dirty state"**
```bash
# Commit all changes
git status
git add .
git commit -m "your message"
```

**Error: "Token not found"**
```bash
# Set GitHub token
export GITHUB_TOKEN="your_token"
export TAP_GITHUB_TOKEN="your_tap_token"
```

### Homebrew Installation Fails

**Error: "Formula not found"**
```bash
# Update Homebrew
brew update

# If using tap, ensure it's added
brew tap hostodo/tap

# Force refresh
brew untap hostodo/tap
brew tap hostodo/tap
```

**Error: "Checksum mismatch"**
- GoReleaser should handle this automatically
- If manual formula, calculate checksum:
```bash
shasum -a 256 your-binary.tar.gz
```

### Version Not Showing Correctly

Ensure ldflags are set correctly in `.goreleaser.yml`:
```yaml
ldflags:
  - -X github.com/hostodo/hostodo-cli/cmd.Version={{.Version}}
  - -X github.com/hostodo/hostodo-cli/cmd.Commit={{.Commit}}
  - -X github.com/hostodo/hostodo-cli/cmd.Date={{.Date}}
```

## Updating the Formula

### For Patch Releases

GoReleaser handles everything automatically. Just create a new tag.

### For Major Changes

If you change binary name, dependencies, or installation process:

1. Update `.goreleaser.yml`
2. Test with `goreleaser release --snapshot --clean`
3. Review generated formula in `dist/`
4. Create release

### Manual Formula Updates

If you need to manually update the formula in your tap:

```bash
# Clone the tap repository
git clone https://github.com/hostodo/homebrew-tap.git
cd homebrew-tap

# Edit Formula/hostodo.rb
# Update version, URL, sha256

# Test locally
brew install --build-from-source Formula/hostodo.rb

# Commit and push
git add Formula/hostodo.rb
git commit -m "Update hostodo formula to v0.2.0"
git push origin main
```

## Best Practices

1. **Always test locally first**
   - Use `--snapshot` flag to test without publishing
   - Test installation from generated binaries

2. **Use semantic versioning**
   - Clear version numbers help users understand changes
   - Follow semver.org guidelines

3. **Write good release notes**
   - GoReleaser auto-generates from git commits
   - Use conventional commits (feat:, fix:, etc.)

4. **Keep formula simple**
   - Let GoReleaser handle formula generation
   - Only customize if necessary

5. **Monitor installations**
   - Check GitHub release downloads
   - Monitor issues related to installation

## Resources

- [GoReleaser Documentation](https://goreleaser.com)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Homebrew Acceptable Formulae](https://docs.brew.sh/Acceptable-Formulae)
- [Semantic Versioning](https://semver.org/)

## Support

If you encounter issues:
1. Check the [Troubleshooting](#troubleshooting) section
2. Review GoReleaser logs in GitHub Actions
3. Open an issue on GitHub
4. Contact the development team
