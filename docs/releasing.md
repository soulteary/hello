# Release guide

This checklist is for project maintainers. A release is created only from a
`v*` tag; pushing a branch never creates a GitHub Release.

## 1. Prepare the source

1. Start from an up-to-date `main` with a clean working tree.
2. Move completed entries from `Unreleased` into a dated version section in
   [`CHANGELOG.md`](CHANGELOG.md). The heading must use the exact form
   `## [X.Y.Z] - YYYY-MM-DD`; the release workflow extracts that section.
3. Check that both READMEs, the migration guide, animation reference,
   `LICENSE` and `NOTICE` match the code and packaged assets.
4. Confirm the Go version in `go.mod`, the Docker build stage and CI setup are
   aligned.

For a major-version release, add or update the corresponding migration guide.
For a patch or minor release, call out any intentionally changed defaults in
the changelog and both READMEs.

## 2. Run the local gates

Install the declared Go toolchain, golangci-lint and govulncheck, then run:

```bash
make check
make build
./hello -version
```

`make check` includes a non-mutating `go mod tidy -diff` check, formatting,
`go vet`, golangci-lint, govulncheck, race-enabled tests and the
statement-coverage floor. The default minimum is 100%; do not lower it to make
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
update local `main`, and choose the version once. Never tag a commit whose
message contains `[skip ci]`, `[ci skip]`, `[no ci]`, `[skip actions]`,
`[actions skip]` or a `skip-checks` trailer: tag pushes are `push` events and
those markers can suppress both delivery workflows.

Create a release tag on that exact commit. A signed tag is preferred when the
maintainer has a signing identity configured:

```bash
VERSION=2.2.0
git switch main
git pull --ff-only origin main
git log -1 --show-signature
git tag -s "v${VERSION}" -m "hello v${VERSION}"
git push origin "v${VERSION}"
```

Use `git tag -a` when signing is unavailable. Lightweight tags created with
`git tag "v${VERSION}"` or through the GitHub Release form are also accepted;
the workflows still require strict SemVer, a matching dated changelog section
and a target commit contained in `main`. Do not move or reuse a tag after
publication. If a release has a defect, document it on that release and prepare
a new patch version.

Repository administrators should also protect `v*` with a tag ruleset that
restricts updates and deletions, and enable release immutability. These settings
apply outside Actions and prevent an administrator or compromised credential
from replacing an already published tag or asset.

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
- image tags include `${VERSION}`, its minor and major aliases, and the expected
  registry names;
- the version-tag build, rather than an earlier `main` build, updates `latest`;
- `hello -version` reports `${VERSION}` in both a downloaded binary and
  container;
- curl, browser animation, `/events` and `/healthz` work from the published
  image.

## 5. Post-release

Add an empty `Unreleased` section for subsequent work if needed. Update any
downstream examples that pin the old image only after the immutable tag and
both registries have been verified. Follow the checksum and signature commands
in [`verification.md`](verification.md) from a clean machine.
