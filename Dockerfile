# Container image of the Arnakoff commands (#68, docs/standards.md, Supply chain).
#
# Two stages: the commands are built on the Go image, then copied alone onto
# distroless static, which has no shell, no package manager and runs as a
# non-root user. Both images are pinned by the digest of their multi-platform
# index; tags and digests checked on 03/11/2026 with
# `docker buildx imagetools inspect`. Dependabot proposes their updates.
#
#   docker build --build-arg VERSION=0.48.0 -t arnakoff/api .
#
# The image runs the api by default; `sync` and `admin` sit next to it:
#
#   docker run --entrypoint /usr/local/bin/sync arnakoff/api
#
# The root file system can be mounted read-only: the commands write only to
# /data (ARNAKOFF_DATA_DIR), the volume of the snapshots. See compose.yaml.

FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine3.24@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS build

# Static binaries without cgo, built by the toolchain of the image only:
# go.mod requires go1.27.3, the version of the image.
ENV CGO_ENABLED=0 GOTOOLCHAIN=local

WORKDIR /src
# The module cache mount keeps downloads between builds; go build fetches
# only the modules the three commands link, not the pinned tools of go.mod.
COPY go.mod go.sum ./
COPY cmd/ cmd/
COPY internal/ internal/

# The version the api and sync commands log at start (internal/platform/buildinfo).
ARG VERSION=dev
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
      -ldflags="-s -w -X github.com/arnakoff/api/internal/platform/buildinfo.version=${VERSION}" \
      -o /out/bin/ ./cmd/api ./cmd/sync ./cmd/admin \
 && mkdir -p /out/data/snapshots

FROM gcr.io/distroless/static-debian13:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329

ARG VERSION=dev
LABEL org.opencontainers.image.title="arnakoff-api" \
      org.opencontainers.image.description="API publique française anti-arnaque : signaux sourcés sur un site, un e-mail, un numéro, un IBAN ou un SIREN" \
      org.opencontainers.image.source="https://github.com/arnakoff/api" \
      org.opencontainers.image.licenses="AGPL-3.0-only" \
      org.opencontainers.image.version="${VERSION}"

COPY LICENSE /usr/share/doc/arnakoff/LICENSE
COPY --from=build /out/bin/ /usr/local/bin/
# Owned by the nonroot user of distroless (65532): a named volume mounted
# there starts as a copy of it, writable by the sync command.
COPY --from=build --chown=65532:65532 /out/data/ /data/

USER 65532:65532
ENV ARNAKOFF_DATA_DIR=/data
EXPOSE 8100

# No shell in the image: the api binary probes itself (cmd/api, "api probe").
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/usr/local/bin/api", "probe"]

ENTRYPOINT ["/usr/local/bin/api"]
