## What & why

<!-- What does this PR change, and why? Link the issue/ADR if relevant. -->

## Target branch

- [ ] This PR targets the correct long-lived branch (a topic branch → its
      subsystem branch, or a subsystem branch → `main`). Direct pushes to
      `main` are not permitted; PRs into `main` should come from a
      subsystem branch (`architecture`, `cli`, `core`, `sdk`, `templates`,
      `plugins`, `tests`), not ad hoc.

## ADR

- [ ] N/A — no interface/boundary/protocol change
- [ ] Included in this PR: `docs/architecture/adr/____-____.md`

## Checklist

- [ ] `go vet ./...` and `go build ./...` pass for touched modules
- [ ] `go test ./...` passes for touched modules
- [ ] Docs updated if behavior or interfaces changed
- [ ] No secrets or credentials included
