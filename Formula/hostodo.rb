# Homebrew Formula for hostodo-cli
# To install locally: brew install --build-from-source Formula/hostodo.rb

class Hostodo < Formula
  desc "Official CLI for managing Hostodo VPS instances"
  homepage "https://github.com/hostodo/hostodo-cli"
  license "MIT"

  # For local development/testing before releases
  # Once you have releases on GitHub, GoReleaser will auto-generate this
  url "https://github.com/hostodo/hostodo-cli/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "" # Add checksum after first release

  # Uncomment when you have releases:
  # version "0.1.0"
  # url "https://github.com/hostodo/hostodo-cli/releases/download/v#{version}/hostodo_#{version}_Darwin_arm64.tar.gz"
  # sha256 "..." # GoReleaser will generate this

  depends_on "go" => :build

  def install
    # Set build variables
    commit = Utils.git_short_head
    build_date = Time.now.utc.strftime("%Y-%m-%dT%H:%M:%SZ")

    ldflags = %W[
      -s -w
      -X github.com/hostodo/hostodo-cli/cmd.Version=#{version}
      -X github.com/hostodo/hostodo-cli/cmd.Commit=#{commit}
      -X github.com/hostodo/hostodo-cli/cmd.Date=#{build_date}
    ]

    system "go", "build", *std_go_args(ldflags: ldflags), "-o", bin/"hostodo"

    # Generate shell completions (if you add this feature)
    # generate_completions_from_executable(bin/"hostodo", "completion")
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/hostodo --version")

    # Test that the binary runs
    output = shell_output("#{bin}/hostodo --help")
    assert_match "Official CLI for managing Hostodo VPS instances", output
  end
end
