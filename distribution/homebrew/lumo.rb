# Homebrew formula for the Lumo project scaffolding platform
# (github.com/intruder0007/Lumo). See docs/architecture/distribution-protocol.md.
#
# Installs the whole extracted archive into libexec and symlinks only
# `lumo` into bin — a bare `bin.install "lumo"` would strand
# the sibling templates/plugins/builtin directories (see the protocol
# doc's "Why the whole archive"). Since v0.3.0 the binary carries
# ADR-0012's embedded-plugin fallback and no longer *strictly* depends
# on those siblings, but this stays the conservative,
# whole-archive-preserving pattern rather than the simplified one the
# embed feature could unlock — matching ADR-0012's own "don't simplify
# the archive without a measurable need yet" decision.
class Lumo < Formula
  desc "Cross-language project scaffolding platform: generate new projects via a plugin-based CLI"
  homepage "https://github.com/intruder0007/Lumo"
  version "0.4.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/intruder0007/Lumo/releases/download/v0.4.0/lumo_v0.4.0_darwin_arm64.tar.gz"
      sha256 "f1368fa64075a6a0bd9e0a4a13b7c9407759c898750ca938499ff604dbb5b69f"
    else
      url "https://github.com/intruder0007/Lumo/releases/download/v0.4.0/lumo_v0.4.0_darwin_amd64.tar.gz"
      sha256 "6953401c0c8bc9d923ab30a2bc980e09fe6c6f12695d8f440bfb00b170813aad"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/intruder0007/Lumo/releases/download/v0.4.0/lumo_v0.4.0_linux_arm64.tar.gz"
      sha256 "8aaefa766a227fc950841ea2d278a017814edf3eb89f89193fa6151cb0ca336c"
    else
      url "https://github.com/intruder0007/Lumo/releases/download/v0.4.0/lumo_v0.4.0_linux_amd64.tar.gz"
      sha256 "fccf79b964911215da618e85806db2745289299cc6f46ac2ff9fe5004f954d07"
    end
  end

  def install
    libexec.install Dir["*"]
    bin.install_symlink libexec/"lumo"
  end

  test do
    # `lumo version` is the cross-ecosystem proof point every
    # distribution-verify.yml job checks — a real invocation of the
    # installed binary. `lumo doctor`'s plugin-discovery result
    # depends on cwd/exe-relative lookup nuances specific to Homebrew's
    # test sandbox and isn't this formula's job to re-prove — that's
    # already covered by the Go test suite (tests/integration).
    assert_match "lumo version", shell_output("#{bin}/lumo version")
  end
end
