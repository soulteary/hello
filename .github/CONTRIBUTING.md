# Contributing

Thanks for your interest in improving `hello`! This is a small, single-module
Go project with no third-party dependencies, so the workflow is deliberately
lightweight.

## Development workflow

```bash
make build   # build the ./hello binary
make test    # run tests with the race detector
make check   # tidy + formatting + vet + lint + vuln + race/coverage gates
```

`make check` is the CI-equivalent gate. `golangci-lint` and `govulncheck` are
required locally and in CI; install them before running the gate:
<https://golangci-lint.run/welcome/install/>.

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

Optional extras:

```bash
make fuzz    # fuzz the animation parser for 30s
make bench   # run benchmarks
make cover   # run tests and enforce the default 90% coverage floor
make vuln    # scan reachable code and dependencies
```

New behavior needs tests at the narrowest useful level. HTTP changes should
cover content type, status/method contracts, security headers, cancellation and
streaming errors where relevant. Terminal rendering changes must cover output
failures as well as successful buffers. Do not weaken `COVERAGE_MIN` to make a
change pass.

## Commit messages

Keep them short and in the imperative mood, matching the existing history
(e.g. "Add coffee animation", "Fix delay validation"). One logical change per
commit where practical.

## Adding a new animation

1. Create `internal/animation/assets/animations/<name>.animation` following the format described in
   [`docs/animation-format.md`](../docs/animation-format.md): a metadata header,
   then at least two non-empty frames separated by `!--FRAME--!`.
2. Include a `description:` metadata line — it shows up in `hello -list`.
3. If the artwork is your own original work, add `author:` and
   `license:` metadata. If it comes from elsewhere, add `source:` and make sure
   you have the right to redistribute it.
4. Update [`NOTICE`](../NOTICE) with attribution for any third-party artwork.
5. Update the animations table in both `README.md` and `README.zh-CN.md`.
6. Run `make check` — the inventory tests assert that bundled animations load.

## Changing HTTP behavior

The root endpoint is consumed by both terminals and browsers. Preserve the
documented content-negotiation order, keep `/healthz` small and non-streaming,
and avoid reflecting new headers unless they are demonstrably non-sensitive.
Any `/events` change must be exercised through a real `httptest.Server`, not
only a response recorder, so flush and cancellation behavior are covered.

Changes that alter a documented HTTP body, endpoint, flag interaction or
header allowlist must update both READMEs, `docs/CHANGELOG.md` and the migration
guide when compatibility is affected. Deployment-facing changes should also
update `docs/deployment.md`; signing or packaging changes should update
`docs/verification.md`.

## Releases

Only maintainers create tags. Release preparation and post-publish checks are
documented in [`docs/releasing.md`](../docs/releasing.md).

## Reporting bugs and requesting features

Please use the issue templates. For security-sensitive reports, follow
[`SECURITY.md`](SECURITY.md) instead of opening a public issue.
