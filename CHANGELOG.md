# Changelog

All notable changes to this project are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versioning
follows [Semantic Versioning](https://semver.org/).

See [ADR-0006](docs/architecture/adr/0006-release-process.md) for the
release process this file is part of, and [CONTRIBUTING.md](CONTRIBUTING.md)
for how to add an entry.

## [Unreleased]

## [0.1.0] - TBD

Initial release. Interactive/non-interactive CLI wizard (theme, project
type, language, framework, capabilities), generating a Go REST API
backend service (`templates/go-rest-api`) with three capability plugins
(`git-init`, `readme`, `github-actions-ci`), over a subprocess +
line-delimited JSON-RPC 2.0 plugin protocol (`sdk/go`).

[Unreleased]: https://github.com/intruder0007/Cli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/intruder0007/Cli/releases/tag/v0.1.0
