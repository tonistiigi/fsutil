# syntax=docker/dockerfile:1

ARG GO_VERSION=1.23
ARG XX_VERSION=1.5.0
ARG GOLANGCI_LINT_VERSION=1.61.0

FROM --platform=$BUILDPLATFORM tonistiigi/xx:${XX_VERSION} AS xx
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine
RUN apk add --no-cache git gcc musl-dev
ENV GOFLAGS="-buildvcs=false"
ARG GOLANGCI_LINT_VERSION
RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v${GOLANGCI_LINT_VERSION}
COPY --link --from=xx / /
WORKDIR /go/src/github.com/tonistiigi/fsutil
ARG TARGETPLATFORM
RUN --mount=target=. \
    --mount=target=/root/.cache,type=cache,id=lint-cache-$TARGETPLATFORM \
    --mount=target=/go/pkg/mod,type=cache \
    xx-go --wrap && \
    golangci-lint run
