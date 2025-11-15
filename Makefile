SHELL := /bin/bash
OUT := $(shell pwd)/_out
BUILDARCH := $(shell uname -m)
GCC := $(OUT)/$(BUILDARCH)-linux-musl-cross/bin/$(BUILDARCH)-linux-musl-gcc
BIN_NAME := meteotrentino-exporter
BIN_PATH := $(OUT)/$(BIN_NAME)
IMAGE := ghcr.io/wouldgo/meteotrentino-exporter
VERSION := 0.0.1

# EXCLUDED_PACKAGES := \
# 	wouldgo.me/meteotrentino-exporter \
# 	wouldgo.me/meteotrentino-exporter/internal/api \
# 	wouldgo.me/meteotrentino-exporter/mocks

# PACKAGES := $(shell go list ./... | grep -Fvx -f <(printf '%s\n' $(EXCLUDED_PACKAGES)))

.PHONY: clean update install lint generate build run test musl
default: clean install build

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

build:
	CGO_ENABLED=1 \
	CC_FOR_TARGET=$(GCC) \
	CC=$(GCC) \
	go build \
		-ldflags "-s -w -linkmode external -extldflags -static" \
		-trimpath \
		-a -o $(BIN_PATH) cmd/*.go

run: lint install
	STATION="T0147" go run cmd/*.go

# test:
# 	go test -tags=test -race -parallel=10 -timeout 120s -cover -coverprofile=_out/.coverage -v $(PACKAGES);
# 	go tool cover -html=_out/.coverage -o=./_out/coverage.html

docker:
	docker build --tag "$(IMAGE):$(VERSION)" --file cmd/Dockerfile .

musl:
	if [ ! -d "$(OUT)/$(BUILDARCH)-linux-musl-cross" ]; then \
		(cd $(OUT); curl -LOk https://musl.cc/$(BUILDARCH)-linux-musl-cross.tgz) && \
		tar zxf $(OUT)/$(BUILDARCH)-linux-musl-cross.tgz -C $(OUT); \
	fi
