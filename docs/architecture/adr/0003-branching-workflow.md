# ADR-0003: Single repository, dedicated long-lived subsystem branches, no direct commits to main

## Status

Accepted

## Context

The project spans several independently evolvable subsystems (CLI, engine,
SDK, templates, plugins, tests) plus architecture/design docs. Options
considered: multiple repositories (one per subsystem) vs. one repository.
Multiple repos would give the strongest boundary enforcement, but adds real
overhead for a young project (cross-repo versioning, CI, issue linking) and
makes it harder to land a coherent V1 across subsystems at once. A single
repository with disciplined branch ownership gets most of the boundary
benefit (see ADR-0001's module-per-subsystem structure, which is enforced
by the Go compiler regardless of repo layout) without the multi-repo
overhead, and can still be split later if a subsystem outgrows the
monorepo.

## Decision

One Git repository. Eight long-lived branches: `main`, `architecture`,
`cli`, `core`, `templates`, `plugins`, `sdk`, `tests` — one per subsystem
plus `main` as the integration/release branch. Docs live in-repo under
`docs/`, versioned alongside the code they describe (no separate docs
branch or repo).

**No branch ever commits directly to `main`.** Changes land on the
relevant subsystem branch (directly, or via a short-lived topic branch off
it), then that subsystem branch is merged into `main` via pull request.

## Consequences

- `main` history is a sequence of subsystem-branch merges, making it easy
  to see what shipped and when.
- Contributors working on one subsystem don't need to touch others'
  branches; merge conflicts are contained to genuine cross-subsystem
  changes (which should be rare given the compiler-enforced module
  boundaries).
- Enforcement today is convention (`CONTRIBUTING.md`) plus, where the
  repository owner's permissions allow it, GitHub branch protection on
  `main` (required PR, no direct pushes). If protection can't be applied
  (insufficient admin rights at setup time), it's a manual follow-up for
  the repo owner, not a blocker for V1.
