# Homebrew formula for the Cli developer bootstrap platform
# (github.com/intruder0007/Cli). See docs/architecture/distribution-protocol.md.
#
# Installs the whole extracted archive into libexec and symlinks only
# `bootstrap` into bin — a bare `bin.install "bootstrap"` would strand
# the sibling templates/plugins/builtin directories the running binary
# still depends on today (see the protocol doc's "Why the whole
# archive"). v0.2.0 predates ADR-0012's embedded-plugin fallback, so
# this stays the conservative, whole-archive-preserving pattern rather
# than the simplified one the embed feature will eventually unlock —
# matching ADR-0012's own "don't simplify the archive without a
# measurable need yet" decision.
class Bootstrap < Formula
  desc "Cross-language developer bootstrap platform: generate new projects via a plugin-based CLI"
  homepage "https://github.com/intruder0007/Cli"
  version "0.2.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/intruder0007/Cli/releases/download/v0.2.0/cli_v0.2.0_darwin_arm64.tar.gz"
      sha256 "2496215b0c41896a354797097733726c9ea402dc66e3478d6d2322242038de19"
    else
      url "https://github.com/intruder0007/Cli/releases/download/v0.2.0/cli_v0.2.0_darwin_amd64.tar.gz"
      sha256 "0124052d890606051bcad0fc367cde23804215e77c5a28d8d9ac4ed28a911c2e"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/intruder0007/Cli/releases/download/v0.2.0/cli_v0.2.0_linux_arm64.tar.gz"
      sha256 "a5982fecdc7d2348e4f5385dd7c393b1265bf9a3ced568e2a0005e06a3c36cd1"
    else
      url "https://github.com/intruder0007/Cli/releases/download/v0.2.0/cli_v0.2.0_linux_amd64.tar.gz"
      sha256 "8292a7acca11fa6b5df346fd2b6bb8989c4f0445ff52748640c88d3f23876b2d"
    end
  end

  def install
    libexec.install Dir["*"]
    bin.install_symlink libexec/"bootstrap"
  end

  test do
    # `bootstrap version` is the cross-ecosystem proof point every
    # distribution-verify.yml job checks — a real invocation of the
    # installed binary. `bootstrap doctor`'s plugin-discovery result
    # depends on cwd/exe-relative lookup nuances specific to Homebrew's
    # test sandbox and isn't this formula's job to re-prove — that's
    # already covered by the Go test suite (tests/integration).
    assert_match "bootstrap version", shell_output("#{bin}/bootstrap version")
  end
end
