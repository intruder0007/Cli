# Cargo wrapper (not implemented)

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).

Least natural fit of the eight — Cargo is source-first, and `bootstrap`
isn't Rust source. The honest options: (a) a `build.rs` that
downloads/verifies the matching release archive at `cargo install` time
and installs a tiny Rust shim binary that `std::process::Command`s the
real `bootstrap` with inherited stdio, or (b) skip Cargo and document Go
users installing via `go install` / a release archive instead. Worth
revisiting only if real demand shows up — lowest priority of the eight.
