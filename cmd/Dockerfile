FROM golang:1.25.4-alpine3.22 AS prometheus_builder

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

FROM golang:1.25.4-alpine3.22 AS influxdb_builder

RUN apk add --no-cache \
  build-base \
  make \
  curl \
  bash

COPY . /workdir
WORKDIR /workdir

ENV CGO_CPPFLAGS="-D_FORTIFY_SOURCE=2 -fstack-protector-all"
ENV GOFLAGS="-buildmode=pie"

RUN make influxdb

FROM scratch AS prometheus
COPY --from=prometheus_builder /workdir/_out/meteotrentino-exporter /entrypoint

ENTRYPOINT ["/entrypoint"]

FROM scratch AS influxdb
COPY --from=influxdb_builder /workdir/_out/meteotrentino-exporter /entrypoint

ENTRYPOINT ["/entrypoint"]
