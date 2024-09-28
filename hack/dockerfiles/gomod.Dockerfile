# syntax=docker/dockerfile:1

ARG GO_VERSION=1.23

FROM golang:${GO_VERSION}-alpine AS gomod
RUN  apk add --no-cache git
WORKDIR /src
RUN --mount=target=/src,rw \
  --mount=target=/go/pkg/mod,type=cache \
  go mod tidy && \
  mkdir /out && cp -r go.mod go.sum /out && \
  cd bench && go mod tidy && \
  mkdir /out/bench && cp -r go.mod go.sum /out/bench

FROM scratch AS update
COPY --from=gomod /out /

FROM gomod AS validate
RUN --mount=target=.,rw <<EOT
  set -e
  git add -A
  cp -rf /out/* .
  diff=$(git status --porcelain -- go.mod go.sum bench/go.mod bench/go.sum 2>/dev/null)
  if [ -n "$diff" ]; then
    echo >&2 'ERROR: The result of "./hack/validate-gomod" differs. Please update with "./hack/update-gomod".'
    echo "$diff"
    exit 1
  fi
EOT
