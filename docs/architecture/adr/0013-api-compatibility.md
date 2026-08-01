# ADR-0013: Public API compatibility policy — lifecycle stages, semver rules, stability guarantees

## Status

Accepted

## Context

The project is entering its ecosystem phase: external developers are
expected to build on top of it — plugin authors, template authors,
distribution-wrapper maintainers, and future SDK users. Until now the
project has had de-facto contracts (`plugin-protocol.md`, `sdk/go`,
`distribution-protocol.md`, `docs/cli/usage.md`) but no explicit
statement of *which* of them are public, *how stable* each one is, and
*what versioning rules* bind changes to them. ADR-0006 defines release
versioning (semver tags, when to cut) but not API-level guarantees.
Without a policy, every change to a contract is a judgment call, and
external contributors have no way to know what they can rely on —
which is exactly the kind of ambiguity that makes an ecosystem hard to
adopt.

A concrete example of the ambiguity: the Theme API
(`cli/internal/prompt`'s `RegisterTheme`) is described in ADR-0007 as
"the seam a future theme-plugin mechanism would call into" — but it
lives in an `internal/` package, so no external code can import it.
That's a legitimate design (the seam is for the CLI team until a
theme-plugin mechanism exists), but it needs to be *stated*, so nobody
treats it as a promised public API.

## Decision

Adopt the policy in `docs/architecture/api-compatibility.md`:

1. **A public API inventory** — an explicit table of every surface
   that exists, where it lives, and its stability stage. Four surfaces
   are declared **Stable**: the plugin wire protocol,
   `sdk/go`, the CLI command surface, and the distribution wrapper
   contract. The Theme API and the `core` packages are declared
   **Experimental (internal)**: reachable in source and used by the
   CLI, but not promised to external contributors yet. Anything under
   `cli/internal/` is **internal by definition** (Go's own `internal`
   rule already enforces this).
2. **Lifecycle stages** — Experimental → Stable → Deprecated →
   Removed, with the transition rules. Promotion or demotion of a
   surface requires an ADR; deprecations and removals are announced in
   `CHANGELOG.md` and keep working for at least one full minor release.
3. **Semver rules binding releases to surfaces** — a breaking change
   to a Stable surface requires a major version (post-`v1.0.0`); while
   at `v0.x`, it still bumps the minor version (common Go ecosystem
   convention) and is called out in the changelog. Additive changes
   need no ADR; breaking ones do.
4. **Wire-protocol versioning** — `protocolVersion`/`sdk.ProtocolVersion`
   stay single-version per release; a mismatch is rejected at the
   handshake (existing ADR-0008 behavior, now *stated as policy*); a
   protocol change is always breaking and follows the same deprecation
   rules. Multi-version negotiation remains deferred (`roadmap.md`).

## Consequences

- External contributors get a documented, reviewable contract: what
  they can build against (the four Stable surfaces) and what they
  can't yet (Theme API, `core` internals).
- `sdk/go`'s exported API is now a documented commitment — its
  stability is stated in its package doc, and changes to it are
  governed by the semver rules above.
- The Theme API and `core` packages are *honestly* labeled
  Experimental: they can be promoted to Stable later via a new ADR
  (e.g. when a theme-plugin mechanism actually exists), without that
  promotion being a surprise.
- No code behavior changes in this ADR — it is a documentation and
  process decision. The existing handshake cross-checks (ADR-0008),
  versioning (ADR-0006), and module boundaries (ADR-0001/0002/0003)
  are unchanged and now have explicit policy on top.
