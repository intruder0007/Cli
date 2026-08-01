# SDK architecture

Implements [ADR-0014](adr/0014-sdk-foundation.md). This document is
the language-neutral design every SDK implementation must follow —
the **same developer experience in every language**. `sdk/go` is the
reference implementation (the only one shipped today); `sdk/node`,
`sdk/python`, `sdk/rust`, and `sdk/future` are design notes in this
repository waiting for an implementation phase. The wire protocol this
all implements is `docs/architecture/plugin-protocol.md`.

## The one idea

An SDK's job is to make "implement `Generate`/`Apply`, hand it to a
`Serve`, ship a binary + `plugin.json`" trivial in its language. It
must never invent behavior the protocol doesn't have, and it must
never hide the protocol's rules (identity cross-check, timeouts,
ordering) — an SDK that "helps" by silently doing something the
protocol doesn't specify is an SDK that breaks compatibility.

## Language-neutral abstractions

Every SDK exposes the same four concepts, by whatever names fit the
host language's conventions:

| Concept | Go reference | What it must do |
|---|---|---|
| **Manifest** | `sdk.Manifest` + `Validate()` | Mirror `plugin.json` field-for-field; validate required fields per `kind` before serving. |
| **Plugin** | `sdk.TemplatePlugin` (`Generate`), `sdk.CapabilityPlugin` (`Apply`) | The author's logic. Must be implementable with only the request structs and the filesystem. |
| **Requests/Responses** | `GenerateRequest/Response`, `ApplyRequest/Response` | Exact wire shapes: `targetDir`, `projectName`, `answers`, `filesWritten`, `filesModified`, `nextSteps`. |
| **Serve** | `sdk.Serve(plugin)` | The transport loop: read line-delimited JSON-RPC 2.0 from stdin, dispatch to the plugin, write responses to stdout, log to stderr, exit on `plugin.shutdown`/EOF. |

The request/response fields are **protocol constants**, not design
choices. The bindings below list them; an SDK must not rename,
reorder, or default any of them (Go's `json:"..."` tags are the
canonical names).

## Protocol bindings (exact, per SDK)

A new SDK implements, with zero assumptions beyond this list:

- Methods: `plugin.initialize`, `plugin.generate` (templates),
  `plugin.apply` (capabilities), `plugin.shutdown`.
- The initialize response must return the plugin's *own* loaded
  manifest (the host cross-checks it against the on-disk one — a
  misreport is a compatibility failure, not a helper's concern).
- `plugin.generate`/`plugin.apply` responses: `filesWritten` (both),
  `filesModified` (apply), `nextSteps` (both, optional).
- Framing: exactly one JSON object per line, LF-terminated; stderr is
  for human logs, never protocol traffic.
- Error responses use JSON-RPC 2.0 error objects; the SDK should map
  author errors to `-32000` (as `sdk/go` does) so the host's typed
  error handling works unchanged.

## Package layout

Each SDK lives under `sdk/<lang>/` and is its own distributable unit:

```text
sdk/go/      # the reference implementation; importable as a Go module
sdk/node/    # npm package (design note today)
sdk/python/  # PyPI package (design note today)
sdk/rust/    # crates.io package (design note today)
sdk/future/  # placeholder for languages added later
```

Rules:

1. Every SDK must be independently installable (its ecosystem's
   standard package manager) and must depend on nothing but that
   language's standard library — matching the project-wide
   "no dependency for something the stdlib does" convention
   (ADR-0007, `templates/node-rest-api`, `distribution/npm`).
2. Every SDK's `README.md` documents: how to implement a plugin,
   how to build it, and the minimum supported version of the SDK's
   own ecosystem.
3. An SDK is considered **done** only when its clean-machine install
   and `bootstrap new` round-trip is verified in CI — the same bar the
   distribution wrappers already meet (`distribution-verify.yml`).

## Transport layer

The transport is **not** per-SDK code to reinvent: it is the four
lines of behavior above (read line → dispatch → write line → exit).
An SDK should implement it in the fewest possible moving parts. The
reference (`sdk/go`) does it with a `bufio` loop and a dispatch
switch; an SDK that needs a framework or an async runtime to do this
is over-engineered for this protocol.

## Compatibility strategy

- The wire protocol is versioned by `protocolVersion` (today `"1"`).
  Every SDK binds to exactly one protocol version and reports it in
  its manifest; the host rejects mismatches at the handshake.
- SDKs are additive: a new SDK (or a new SDK feature) can never
  require a protocol change. If a feature seems to require one, the
  feature is wrong, not the protocol — see `api-compatibility.md`
  for the change procedure (it requires an ADR and a major version).
- SDKs keep the `docs/plugins/authoring.md` and
  `docs/templates/authoring.md` examples truthful: every SDK's
  tutorial must use the same manifest and the same protocol, so an
  author switching languages doesn't relearn semantics.
- `sdk/go` is the reference implementation: when in doubt about a
  behavior, the other SDKs match it, and `sdk/go`'s tests are the
  behavioral spec a new SDK's tests must reproduce.

## Version negotiation

- There is no negotiation today: the host speaks one protocol version
  and rejects others (`plugin.ProtocolMismatchError`). An SDK
  therefore hardcodes its `protocolVersion` and never negotiates.
- If a future protocol change needs multi-version serving, the design
  is: the host advertises the versions it supports in
  `plugin.initialize`'s request, the plugin picks the highest it
  understands, and both speak that one. Until then, no SDK implements
  any negotiation code — YAGNI, and the ADR-0013 rules apply to any
  protocol change anyway.

## What "identical developer experience" means

The same 20-line tutorial, translated, is the acceptance test:

```go
// Go (reference) — a capability plugin
type mine struct{}
func (mine) Apply(req sdk.ApplyRequest) (sdk.ApplyResponse, error) { /* ... */ }
func main() { sdk.Serve(mine{}) }
```

```js
// Node (design) — the same capability plugin
class Mine {
  apply(req) { /* ... */ }
}
serve(new Mine());
```

```python
# Python (design) — the same capability plugin
class Mine:
    def apply(self, req):
        ...
serve(Mine())
```

```rust
// Rust (design) — the same capability plugin
struct Mine;
impl CapabilityPlugin for Mine {
    fn apply(&self, req: ApplyRequest) -> Result<ApplyResponse, Error> {
        /* ... */
    }
}
fn main() {
    serve(Mine);
}
```

An author must be able to port a plugin between these with only the
language's syntax changing: same manifest, same fields, same
behaviors. Anything that breaks that equivalence is an SDK design
bug, not an implementation detail.
