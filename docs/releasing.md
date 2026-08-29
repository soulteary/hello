# Release guide

This checklist is for project maintainers. A release is created only from a
`v*` tag; pushing a branch never creates a GitHub Release.

## 1. Prepare the source

1. Start from an up-to-date `main` with a clean working tree.
2. Move completed entries from `Unreleased` into a dated version section in
   [`CHANGELOG.md`](CHANGELOG.md). The heading must use the exact form
   `## [2.0.0] - YYYY-MM-DD`; the release workflow extracts that section.
3. Check that both READMEs, the migration guide, animation reference,
   `LICENSE` and `NOTICE` match the code and packaged assets.
4. Confirm the Go version in `go.mod`, the Docker build stage and CI setup are
   aligned.

For 2.0.0, review [`migration-v2.md`](migration-v2.md) as part of the release
PR because the HTTP root response is intentionally incompatible with 1.x body
parsers.

## 2. Run the local gates

Install the declared Go toolchain, golangci-lint and govulncheck, then run:

```bash
make check
make build
./hello -version
```

`make check` includes a non-mutating `go mod tidy -diff` check, formatting,
`go vet`, golangci-lint, govulncheck, race-enabled tests and the
statement-coverage floor. The default minimum is 90%; do not lower it to make
a release pass.

Build and smoke-test the container when Docker is available:

```bash
make docker DOCKER_TAG=release-candidate
docker run --rm soulteary/hello:release-candidate -loops 1
docker run --rm -p 8080:8080 soulteary/hello:release-candidate -listen :8080
```

In another terminal, verify:

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/
curl --fail 'http://127.0.0.1:8080/?format=html'
```

Open the root URL in a browser and confirm that frames continue changing for
more than ten seconds; this catches proxy or server write-timeout regressions.

## 3. Merge and tag

Wait for the release-preparation PR and all required checks to pass. Merge it,
update local `main`, and create an annotated or signed tag on that exact commit:

```bash
git switch main
git pull --ff-only origin main
git tag -s v2.0.0 -m "hello v2.0.0"
git push origin v2.0.0
```

Do not move or reuse a published version tag. If a release has a defect,
prepare a new patch version.

## 4. Verify automation

The tag starts two independent delivery paths:

- `release.yml` verifies the tagged source, builds six OS/architecture
  archives, aggregates SHA-256 checksums and creates the GitHub Release from
  the matching changelog section.
- `docker.yml` builds linux/amd64 and linux/arm64 images for Docker Hub and
  GHCR, attaches SBOM/provenance attestations and signs the resulting digest
  with keyless cosign.

Confirm all jobs are green, then inspect the published result:

- release notes are the intended changelog section, not the generic fallback;
- Linux amd64/arm64, macOS amd64/arm64 and Windows amd64/arm64 archives exist;
- every archive contains the binary, `LICENSE` and `NOTICE`;
- `checksums.txt` covers every archive and verifies locally;
- image tags include `2.0.0`, `2.0`, `2` and the expected registry names;
- the version-tag build, rather than an earlier `main` build, updates `latest`;
- `hello -version` reports `2.0.0` in both a downloaded binary and container;
- curl, browser animation, `/events` and `/healthz` work from the published
  image.

## 5. Post-release

Add an empty `Unreleased` section for subsequent work if needed. Update any
downstream examples that pin the old image only after the immutable 2.0.0 tag
and both registries have been verified.
