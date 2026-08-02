# Cargo wrapper — built, CI-verified, not published

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).

Still the least natural fit of the 8 (Cargo is source-first, and
`lumo` isn't Rust source) — built anyway since it was in scope, but
flagged as the most likely to be dropped first if this project has to
prioritize.

`Cargo.toml` + `build.rs`, zero dependencies (not even
`build-dependencies`): `build.rs` runs at `cargo build`/`cargo install`
time, resolves the target platform from Cargo's own
`CARGO_CFG_TARGET_OS`/`CARGO_CFG_TARGET_ARCH` env vars, shells out to
`curl` to download the matching release archive + `SHA256SUMS.txt`
(Rust's stdlib has no HTTP client), shells out to `sha256sum`/`shasum`
(Unix) or `certutil` (Windows) to verify the checksum (no stdlib SHA256
either), shells out to `tar` to extract (same Windows bsdtar-by-full-path
fix as `distribution/npm/scripts/postinstall.js`), and bakes the final
binary path into the compiled shim via `cargo:rustc-env`. `src/main.rs`
reads it back through `env!("LUMO_BIN_PATH")` and execs it.

**Verified in CI** (`.github/workflows/distribution-verify.yml`, `cargo`
job, `ubuntu-latest`, Rust preinstalled): `cargo install --path
distribution/cargo` followed by a real `lumo version` invocation.
Not verified locally — Rust/Cargo aren't installed in this repo's own
dev environment.

**Not published.** `cargo publish` needs a real crates.io API token,
which isn't available in the environment that built this.
