#!/bin/bash
#
# Homebrew Setup Script
# This script helps you set up everything needed for Homebrew distribution
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_step() {
    echo -e "${BLUE}==>${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}!${NC} $1"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Main setup
main() {
    echo -e "${GREEN}╔════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  Hostodo CLI - Homebrew Setup Assistant   ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════╝${NC}"
    echo

    # Check prerequisites
    print_step "Checking prerequisites..."

    if ! command_exists go; then
        print_error "Go is not installed. Please install Go 1.24+ first."
        echo "  Visit: https://golang.org/dl/"
        exit 1
    fi
    print_success "Go is installed: $(go version)"

    if ! command_exists git; then
        print_error "Git is not installed. Please install Git first."
        exit 1
    fi
    print_success "Git is installed: $(git --version)"

    if ! command_exists goreleaser; then
        print_warning "GoReleaser is not installed"
        read -p "Install GoReleaser now? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            if command_exists brew; then
                brew install goreleaser
                print_success "GoReleaser installed"
            else
                print_error "Homebrew not found. Please install GoReleaser manually:"
                echo "  Visit: https://goreleaser.com/install/"
                exit 1
            fi
        else
            print_error "GoReleaser is required. Exiting."
            exit 1
        fi
    else
        print_success "GoReleaser is installed: $(goreleaser --version | head -n 1)"
    fi

    echo

    # Check Git repository status
    print_step "Checking Git repository..."

    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_error "Not a Git repository"
        exit 1
    fi
    print_success "Git repository detected"

    REMOTE_URL=$(git config --get remote.origin.url || echo "")
    if [[ -z "$REMOTE_URL" ]]; then
        print_warning "No remote 'origin' found"
    else
        print_success "Remote origin: $REMOTE_URL"
    fi

    echo

    # Validate GoReleaser configuration
    print_step "Validating GoReleaser configuration..."

    if [ ! -f ".goreleaser.yml" ]; then
        print_error ".goreleaser.yml not found"
        exit 1
    fi

    if goreleaser check; then
        print_success "GoReleaser configuration is valid"
    else
        print_error "GoReleaser configuration has errors"
        exit 1
    fi

    echo

    # Test build
    print_step "Testing snapshot build..."

    if goreleaser release --snapshot --clean --skip=publish; then
        print_success "Snapshot build successful"

        # Show generated files
        echo
        print_step "Generated files:"
        ls -lh dist/ | grep -E "(darwin|linux|windows)" | head -10

        # Test binary
        if [ -f "dist/hostodo_darwin_arm64/hostodo" ]; then
            echo
            print_step "Testing binary..."
            ./dist/hostodo_darwin_arm64/hostodo --version
            print_success "Binary works correctly"
        fi
    else
        print_error "Snapshot build failed"
        exit 1
    fi

    echo
    echo -e "${GREEN}╔════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║           Setup Complete! 🎉               ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════╝${NC}"
    echo
    echo "Next steps:"
    echo
    echo "1. Create a GitHub repository (if not already done):"
    echo "   ${BLUE}https://github.com/new${NC}"
    echo
    echo "2. Create a Homebrew tap repository:"
    echo "   ${BLUE}https://github.com/new${NC}"
    echo "   Name it: ${GREEN}homebrew-tap${NC}"
    echo
    echo "3. Create a Personal Access Token:"
    echo "   ${BLUE}https://github.com/settings/tokens${NC}"
    echo "   Scope: ${GREEN}repo${NC}"
    echo
    echo "4. Add token to repository secrets:"
    echo "   Repository → Settings → Secrets → New secret"
    echo "   Name: ${GREEN}TAP_GITHUB_TOKEN${NC}"
    echo
    echo "5. Create your first release:"
    echo "   ${YELLOW}git tag -a v0.1.0 -m \"Release v0.1.0\"${NC}"
    echo "   ${YELLOW}git push origin v0.1.0${NC}"
    echo
    echo "6. Or use the Makefile:"
    echo "   ${YELLOW}make release-check${NC}  # Check everything is ready"
    echo "   ${YELLOW}make release${NC}        # Create a release (requires git tag)"
    echo
    echo "For more information, see: ${BLUE}HOMEBREW_SETUP.md${NC}"
    echo
}

# Run main function
main "$@"
