# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Documentation for the `*.animation` file format (`docs/animation-format.md`).
- `-loops` negative-value validation and an upper bound check for `-delay`.
- `-list` now prints the animation `description` metadata next to each name.
- End-to-end CLI tests, a fuzz target for the animation parser, and a renderer
  benchmark.
- `golangci-lint` configuration and `make lint`, `make fuzz`, `make bench`
  targets.
- Cross-platform binary releases on `v*` tags (linux/darwin/windows,
  amd64/arm64) with SHA-256 checksums.
- Container image signing (cosign keyless) and a Docker build job on pull
  requests.
- `dependabot.yml` and a CodeQL workflow.
- Community health files: `CONTRIBUTING.md`, `SECURITY.md`,
  `CODE_OF_CONDUCT.md`, issue/PR templates.
- `.editorconfig`.

### Changed

- The runtime container image now uses `gcr.io/distroless/static-debian12:nonroot`
  instead of `scratch`, so the binary runs as a non-root user.

[Unreleased]: https://github.com/soulteary/hello/commits/main
