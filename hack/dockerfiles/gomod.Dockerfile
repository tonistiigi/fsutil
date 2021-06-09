# syntax = docker/dockerfile:1.2
ARG GO_VERSION=1.16

FROM golang:${GO_VERSION}-alpine AS gomod
RUN  apk add --no-cache git
WORKDIR /src
RUN --mount=target=/src,rw \
  --mount=target=/go/pkg/mod,type=cache \
  go mod tidy && go mod vendor && \
  mkdir /out && cp -r go.mod go.sum /out

FROM scratch AS update
COPY --from=gomod /out /

FROM gomod AS validate
RUN --mount=target=.,rw \
  git add -A && \
  cp -rf /out/* . && \
  ./hack/validate-gomod check
