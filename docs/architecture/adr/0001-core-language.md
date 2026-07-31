# ADR-0001: Core CLI and engine implemented in Go

## Status

Accepted

## Context

The core CLI and orchestration engine need to run on Linux, macOS, and
Windows, offline, with no required runtime installed on the target machine
(offline-first, cross-platform). Candidates considered: Go, Node.js/TypeScript,
Rust.

- **Node.js/TypeScript** has the largest ecosystem and fastest prototyping,
  but requires a Node runtime on the user's machine (or a bundler like
  `pkg`/`nexe`), and carries a larger npm supply-chain surface for a tool
  that will `go install`/download-and-run for many users.
- **Rust** compiles to a single static binary like Go, with stronger memory
  safety, but has a steeper learning curve that raises the bar for casual
  open-source contributors to this specific project.
- **Go** compiles to a single static binary per OS/arch, has no runtime
  dependency, fast startup, first-class cross-compilation, and a
  contributor pool comfortable with CLI tooling (Go is a common choice for
  developer tools: `git`-adjacent tooling, `kubectl`, `terraform`, `gh`).

## Decision

The core CLI (`cli/`) and orchestration engine (`core/`) are implemented in
Go. Each subsystem is its own Go module; `go.work` ties them together for
local development. This does not require plugins to be written in Go — see
[ADR-0002](0002-plugin-protocol.md).

## Consequences

- Distribution is a single binary per platform; no install-time runtime
  dependency.
- The plugin protocol must be language-agnostic from day one, since the
  core being Go does not imply plugins are Go (see ADR-0002).
- The first-party SDK (`sdk/go`) and V1 template/capability plugins are
  Go too, for consistency and to dogfood the protocol; non-Go SDKs are
  deferred (see [roadmap.md](../roadmap.md)).
