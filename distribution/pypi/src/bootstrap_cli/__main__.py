"""Thin launcher only, per docs/architecture/distribution-protocol.md:
never parses bootstrap's flags, never renders prompts. Resolves the
current platform, downloads the matching release archive (caching it in
a version-scoped directory so repeat runs skip the download), verifies
it against SHA256SUMS.txt, then execs the real binary with stdio
inherited and argv/exit code forwarded exactly. Zero third-party
dependencies — stdlib only (urllib, hashlib, tarfile/zipfile), matching
the project-wide "don't add a dependency for something the standard
library already does" convention (see templates/node-rest-api,
plugins/builtin/git-init, distribution/npm's postinstall.js).
"""

import hashlib
import os
import platform
import shutil
import subprocess
import sys
import tarfile
import tempfile
import urllib.request
import zipfile

REPO = "intruder0007/Cli"
VERSION = "v0.2.0"  # tracks this wrapper's own package version


def _platform_target():
    goos = {"Linux": "linux", "Darwin": "darwin", "Windows": "windows"}.get(platform.system())
    machine = platform.machine().lower()
    goarch = {"x86_64": "amd64", "amd64": "amd64", "arm64": "arm64", "aarch64": "arm64"}.get(machine)
    if not goos or not goarch:
        sys.exit(f"bootstrap-cli: unsupported platform: {platform.system()}/{platform.machine()}")
    return goos, goarch


def _cache_dir(goos, goarch):
    base = os.environ.get("LOCALAPPDATA") if goos == "windows" else os.path.expanduser("~/.cache")
    return os.path.join(base or os.path.expanduser("~"), "bootstrap-cli", VERSION, f"{goos}_{goarch}")


def _download(url, dest):
    with urllib.request.urlopen(url) as resp, open(dest, "wb") as f:
        shutil.copyfileobj(resp, f)


def _verify_checksum(archive_name, archive_path, sums_path):
    with open(sums_path, "r", encoding="utf-8") as f:
        line = next((l for l in f if l.strip().endswith(archive_name)), None)
    if line is None:
        sys.exit(f"bootstrap-cli: no checksum entry for {archive_name} in SHA256SUMS.txt")
    expected = line.split()[0]
    digest = hashlib.sha256()
    with open(archive_path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            digest.update(chunk)
    actual = digest.hexdigest()
    if actual != expected:
        sys.exit(f"bootstrap-cli: checksum mismatch for {archive_name}: expected {expected}, got {actual}")


def _extract(archive_path, dest_dir, is_zip):
    if is_zip:
        with zipfile.ZipFile(archive_path) as zf:
            zf.extractall(dest_dir)
    else:
        with tarfile.open(archive_path) as tf:
            tf.extractall(dest_dir)


def _ensure_binary(goos, goarch):
    bin_name = "bootstrap.exe" if goos == "windows" else "bootstrap"
    cache_dir = _cache_dir(goos, goarch)
    bin_path = os.path.join(cache_dir, bin_name)
    if os.path.exists(bin_path):
        return bin_path

    is_zip = goos == "windows"
    ext = "zip" if is_zip else "tar.gz"
    archive_name = f"cli_{VERSION}_{goos}_{goarch}.{ext}"
    base_url = f"https://github.com/{REPO}/releases/download/{VERSION}"

    with tempfile.TemporaryDirectory(prefix="bootstrap-cli-") as tmp:
        archive_path = os.path.join(tmp, archive_name)
        sums_path = os.path.join(tmp, "SHA256SUMS.txt")
        print(f"bootstrap-cli: downloading {archive_name} ({VERSION})...", file=sys.stderr)
        _download(f"{base_url}/{archive_name}", archive_path)
        _download(f"{base_url}/SHA256SUMS.txt", sums_path)

        print("bootstrap-cli: verifying checksum...", file=sys.stderr)
        _verify_checksum(archive_name, archive_path, sums_path)

        print("bootstrap-cli: extracting...", file=sys.stderr)
        _extract(archive_path, tmp, is_zip)

        extracted_dir = os.path.join(tmp, f"cli_{VERSION}_{goos}_{goarch}")
        os.makedirs(cache_dir, exist_ok=True)
        shutil.copyfile(os.path.join(extracted_dir, bin_name), bin_path)
        if goos != "windows":
            os.chmod(bin_path, 0o755)

    return bin_path


def main():
    goos, goarch = _platform_target()
    bin_path = _ensure_binary(goos, goarch)
    args = [bin_path] + sys.argv[1:]

    if goos == "windows":
        # No real execv on Windows; run as a child with inherited
        # handles (stdio: inherited by default in subprocess.run) and
        # exit with its exact code.
        result = subprocess.run(args)
        sys.exit(result.returncode)
    else:
        # execv replaces this process image entirely — the cleanest way
        # to satisfy the protocol's stdio-passthrough requirement.
        os.execv(bin_path, args)


if __name__ == "__main__":
    main()
