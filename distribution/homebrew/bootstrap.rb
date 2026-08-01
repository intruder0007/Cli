# Homebrew formula for the Cli developer bootstrap platform
# (github.com/intruder0007/Cli). See docs/architecture/distribution-protocol.md.
#
# Installs the whole extracted archive into libexec and symlinks only
# `bootstrap` into bin — a bare `bin.install "bootstrap"` would strand
# the sibling templates/plugins/builtin directories (see the protocol
# doc's "Why the whole archive"). Since v0.3.0 the binary carries
# ADR-0012's embedded-plugin fallback and no longer *strictly* depends
# on those siblings, but this stays the conservative,
# whole-archive-preserving pattern rather than the simplified one the
# embed feature could unlock — matching ADR-0012's own "don't simplify
# the archive without a measurable need yet" decision.
class Bootstrap < Formula
  desc "Cross-language developer bootstrap platform: generate new projects via a plugin-based CLI"
  homepage "https://github.com/intruder0007/Cli"
  version "0.3.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/intruder0007/Cli/releases/download/v0.3.0/cli_v0.3.0_darwin_arm64.tar.gz"
      sha256 "2bbce76476399baf1b8b6cc66a0413d477ca433bd1bce06396dead49c389820e"
    else
      url "https://github.com/intruder0007/Cli/releases/download/v0.3.0/cli_v0.3.0_darwin_amd64.tar.gz"
      sha256 "3573db24c99a885d089766baf9dc612727d39011e37ebcb6048f1ea539d114e1"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/intruder0007/Cli/releases/download/v0.3.0/cli_v0.3.0_linux_arm64.tar.gz"
      sha256 "a72431abe33c0c66111c79a3bae3e2976a04a7003c156f350f87256295c34d99"
    else
      url "https://github.com/intruder0007/Cli/releases/download/v0.3.0/cli_v0.3.0_linux_amd64.tar.gz"
      sha256 "40e5bfb27205669312387e10c84db624ce500d4dad46a7317edd04d4e046a92e"
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
