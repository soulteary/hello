# syntax=docker/dockerfile:1.7

# ---------- build stage ----------
# Pin a Go version that matches go.mod's `go 1.26.4` directive.
FROM --platform=$BUILDPLATFORM golang:1.26.4-alpine AS build

# TARGETOS/TARGETARCH are provided by buildx for multi-arch builds.
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

WORKDIR /src

# Cache module downloads in a separate layer. go.sum is optional today
# (no third-party deps yet) but copying via a glob keeps this future-proof.
COPY go.mod go.su[m] ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy only what the binary actually needs: sources + embedded animations.
# Tests, docs, snap config etc. are excluded via .dockerignore.
COPY *.go ./
COPY animations/ ./animations/

# Static, stripped, reproducible single binary. CGO is disabled so the
# resulting ELF has no glibc/musl dependency and runs on `scratch`.
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
        -trimpath \
        -buildvcs=false \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o /out/hello .

# ---------- runtime stage ----------
# distroless/static:nonroot is a tiny (~2 MB) base that ships CA certs, tzdata
# and a non-root user (uid/gid 65532). The binary is fully static (CGO off),
# so `static` is sufficient and the image runs unprivileged out of the box.
FROM gcr.io/distroless/static-debian12:nonroot

ARG VERSION=dev
ARG REVISION=unknown
ARG CREATED

LABEL org.opencontainers.image.title="hello" \
      org.opencontainers.image.description="Drop-in replacement for hello-world, with a party parrot." \
      org.opencontainers.image.url="https://github.com/soulteary/hello" \
      org.opencontainers.image.source="https://github.com/soulteary/hello" \
      org.opencontainers.image.documentation="https://github.com/soulteary/hello#readme" \
      org.opencontainers.image.authors="soulteary <soulteary@gmail.com>" \
      org.opencontainers.image.vendor="soulteary" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.created="${CREATED}"

COPY --from=build /out/hello /usr/local/bin/hello

# distroless:nonroot already runs as 65532, but make it explicit.
USER nonroot:nonroot
STOPSIGNAL SIGTERM

ENTRYPOINT ["/usr/local/bin/hello"]
