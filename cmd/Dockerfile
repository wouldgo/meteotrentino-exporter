FROM --platform=$BUILDPLATFORM golang:1.25.4-alpine3.22 AS prometheus_builder

RUN apk add --no-cache \
  build-base \
  make \
  curl \
  bash

COPY . /workdir
WORKDIR /workdir

ARG TARGETARCH
ARG BUILDPLATFORM

ENV CGO_CPPFLAGS="-D_FORTIFY_SOURCE=2 -fstack-protector-all"
ENV GOFLAGS="-buildmode=pie"

RUN make build TAGS="prometheus" ARCH=$TARGETARCH

RUN find _out -maxdepth 4 -type d -exec ls -ld "{}" \;

FROM --platform=$BUILDPLATFORM golang:1.25.4-alpine3.22 AS influxdb_builder

RUN apk add --no-cache \
  build-base \
  make \
  curl \
  bash

COPY . /workdir
WORKDIR /workdir

ARG TARGETARCH
ARG BUILDPLATFORM

ENV CGO_CPPFLAGS="-D_FORTIFY_SOURCE=2 -fstack-protector-all"
ENV GOFLAGS="-buildmode=pie"

RUN make build TAGS="influxdb" ARCH=$TARGETARCH

RUN find _out -maxdepth 4 -type d -exec ls -ld "{}" \;

FROM --platform=$BUILDPLATFORM scratch AS prometheus

ARG TARGETARCH
ARG BUILDPLATFORM

COPY --from=prometheus_builder /workdir/_out/linux/${TARGETARCH}/meteotrentino-exporter-prometheus /entrypoint
ENTRYPOINT ["/entrypoint"]

FROM --platform=$BUILDPLATFORM scratch AS influxdb

ARG TARGETARCH
ARG BUILDPLATFORM

COPY --from=influxdb_builder /workdir/_out/linux/${TARGETARCH}/meteotrentino-exporter-influxdb /entrypoint
ENTRYPOINT ["/entrypoint"]
