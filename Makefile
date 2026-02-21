SHELL := /bin/bash
OUT := $(shell pwd)/_out
BUILDARCH := $(shell uname -m)
GCC := $(OUT)/$(BUILDARCH)-linux-musl-cross/bin/$(BUILDARCH)-linux-musl-gcc
BIN_NAME := meteotrentino-exporter
BIN_PATH := $(OUT)/$(BIN_NAME)
IMAGE := ghcr.io/wouldgo/meteotrentino-exporter
VERSION := 0.1.0

.PHONY: clean update install lint generate build_influxdb build run_influxdb run run_profile profile docker musl
default: clean install build
influxdb: clean install

clean:
	rm -Rf $(OUT)/*
	mkdir -p $(OUT)
	touch $(OUT)/.keep

update:
	go mod tidy -v

install: musl
	go mod download

lint:
	golangci-lint run

generate:
	go generate -v ./...

build_influxdb:
	CGO_ENABLED=1 \
	CC_FOR_TARGET=$(GCC) \
	CC=$(GCC) \
	go build \
		-ldflags "-s -w -linkmode external -extldflags -static" \
		-trimpath \
		-a -o $(BIN_PATH) ./cmd/influxdb

build:
	CGO_ENABLED=1 \
	CC_FOR_TARGET=$(GCC) \
	CC=$(GCC) \
	go build \
		-ldflags "-s -w -linkmode external -extldflags -static" \
		-trimpath \
		-a -o $(BIN_PATH) ./cmd/prometheus

run_influxdb:
	ENABLE_INFLUXDB=true \
	INFLUXDB_URL=http://127.0.0.1:9000 \
	INFLUXDB_DATABASE="nothing" \
	INFLUXDB_TOKEN="nothing" \
	STATION=T0147 \
	go run ./cmd/influxdb

run: lint install
	STATION="T0147" go run ./cmd/prometheus

run_profile: lint install
	STATION="T0147" go run -tags profile ./cmd/prometheus

profile:
	go tool pprof --http=:8081 http://127.0.0.0:8080/debug/pprof/allocs?seconds=120

docker:
	docker build --tag "$(IMAGE):$(VERSION)" --file cmd/Dockerfile .

musl:
	if [ ! -d "$(OUT)/$(BUILDARCH)-linux-musl-cross" ]; then \
		(cd $(OUT); curl -LOk https://musl.cc/$(BUILDARCH)-linux-musl-cross.tgz) && \
		tar zxf $(OUT)/$(BUILDARCH)-linux-musl-cross.tgz -C $(OUT); \
	fi
