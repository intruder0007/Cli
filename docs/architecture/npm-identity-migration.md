# npm identity migration — audit record

Date: 2026-08-02. Companion to [ADR-0016](adr/0016-npm-identity-migration.md).
Records what was found, what was fixed, what was verified, and what is
left until `lumo-cli@1.0.0` is live.

## Registry state (audit start)

| Package | State |
|---|---|
| `bootstrap-cli-dev@0.3.0` | live on npm (published 2026-08-02 under the old name, ADR-0015) |
| `lumo-cli` | **unclaimed** on npm — name available |
| `lumo` | taken (unrelated WebGL library) — why the wrapper is `lumo-cli` |

## Corruption findings and fixes

The wrapper family carried damage from an earlier bulk edit: single
letter substitutions (first-letter or in-word) plus stale prose and
stale "published" claims. All fixed in the tree:

| File | Finding | Fix |
|---|---|---|
| `distribution/npm/package.json` | description "aauncher for the aumo developer lumo platform… (github.com/intruder0007/aumo)"; homepage, bugs, repository all `intruder0007/aumo` | description → "Launcher for the Lumo project scaffolding platform… (github.com/intruder0007/Lumo)"; homepage/bugs/repository → `github.com/intruder0007/Lumo` |
| `distribution/npm/README.md` | 13 corrupted words ("hhen", "hhe" ×9, "hhree", "EBADPLAhFORM", `NPM_hOKEN` ×2); stale asset name `cli_v<version>_…`; false claims `lumo-cli@0.3.0` live/published; status section fiction | full rewrite: corrected words, `lumo_v<version>_…`, truthful status (interim deprecated, `lumo-cli` pending v1.0.0) |
| `distribution/cargo/Cargo.toml` | description "hhin launcher for the Lumo developer lumo platform"; `license = "MIh"` | description → "Thin launcher for the Lumo project scaffolding platform…"; `license = "MIT"` |
| `distribution/pypi/pyproject.toml` | same description corruption; `license = { text = "MIh" }` | same fixes |
| `distribution/homebrew/lumo.rb` | comment + `desc` "Cross-language developer lumo platform…" | → "Lumo project scaffolding platform" prose |
| `distribution/chocolatey/lumo.nuspec` | `description` "developer lumo platform" | same |
| `distribution/scoop/lumo.json` | `description` "developer lumo platform" | same |
| `distribution/winget/…/intruder0007.Lumo.locale.en-US.yaml` | `ShortDescription` "developer lumo platform" | same |
| `docs/architecture/roadmap.md`, `docs/guides/releasing.md`, `CHANGELOG.md`, `distribution/README.md` | claimed `lumo-cli@0.3.0` was published / live | corrected to: `bootstrap-cli-dev@0.3.0` live + deprecated; `lumo-cli@1.0.0` pending the v1.0.0 release |

Canonical prose applied everywhere: *"Lumo project scaffolding
platform"* (matching the root README title and `distribution/npm/README.md`).

Workflow files (`release.yml`, `distribution-verify.yml`,
`npm-verify-published.yml`) were checked for the same corruption —
none; they correctly reference `secrets.NPM_TOKEN`.

## Verification evidence

- Final corruption sweep (`hhe|hhen|hhree|hhin|MIh|aumo|aauncher|hOKEN|EBADPLAh|developer lumo` over the whole repo): **zero matches**.
- `npm pack --dry-run` (in `distribution/npm/`): exactly 5 files —
  README.md, `bin/lumo.js`, `package.json`, `scripts/postinstall.js`,
  `scripts/verify-release.js`; 6.0 kB packed, 15.4 kB unpacked. No
  stray files (no `.bin/`, no marker).
- `node scripts/verify-release.js`: exits 1 with all six
  `lumo_v1.0.0_*` + `SHA256SUMS.txt` URLs listed as HTTP 404 and the
  "cut the GitHub release first" fix message — the pre-publish guard
  provably blocks publishing before the v1.0.0 release exists.
- Sandbox install of the packed tarball (scripts enabled): postinstall
  downloads, hits the 404, prints the actionable error, `npm install`
  fails cleanly with exit 1 — a broken package cannot be installed.
- Sandbox install with `--ignore-scripts`: install succeeds; running
  the shim prints the actionable missing-binary error and exits 1.
- `bootstrap-cli-dev@0.3.0` deprecated via the registry API; final
  message verified with a cache-busted query.

## Remaining until `lumo-cli@1.0.0` is live (blocked on the release)

1. Cut the v1.0.0 GitHub release (`.github/workflows/release.yml`):
   assets `lumo_v1.0.0_{linux,darwin,win32}…` + `SHA256SUMS.txt`.
   This is the release decision at the end of the mission.
2. `publish-npm` job publishes `lumo-cli@1.0.0` with provenance.
3. Then run `.github/workflows/npm-verify-published.yml`
   (workflow_dispatch) for the clean-machine check on Linux + Windows —
   **not before**: it installs `lumo-cli@latest`, which does not exist
   until step 2.
4. Flip the status sections to "published" (npm README,
   `distribution/README.md`, `roadmap.md`, `releasing.md`).
