# PyPI wrapper (not implemented)

Follow the [wrapper protocol](../../docs/architecture/distribution-protocol.md).

A minimal Python package exposing a `console_scripts` entry point (e.g.
`bootstrap = bootstrap_cli.__main__:main`) whose `main()` resolves
platform via `platform.system()`/`platform.machine()`, locates/downloads
the matching release archive into a cache dir (`platformdirs`-style
location), and `os.execv`s the extracted `bootstrap` binary — `execv`
replaces the process image entirely, which is the cleanest way to
satisfy the protocol's stdio-passthrough requirement on POSIX. Windows
has no real `execv`; use `subprocess.run` with inherited handles and
`sys.exit(result.returncode)` instead.
