# ADR-0014: SDK foundation — one language-neutral design, multiple bindings

## Status

Accepted

## Context

The project is entering its ecosystem phase. `sdk/go` is the only SDK
and it works well (it powers all five first-party plugins), but the
plugin protocol is language-agnostic by construction (ADR-0002) and
the roadmap names non-Go SDKs as additive future work. Two gaps exist
between "one Go SDK" and "an ecosystem": there is no written,
language-neutral definition of what an SDK *is* (the abstractions,
bindings, and behaviors are only implied by `sdk/go`'s source), and
there is no package layout or compatibility policy for the SDK family
an external author can rely on. Without the design document, the first
community-contributed SDK would reverse-engineer its contract from
`sdk/go`'s implementation — exactly what ADR-0010's distribution
contract exists to prevent for wrappers.

The mandate for this phase is explicit: define the SDK architecture
and future SDKs (node, python, rust, future), and **do not fully
implement them** — define interfaces, protocol bindings, package
layout, transport layer, compatibility strategy, and version
negotiation.

## Decision

1. **`docs/architecture/sdk-architecture.md` is the language-neutral
   SDK specification.** It defines the four abstractions every SDK
   exposes (Manifest, Plugin, Requests/Responses, Serve), the exact
   protocol bindings (methods, framing, error mapping), the package
   layout (`sdk/<lang>/`), the transport rules (fewest moving parts,
   stdlib-only), the compatibility strategy (SDKs are additive and
   can never require a protocol change; `sdk/go` is the reference
   implementation whose tests are the behavioral spec), and version
   negotiation (none today — the host speaks one protocol version;
   the future multi-version design is sketched but unimplemented per
   YAGNI).
2. **`sdk/go` is declared the reference implementation.** Its
   behavior, not just its docs, is the spec other SDKs match. It
   stays exactly as it is — Stable per ADR-0013, no code changes in
   this ADR.
3. **`sdk/{node,python,rust,future}/` get design-note READMEs** —
   the same pattern ADR-0010 used for `distribution/<ecosystem>/`:
   scaffolded, honest "not implemented yet" notes documenting each
   language's planned package name, stdlib-only constraint, and the
   binding work, so a future contributor has a precise starting point
   and the layout is stable from today.
4. **The acceptance test for "identical developer experience" is a
   translated 20-line tutorial** (defined in `sdk-architecture.md`):
   the same capability plugin in Go/Node/Python/Rust with only
   syntax changing. An author switching languages must not relearn
   semantics.

## Consequences

- A community contributor implementing `sdk/node` has a written
  spec, not an implementation to reverse-engineer — the same
  decoupling ADR-0010 gave the distribution wrappers.
- `sdk/go`'s tests implicitly become part of the SDK contract
  (the behavioral spec) — worth stating explicitly in its README in
  the implementation phase, without changing the tests themselves.
- The `sdk/` directory gains three empty-but-real subdirectories and
  one placeholder; `sdk/go` is untouched. No build, test, or release
  impact.
- Non-Go SDKs remain unbuilt (per mandate); the roadmap's existing
  "Non-Go plugin SDKs" entry is now backed by a written design
  instead of being an open question.
