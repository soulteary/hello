# Verifying releases and images

Release archives carry SHA-256 checksums. Container images carry BuildKit SBOM
and provenance attestations and are signed with GitHub Actions' keyless OIDC
identity.

## Release archives

Download `checksums.txt` and the archive for the same version, then verify it
before extracting:

```bash
VERSION=2.2.0
ASSET="hello_${VERSION}_linux_amd64.tar.gz"
BASE="https://github.com/soulteary/hello/releases/download/v${VERSION}"

curl --fail --location --remote-name "${BASE}/checksums.txt"
curl --fail --location --remote-name "${BASE}/${ASSET}"
sha256sum --ignore-missing --check checksums.txt
tar -tzf "${ASSET}"
```

The archive must contain only the platform binary, `LICENSE` and `NOTICE`.
macOS can replace `sha256sum` with `shasum -a 256` and compare the selected
line from `checksums.txt`.

GitHub also stores signed build provenance for every uploaded archive and the
aggregate checksum file. With an authenticated GitHub CLI:

```bash
gh attestation verify "${ASSET}" --repo soulteary/hello
```

## Container signatures

Install [cosign](https://docs.sigstore.dev/cosign/system_config/installation/),
then verify either registry. Replace the version with the exact tag being
deployed:

```bash
IMAGE=ghcr.io/soulteary/hello:2.2.0

cosign verify \
  --certificate-identity-regexp '^https://github.com/soulteary/hello/.github/workflows/docker.yml@refs/tags/v.*$' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  "${IMAGE}"
```

The Docker Hub equivalent is `soulteary/hello:2.2.0`. Verification constrains
the signer to this repository's Docker workflow and GitHub Actions' OIDC
issuer; a signature from an unrelated workflow is not accepted.

After verification, resolve and record the immutable digest used by a
deployment:

```bash
docker pull "${IMAGE}"
docker image inspect --format '{{index .RepoDigests 0}}' "${IMAGE}"
```

Use the returned `name@sha256:...` value in production manifests when alias
movement is not acceptable.
