# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.0.0] - 2026-08-29

### Added

- Content-negotiated HTTP root: curl and other non-browser clients receive one
  raw ASCII animation frame plus safe request diagnostics, while browsers
  receive a responsive terminal-style HTML page.
- `/events` Server-Sent Events endpoint that pushes animation frames and color
  changes to the browser, reconnects automatically and stops on client
  cancellation.
- HTTP support for `-animation`, `-delay` and `-mono`, including explicit
  `?format=html` / `?format=text` response overrides.
- Optional `-listen` HTTP mode with `/healthz`, graceful shutdown and bounded
  request/header limits.
- End-to-end HTTP content-negotiation tests, stream lifecycle/error tests,
  command-dispatch tests, parser edge cases and inventory atomicity tests.
- A 90% statement-coverage gate; the 2.0.0 source is above 95% under
  race-enabled tests.
- Reachability-aware `govulncheck` gates in local development, pull requests
  and tagged releases.
- Animation format reference, v2 migration guide and maintainer release guide.

### Changed

- HTTP streaming uses per-write deadlines instead of a global `WriteTimeout`
  that would terminate animations after ten seconds.
- Reflected proxy headers now use an explicit identity/routing allowlist;
  credential-like `X-Auth-*` and `X-Forwarded-*` fields are no longer accepted
  by prefix alone.
- The command parser now returns normal exit codes for help and parse failures,
  rejects extra positional arguments, and validates options at both dispatch
  and CLI package boundaries.
- `-version` falls back to Go module build information, so binaries installed
  with `go install ...@version` report that module version instead of `dev`.
- Animation parsing normalizes LF/CRLF input, rejects bare carriage returns,
  ignores empty metadata keys and reports one-based empty-frame numbers.
- Inventory loading is atomic and rejects a nil destination instead of
  panicking during map assignment.
- CI and release actions were upgraded and pinned to immutable commit SHAs;
  golangci-lint is now a required gate rather than an optional local warning.
- Tagged releases run formatting, vet, lint, race tests and coverage checks
  before any artifact is built.
- The mutable `latest` container tag is now updated only by a version tag;
  ordinary `main` pushes publish the `main` tag without pre-releasing changes.
- English and Chinese documentation now describe terminal behavior, HTTP
  negotiation, SSE proxy requirements, security boundaries, testing and
  release operations in full.

### Fixed

- SSE capability detection now follows middleware `Unwrap` chains instead of
  rejecting a compatible wrapped response writer.
- The documented `go install` path now targets `cmd/hello`, where the main
  package lives in the modern project layout.
- Release archives now include `LICENSE` and `NOTICE`; container images retain
  them under `/usr/share/licenses/hello/`.
- Windows arm64 artifacts are now built, matching the documented platform
  matrix.
- Dockerfile comments and project descriptions no longer claim a `scratch`
  runtime or exact `docker/hello-world` drop-in behavior.
- The release workflow now uses the maintained changelog section for 2.0.0
  instead of falling back to a generic message.

### Security

- The build toolchain was raised from Go 1.26.4 to 1.26.6 to include standard
  library fixes reported by `govulncheck` for `html/template`, `net/http`,
  `crypto/tls` and `encoding/asn1`.
- HTML responses include a restrictive Content Security Policy and standard
  anti-framing, MIME-sniffing, referrer and browser-permission headers.
- Authorization, cookie, token and unrecognized proxy headers are excluded
  from plain-text diagnostics.
- Third-party GitHub Actions are pinned to reviewed full commit SHAs.

[Unreleased]: https://github.com/soulteary/hello/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/soulteary/hello/compare/v1.0.24...v2.0.0
