# ADR-0002: Plugins communicate over subprocess + line-delimited JSON-RPC 2.0

## Status

Accepted

## Context

Plugins (templates and capabilities) must be authorable in any language,
sandboxable, debuggable, and usable fully offline. Candidates considered:

- **WebAssembly (WASM) modules**, loaded in-process via a WASM runtime.
  Fast and sandboxed, but restricts plugin authors to languages with
  mature WASM toolchains, and adds real host-runtime complexity that isn't
  justified for a V1 with one first-party template.
- **Native dynamic libraries** (`.so`/`.dll`/`.dylib`), loaded in-process.
  Fastest, but ties plugins to the host's ABI and Go version, is fragile
  across OS/arch, and cannot honestly support "any language" plugins.
- **Subprocess + JSON-RPC over stdio.** Plugins are standalone executables
  the core spawns; language-agnostic by construction, easy to sandbox
  (OS process boundary), easy to debug (run the plugin binary directly),
  and works fully offline. The same shape as Terraform providers and
  Language Server Protocol servers.

## Decision

Core CLI plugins (templates and capabilities) are standalone executables,
discovered via a `plugin.json` manifest, and spawned as subprocesses. The
core and the plugin exchange **JSON-RPC 2.0 messages, one per line, over
the plugin's stdin/stdout**; stderr is reserved for plugin logs.

Methods:

- `plugin.initialize` — capability negotiation (protocol version, manifest).
- `plugin.generate` — templates only: given `targetDir`, `projectName`,
  `answers`, write the initial project; return `filesWritten` + `nextSteps`.
- `plugin.apply` — capabilities only: same input shape, mutate/add files in
  an existing target directory; return `filesWritten`/`filesModified` +
  `nextSteps`.
- `plugin.shutdown`.

Full manifest and message schemas: [plugin-protocol.md](../plugin-protocol.md).

## Consequences

- Line-delimited JSON-RPC is simpler to implement than LSP-style
  `Content-Length`-framed messages, at the cost of not being safely
  binary-transparent. Acceptable for V1 (all payloads are JSON-serializable
  file content); revisit if a future plugin needs to stream large binary
  assets.
- `sdk/go` owns the canonical wire-type definitions; `core` imports only
  that subpackage (not any plugin code), keeping the "core never imports
  plugins" boundary intact while avoiding client/server type drift.
- Discovery is local-only for V1 (a scanned directory of `plugin.json`
  files); a remote plugin registry is deferred (see
  [roadmap.md](../roadmap.md)) but the manifest/registry interface is
  designed to accommodate one later without a breaking change.
