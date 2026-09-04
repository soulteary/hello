# syntax=docker/dockerfile:1.7@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e

# ---------- build stage ----------
# Pin a Go version that matches go.mod's `go 1.27.0` directive.
FROM --platform=$BUILDPLATFORM golang:1.27.0-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS build

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

# Copy the full module: sources, embedded assets and go files. Tests, docs,
# local artifacts and CI are excluded via .dockerignore.
COPY . .

# Static, stripped, reproducible single binary. CGO is disabled so the
# resulting ELF has no glibc/musl dependency and runs in the distroless image.
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
        -trimpath \
        -buildvcs=false \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o /out/hello ./cmd/hello

# ---------- runtime stage ----------
# distroless/static:nonroot is a tiny (~2 MB) base that ships CA certs, tzdata
# and a non-root user (uid/gid 65532). The binary is fully static (CGO off),
# so `static` is sufficient and the image runs unprivileged out of the box.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

ARG VERSION=dev
ARG REVISION=unknown
ARG CREATED

LABEL org.opencontainers.image.title="hello" \
      org.opencontainers.image.description="A tiny terminal and HTTP demo with an animated ASCII parrot." \
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
COPY --from=build /src/LICENSE /usr/share/licenses/hello/LICENSE
COPY --from=build /src/NOTICE /usr/share/licenses/hello/NOTICE

# Optional HTTP mode listens on this unprivileged port when the container is
# started with `-listen :8080`. The default remains the terminal animation.
EXPOSE 8080

# distroless:nonroot already runs as 65532, but make it explicit.
USER nonroot:nonroot
STOPSIGNAL SIGTERM

ENTRYPOINT ["/usr/local/bin/hello"]
