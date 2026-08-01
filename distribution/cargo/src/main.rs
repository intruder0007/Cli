// Thin launcher only, per docs/architecture/distribution-protocol.md:
// never parses bootstrap's flags, never renders prompts. BOOTSTRAP_BIN_PATH
// is baked in at compile time by build.rs, pointing at the real binary
// build.rs already downloaded and verified.

use std::process::{exit, Command};

fn main() {
    let bin_path = env!("BOOTSTRAP_BIN_PATH");
    let args: Vec<String> = std::env::args().skip(1).collect();

    let status = Command::new(bin_path)
        .args(&args)
        .status()
        .unwrap_or_else(|e| {
            eprintln!("bootstrap-cli: failed to launch {bin_path}: {e}");
            exit(1);
        });

    exit(status.code().unwrap_or(1));
}
