# syntax = docker/dockerfile:1.1-experimental
FROM mvdan/shfmt:v3.1.2-alpine AS shfmt
WORKDIR /src
ARG SHFMT_FLAGS="-i 2 -ci"

FROM shfmt AS generate
RUN --mount=target=/src,rw \
  shfmt -l -w $SHFMT_FLAGS ./hack && \
  mkdir -p /out && cp -r ./hack /out/

FROM scratch AS update
COPY --from=generate /out /

FROM shfmt AS validate
RUN --mount=target=. \
  shfmt $SHFMT_FLAGS -d ./hack
