// Runs at `cargo build`/`cargo install` time: resolves the target
// platform, downloads the matching release archive (skipping the
// download if a version-scoped cache already has it), verifies it
// against SHA256SUMS.txt, extracts the bootstrap binary, and bakes its
// final path into the compiled shim via cargo:rustc-env — read back by
// src/main.rs through env!("BOOTSTRAP_BIN_PATH"). See
// docs/architecture/distribution-protocol.md.

use std::env;
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;

const REPO: &str = "intruder0007/Cli";
const VERSION: &str = "v0.2.0"; // tracks this wrapper's own package version

fn target_pair() -> (String, String) {
    let os = env::var("CARGO_CFG_TARGET_OS").unwrap();
    let arch = env::var("CARGO_CFG_TARGET_ARCH").unwrap();
    let goos = match os.as_str() {
        "linux" => "linux",
        "macos" => "darwin",
        "windows" => "windows",
        other => panic!("bootstrap-cli: unsupported target OS: {other}"),
    };
    let goarch = match arch.as_str() {
        "x86_64" => "amd64",
        "aarch64" => "arm64",
        other => panic!("bootstrap-cli: unsupported target arch: {other}"),
    };
    (goos.to_string(), goarch.to_string())
}

fn cache_dir(goos: &str, goarch: &str) -> PathBuf {
    let base = if goos == "windows" {
        env::var("LOCALAPPDATA").expect("LOCALAPPDATA not set")
    } else {
        format!("{}/.cache", env::var("HOME").expect("HOME not set"))
    };
    PathBuf::from(base)
        .join("bootstrap-cli")
        .join(VERSION)
        .join(format!("{goos}_{goarch}"))
}

fn run(cmd: &mut Command) {
    let status = cmd.status().unwrap_or_else(|e| panic!("bootstrap-cli: failed to run {cmd:?}: {e}"));
    if !status.success() {
        panic!("bootstrap-cli: command failed: {cmd:?}");
    }
}

// Windows' `tar` can resolve to Git for Windows' GNU tar (MSYS), which
// misinterprets a bare drive-letter path like "C:\Users\..." as
// remote-host syntax — see distribution/npm/scripts/postinstall.js's
// identical comment. Invoke the OS-bundled bsdtar by full path instead.
fn tar_path() -> String {
    if cfg!(target_os = "windows") {
        let root = env::var("SystemRoot").unwrap_or_else(|_| "C:\\Windows".to_string());
        format!("{root}\\System32\\tar.exe")
    } else {
        "tar".to_string()
    }
}

fn verify_checksum(archive_name: &str, archive_path: &Path, sums_path: &Path) {
    let sums = fs::read_to_string(sums_path).expect("reading SHA256SUMS.txt");
    let expected = sums
        .lines()
        .find(|l| l.trim_end().ends_with(archive_name))
        .unwrap_or_else(|| panic!("bootstrap-cli: no checksum entry for {archive_name}"))
        .split_whitespace()
        .next()
        .unwrap()
        .to_lowercase();

    let actual = if cfg!(target_os = "windows") {
        let out = Command::new("certutil")
            .args(["-hashfile", archive_path.to_str().unwrap(), "SHA256"])
            .output()
            .expect("running certutil");
        String::from_utf8_lossy(&out.stdout)
            .lines()
            .nth(1)
            .expect("certutil output")
            .trim()
            .replace(' ', "")
            .to_lowercase()
    } else {
        let out = Command::new("shasum")
            .args(["-a", "256", archive_path.to_str().unwrap()])
            .output()
            .or_else(|_| Command::new("sha256sum").arg(archive_path).output())
            .expect("running shasum/sha256sum");
        String::from_utf8_lossy(&out.stdout)
            .split_whitespace()
            .next()
            .expect("shasum output")
            .to_lowercase()
    };

    if actual != expected {
        panic!("bootstrap-cli: checksum mismatch for {archive_name}: expected {expected}, got {actual}");
    }
}

fn main() {
    let (goos, goarch) = target_pair();
    let bin_name = if goos == "windows" { "bootstrap.exe" } else { "bootstrap" };
    let cache = cache_dir(&goos, &goarch);
    let bin_path = cache.join(bin_name);

    if !bin_path.exists() {
        let ext = if goos == "windows" { "zip" } else { "tar.gz" };
        let archive_name = format!("cli_{VERSION}_{goos}_{goarch}.{ext}");
        let base_url = format!("https://github.com/{REPO}/releases/download/{VERSION}");

        let tmp = env::temp_dir().join(format!("bootstrap-cli-build-{}", std::process::id()));
        fs::create_dir_all(&tmp).expect("creating temp dir");
        let archive_path = tmp.join(&archive_name);
        let sums_path = tmp.join("SHA256SUMS.txt");

        println!("cargo:warning=bootstrap-cli: downloading {archive_name} ({VERSION})...");
        run(Command::new("curl").args([
            "-fsSL", "-o", archive_path.to_str().unwrap(),
            &format!("{base_url}/{archive_name}"),
        ]));
        run(Command::new("curl").args([
            "-fsSL", "-o", sums_path.to_str().unwrap(),
            &format!("{base_url}/SHA256SUMS.txt"),
        ]));

        verify_checksum(&archive_name, &archive_path, &sums_path);

        run(Command::new(tar_path()).args([
            "-xf", archive_path.to_str().unwrap(),
            "-C", tmp.to_str().unwrap(),
        ]));

        let extracted_dir = tmp.join(format!("cli_{VERSION}_{goos}_{goarch}"));
        fs::create_dir_all(&cache).expect("creating cache dir");
        fs::copy(extracted_dir.join(bin_name), &bin_path).expect("copying binary");
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            fs::set_permissions(&bin_path, fs::Permissions::from_mode(0o755)).unwrap();
        }

        let _ = fs::remove_dir_all(&tmp);
    }

    println!("cargo:rustc-env=BOOTSTRAP_BIN_PATH={}", bin_path.display());
}
