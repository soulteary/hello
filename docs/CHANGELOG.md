# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- The race-enabled statement-coverage floor is now 100% for local checks,
  pull requests and tagged releases.

### Fixed

- Query reflection now also controls `X-Forwarded-Uri`, preventing proxy
  headers containing sensitive query parameters from bypassing the default
  diagnostic redaction.
- Release validation rejects numeric SemVer prerelease identifiers with
  leading zeroes, such as `v2.2.1-01`.

## [2.2.0] - 2026-09-04

### Added

- A configurable `-http-max-streams` limit for concurrent SSE clients, with a
  bounded default of 64 and a retryable `503 Service Unavailable` response
  when the limit is full.
- SSE keepalive comments, browser pause/resume controls, reduced-motion
  support and automatic stream suspension while the page is hidden.
- Explicit `-reflect-query`, `-reflect-identity` and `-reflect-hostname`
  controls for HTTP diagnostics.
- Container smoke tests covering the version, OCI labels, non-root user,
  health endpoint, browser page and default diagnostic redaction.
- Release-archive checks for filenames, license notices, checksums and the
  executable's stamped version.
- Signed GitHub build-provenance attestations for release archives and the
  aggregate checksum file.
- Deployment recipes for Docker Compose, Nginx, Traefik and Kubernetes, plus
  container-signature and release-checksum verification instructions.

### Changed

- URL query strings and identity-bearing proxy headers are no longer reflected
  by default. Operators can enable them explicitly when the echo behavior is
  required in a controlled test environment.
- Browser HTML now uses per-response CSP nonces instead of allowing arbitrary
  inline scripts and styles.
- Docker pull-request and manual runs build without registry write or OIDC
  permissions; only push-triggered publish jobs receive those permissions.
- Manual Docker workflow runs no longer publish images. Release tags must be
  valid SemVer, point to a commit contained in `main`, use an annotated or
  signed tag object, and have a dated changelog section before images or
  archives are published.
- Go Report Card updates no longer use a commit-message marker that suppresses
  push-triggered release workflows.
- The Dockerfile frontend, Go builder and distroless runtime images are pinned
  by digest while retaining readable tags for automated dependency updates.

### Fixed

- Terminal rendering now propagates output and short-write failures instead of
  continuing an infinite animation after a pipe or output destination closes.
- Release and container workflows now verify the artifacts they claim to
  publish and verify keyless container signatures immediately after signing.

### Security

- Identity headers and raw query strings require explicit opt-in before they
  can appear in diagnostics.
- Long-lived SSE connections are bounded and emit low-cost heartbeats.
- Release validation is applied independently in both binary and container
  publication paths so a malformed tag cannot update `latest`.

## [2.1.0] - 2026-08-30

### Changed

- Simplified end-to-end content-negotiation, animation-parser and SSE stream
  tests without changing runtime behavior.

### Known issues

- The GitHub Release was created without binary archives or checksums and its
  tag did not run the release or container workflows. Use `v2.0.0` until a
  later complete release is available.

## [2.0.0] - 2026-08-30

### Added

- Content-negotiated HTTP root: curl and other non-browser clients receive one
  randomly selected raw ASCII animation frame plus safe request diagnostics,
  while browsers receive a responsive terminal-style HTML page. Distinct text
  frames are cached once per handler for efficient concurrent requests.
- Canonical project URL discovery through the browser footer, the final
  plain-text response line and the root endpoint's `Project` response header.
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

- The build toolchain was raised from Go 1.26.4 to 1.27.0 to include standard
  library fixes reported by `govulncheck` for `html/template`, `net/http`,
  `crypto/tls` and `encoding/asn1`.
- HTML responses include a restrictive Content Security Policy and standard
  anti-framing, MIME-sniffing, referrer and browser-permission headers.
- Authorization, cookie, token and unrecognized proxy headers are excluded
  from plain-text diagnostics.
- Third-party GitHub Actions are pinned to reviewed full commit SHAs.

[Unreleased]: https://github.com/soulteary/hello/compare/v2.2.0...HEAD
[2.2.0]: https://github.com/soulteary/hello/compare/v2.1.0...v2.2.0
[2.1.0]: https://github.com/soulteary/hello/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/soulteary/hello/compare/v1.0.24...v2.0.0
