# Release Guide - Quick Reference

## Prerequisites

- [x] GoReleaser installed: `brew install goreleaser`
- [x] GitHub repository set up
- [x] Git tags for versioning

## Quick Start

### 1. Test Everything Works

```bash
# Run setup assistant
./scripts/setup-homebrew.sh

# Or manually:
goreleaser check                           # Validate config
goreleaser release --snapshot --clean      # Test build
```

### 2. Create a Release

```bash
# Update version and commit changes
git add .
git commit -m "chore: prepare v0.1.0 release"

# Create and push tag
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin main
git push origin v0.1.0

# GitHub Actions will automatically:
# - Build for all platforms
# - Create GitHub release
# - Publish to Homebrew tap
```

### 3. Verify Release

```bash
# Check GitHub releases
# https://github.com/hostodo/hostodo-cli/releases

# Test Homebrew installation
brew tap hostodo/tap
brew install hostodo

hostodo --version
```

## Manual Release (Without GitHub Actions)

```bash
export GITHUB_TOKEN="your_github_token"
export TAP_GITHUB_TOKEN="your_tap_token"

git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0

goreleaser release --clean
```

## Using Make

```bash
make release-check    # Validate everything
make snapshot         # Local test build
make release          # Create release (requires tag)
```

## Homebrew Tap Setup (One-Time)

### Option 1: Separate Tap Repository (Recommended)

1. Create repository: `homebrew-tap`
2. Create PAT with `repo` scope
3. Add secret `TAP_GITHUB_TOKEN` to hostodo-cli repo
4. Update `.goreleaser.yml` with tap repo name

Users install with:
```bash
brew tap hostodo/tap
brew install hostodo
```

### Option 2: Same Repository

Users install with:
```bash
brew install hostodo/hostodo-cli/hostodo
```

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- `v0.1.0` - Initial release
- `v0.1.1` - Patch (bug fixes)
- `v0.2.0` - Minor (new features)
- `v1.0.0` - Major (breaking changes)

## Common Commands

```bash
# Build locally
make build

# Install locally
make install

# Build for all platforms
make build-all

# Clean build artifacts
make clean

# Run tests
make test
```

## Troubleshooting

### "dirty git state"
```bash
git status
git add .
git commit -m "message"
```

### "token not found"
```bash
export GITHUB_TOKEN="your_token"
```

### "Formula not found"
```bash
brew update
brew tap hostodo/tap
```

## Files Overview

| File | Purpose |
|------|---------|
| `.goreleaser.yml` | GoReleaser configuration |
| `.github/workflows/release.yml` | GitHub Actions for releases |
| `Makefile` | Build automation |
| `Formula/hostodo.rb` | Homebrew formula (local testing) |
| `HOMEBREW_SETUP.md` | Detailed setup guide |

## Resources

- [GoReleaser Docs](https://goreleaser.com)
- [Homebrew Formula Guide](https://docs.brew.sh/Formula-Cookbook)
- [GitHub Actions](https://docs.github.com/en/actions)
- [Semantic Versioning](https://semver.org/)

## Support

For issues:
1. Check [HOMEBREW_SETUP.md](HOMEBREW_SETUP.md) for detailed troubleshooting
2. Review GoReleaser logs in GitHub Actions
3. Open an issue on GitHub
