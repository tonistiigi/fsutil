# syntax=docker/dockerfile:1.2
ARG GO_VERSION=1.18

FROM golang:${GO_VERSION}-alpine
RUN apk add --no-cache git gcc musl-dev
RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.45.2
WORKDIR /go/src/github.com/tonistiigi/fsutil
RUN --mount=target=/go/src/github.com/tonistiigi/fsutil --mount=target=/root/.cache,type=cache --mount=target=/go/pkg/mod,type=cache \
  golangci-lint run
