# ADR-0005: MIT license

## Status

Accepted

## Context

The project is a long-term open source developer tool that depends on
third parties eventually authoring templates and plugins, including
closed-source ones. Candidates considered: MIT, Apache-2.0, GPL-3.0.
GPL-3.0's copyleft terms would pressure third-party plugin authors
(especially ones linking against `sdk/go`) to also open-source their
plugins, which works against "plugin-first" as an adoption strategy.
Apache-2.0 is comparable to MIT with an explicit patent grant, more common
for larger corporate-backed projects; MIT is simpler and is the most
common choice for developer tooling and plugin ecosystems the project is
modeling itself on (Terraform providers, LSP servers).

## Decision

The repository is licensed under MIT (`LICENSE`).

## Consequences

- Third parties can freely build closed-source templates/plugins against
  `sdk/go` without license obligations flowing back to their code.
- No explicit patent grant (unlike Apache-2.0) — acceptable for a
  developer-tooling project without significant patent exposure at this
  stage; revisit via a new ADR if that changes.
