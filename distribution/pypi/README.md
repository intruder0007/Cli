# PyPI wrapper — built, CI-verified, not published

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).

`pyproject.toml` (setuptools, stdlib-only — no third-party dependency)
exposing a `console_scripts` entry point (`lumo =
lumo_cli.__main__:main`) whose `main()` resolves platform via
`platform.system()`/`platform.machine()`, downloads/verifies (SHA256
against `SHA256SUMS.txt`) the matching release archive into a
version-scoped cache directory (skipping the download on repeat runs),
and hands off to the real binary — `os.execv` on POSIX (replaces the
process image, the cleanest way to satisfy stdio passthrough),
`subprocess.run` + `sys.exit(returncode)` on Windows (no real `execv`
there).

**Verified in CI** (`.github/workflows/distribution-verify.yml`, `pypi`
job, `ubuntu-latest` + `actions/setup-python`): `pip install
./distribution/pypi` followed by a real `lumo version` invocation
through the installed console script. Not verified locally in this
repo's own dev environment — Python isn't actually installed there
(only a Microsoft Store stub; same finding as ADR-0009).

**Not published.** `twine upload` needs a real PyPI API token, which
isn't available in the environment that built this. Package name
(`lumo-cli`) is a placeholder, not reserved or claimed anywhere.
