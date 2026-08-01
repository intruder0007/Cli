# Future SDKs — placeholder

Follow the [SDK architecture spec](../../docs/architecture/sdk-architecture.md)
(ADR-0014) — the language-neutral contract every SDK implements, and
the [plugin protocol](../../docs/architecture/plugin-protocol.md) it
binds to.

Status: **not implemented.** This directory exists so the `sdk/` layout
is stable from today (ADR-0014). When a new language SDK is scoped, it
gets its own `sdk/<lang>/` directory with a design-note README in the
same shape as `sdk/node/`, `sdk/python/`, and `sdk/rust/` — before any
implementation work starts.

## When a language is eligible

- The wire protocol can be implemented in its standard library alone
  (line-delimited JSON-RPC over stdio) — per the project-wide
  stdlib-only convention.
- The language produces a standalone executable the host can spawn
  (the transport is subprocess-based, ADR-0002 — a language that can't
  produce a spawnable binary can't be a plugin language).
- Someone is actually building it: no design notes for languages
  nobody's implementing (YAGNI).
