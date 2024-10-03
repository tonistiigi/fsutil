# syntax=docker/dockerfile:1

ARG GO_VERSION=1.23
ARG PROTOC_VERSION=3.11.4

FROM golang:${GO_VERSION} AS base

FROM base AS protoc
RUN apt-get update && apt-get --no-install-recommends install -y unzip
ARG PROTOC_VERSION
ARG TARGETOS
ARG TARGETARCH
RUN <<EOT
  set -e
  arch=$(echo $TARGETARCH | sed -e s/amd64/x86_64/ -e s/arm64/aarch_64/)
  wget -q https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-${TARGETOS}-${arch}.zip
  unzip protoc-${PROTOC_VERSION}-${TARGETOS}-${arch}.zip -d /opt/protoc
  rm -f /opt/protoc/readme.md
EOT

FROM base AS protoc-libs
WORKDIR /app
RUN --mount=type=bind,source=go.mod,target=/app/go.mod \
    --mount=type=bind,source=go.sum,target=/app/go.sum \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod <<EOT
  set -e
  mkdir -p /opt/protoc
  go mod download github.com/planetscale/vtprotobuf
  cp -r $(go list -m -f='{{.Dir}}' github.com/planetscale/vtprotobuf)/include /opt/protoc
EOT

FROM base AS tools
WORKDIR /app
RUN --mount=type=bind,source=go.mod,target=/app/go.mod \
    --mount=type=bind,source=go.sum,target=/app/go.sum \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
  go install \
    github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto \
    google.golang.org/protobuf/cmd/protoc-gen-go
COPY --link --from=protoc /opt/protoc /usr/local
COPY --link --from=protoc-libs /opt/protoc /usr/local

FROM tools AS generate
RUN --mount=target=github.com/tonistiigi/fsutil \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod <<EOT
  set -e
  mkdir /out
  find github.com/tonistiigi/fsutil -name '*.proto' | xargs protoc --go_out=/out --go-vtproto_out=/out
EOT

FROM scratch AS update
COPY --link --from=generate /out/github.com/tonistiigi/fsutil/ /

FROM tools AS validate
WORKDIR /app/github.com/tonistiigi/fsutil
RUN --mount=target=.,rw \
    --mount=target=/out,from=update <<EOT
  set -e
  git add -A
  cp -rf /out/* /app/
  diff=$(git status --porcelain -- **/*.pb.go 2>/dev/null)
  if [ -n "$diff" ]; then
    echo >&2 'ERROR: The result of "./hack/validate-generated-files" differs. Please update with "./hack/update-generated-files".'
    echo "$diff"
    exit 1
  fi
EOT
