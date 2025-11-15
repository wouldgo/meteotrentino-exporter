FROM golang:1.25.4-alpine3.22 AS builder

RUN apk add --no-cache \
  build-base \
  make \
  curl \
  bash

COPY . /workdir
WORKDIR /workdir

ENV CGO_CPPFLAGS="-D_FORTIFY_SOURCE=2 -fstack-protector-all"
ENV GOFLAGS="-buildmode=pie"

RUN make

FROM scratch
COPY --from=builder /workdir/_out/meteotrentino-exporter /entrypoint

ENTRYPOINT ["/entrypoint"]
