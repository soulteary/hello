# Contributing

Thanks for your interest in improving `hello`! This is a small, single-module
Go project with no third-party dependencies, so the workflow is deliberately
lightweight.

## Development workflow

```bash
make build   # build the ./hello binary
make test    # run tests with the race detector
make check   # gofmt + vet + lint + tests (run this before opening a PR)
```

`make check` is the CI-equivalent gate. If you do not have `golangci-lint`
installed locally, `make lint` is skipped with a warning, but it still runs in
CI, so please install it for a complete local check:
<https://golangci-lint.run/welcome/install/>.

Optional extras:

```bash
make fuzz    # fuzz the animation parser for 30s
make bench   # run benchmarks
make cover   # run tests and print a coverage summary
```

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

## Reporting bugs and requesting features

Please use the issue templates. For security-sensitive reports, follow
[`SECURITY.md`](SECURITY.md) instead of opening a public issue.
