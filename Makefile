SHELL := /bin/bash
OUT := $(shell pwd)/_out

# ----------------------------------------
# build tags
# example:
# make build TAGS="prometheus"
# make build TAGS="influxdb"
# default to prometheus
# ----------------------------------------

TAGS ?= prometheus
GO_BUILD_TAGS := $(if $(strip $(TAGS)),-tags $(TAGS),)
BIN_NAME := meteotrentino-exporter$(if $(strip $(TAGS)),-$(TAGS),)

PACKAGE_REGISTRY := ghcr.io/wouldgo
VERSION := 0.1.2

ARCH ?= amd64
OS ?= linux

# ----------------------------------------
# musl toolchain mapping
# ----------------------------------------

MUSL_MAP_amd64 := x86_64-linux-musl
MUSL_MAP_arm64 := aarch64-linux-musl
MUSL_MAP_386 := i686-linux-musl
MUSL_MAP_arm := arm-linux-musleabihf
MUSL_MAP_riscv64 := riscv64-linux-musl

MUSL_TOOLCHAIN := $(MUSL_MAP_$(ARCH))

# ----------------------------------------

EXT := $(if $(filter windows,$(OS)),.exe,)
BIN_PATH := $(OUT)/$(OS)/$(ARCH)/$(BIN_NAME)$(EXT)

# ----------------------------------------
# supported targets
# ----------------------------------------

SUPPORTED_OS := linux windows darwin

SUPPORTED_ARCH_linux := amd64 arm64 386 arm riscv64
SUPPORTED_ARCH_windows := amd64 arm64 386
SUPPORTED_ARCH_darwin := amd64 arm64

ARCHS_linux := $(SUPPORTED_ARCH_linux)
ARCHS_windows := $(SUPPORTED_ARCH_windows)
ARCHS_darwin := $(SUPPORTED_ARCH_darwin)

# ----------------------------------------

.PHONY: default clean install update lint generate run docker \
        build build-all musl print-archs

default: clean install build

# ----------------------------------------

run_influxdb: lint install
	INFLUXDB_URL=http://127.0.0.1:9000 \
	INFLUXDB_DATABASE="nothing" \
	INFLUXDB_TOKEN="nothing" \
	STATION=T0147 \
	go run \
		-tags influxdb \
		./cmd

run_prometheus: lint install
	STATION="T0147" \
	go run \
		-tags prometheus \
		./cmd

run_profile: lint install
	STATION="T0147" \
	go run \
		-tags profile,prometheus \
		./cmd

profile:
	go tool pprof \
		--http=:8081 \
		http://127.0.0.0:8080/debug/pprof/allocs?seconds=120

docker:
	docker buildx build --target prometheus \
		--progress plain \
		--platform linux/arm64,linux/amd64 \
  	--tag "$(PACKAGE_REGISTRY)/meteotrentino-exporter-prometheus:$(VERSION)" \
		--file cmd/Dockerfile \
		--push \
  	.

	docker buildx build --target influxdb \
		--progress plain \
		--platform linux/arm64,linux/amd64 \
		--tag "$(PACKAGE_REGISTRY)/meteotrentino-exporter-influxdb:$(VERSION)" \
		--file cmd/Dockerfile \
		--push \
		.

clean:
	rm -rf $(OUT)
	mkdir -p $(OUT)
	touch $(OUT)/.keep

update:
	go mod tidy -v

install:
	go mod download

lint:
	golangci-lint run

generate:
	go generate -v ./...

# ----------------------------------------

print-archs:
	@echo "$(ARCHS_$(OS))"

# ----------------------------------------

build-all:
	@for os in $(SUPPORTED_OS); do \
		archs="$$(OS=$$os $(MAKE) -s print-archs)"; \
		echo ""; \
		echo "==== $$os: $$archs ===="; \
		for arch in $$archs; do \
			echo "---- Building $$os/$$arch ----"; \
			$(MAKE) --no-print-directory OS=$$os ARCH=$$arch TAGS="$(TAGS)" build; \
		done; \
	done

# ----------------------------------------

musl:
	@if [ "$(OS)" = "linux" ] && [ -n "$(MUSL_TOOLCHAIN)" ]; then \
		if [ ! -d "$(OUT)/$(MUSL_TOOLCHAIN)-cross" ]; then \
			echo "Downloading musl toolchain $(MUSL_TOOLCHAIN)"; \
			cd $(OUT) && curl -LO https://musl.cc/$(MUSL_TOOLCHAIN)-cross.tgz; \
			tar -xzf $(OUT)/$(MUSL_TOOLCHAIN)-cross.tgz -C $(OUT); \
		fi \
	fi

# ----------------------------------------

build: $(BIN_PATH)

$(BIN_PATH): musl
	mkdir -p $(OUT)/$(OS)/$(ARCH)

	@if [ "$(OS)" = "linux" ]; then \
		if [ -n "$(MUSL_TOOLCHAIN)" ]; then \
			echo "Building linux/$(ARCH) with musl. Tags: $(TAGS) ( $(GO_BUILD_TAGS) )"; \
			CC=$(OUT)/$(MUSL_TOOLCHAIN)-cross/bin/$(MUSL_TOOLCHAIN)-gcc \
			CC_FOR_TARGET=$(OUT)/$(MUSL_TOOLCHAIN)-cross/bin/$(MUSL_TOOLCHAIN)-gcc \
			CGO_ENABLED=1 \
			GOOS=$(OS) GOARCH=$(ARCH) \
			go build \
				$(GO_BUILD_TAGS) \
				-trimpath \
				-ldflags "\
					-buildid= \
					-s -w \
					-linkmode external \
					-extldflags '-static' \
					-X main.version=$(VERSION)" \
				-o $(BIN_PATH) \
				./cmd ; \
		else \
			echo "Building linux/$(ARCH) pure Go. Tags: $(TAGS) ( $(GO_BUILD_TAGS) )"; \
			CGO_ENABLED=0 \
			GOOS=$(OS) GOARCH=$(ARCH) \
			go build \
				$(GO_BUILD_TAGS) \
				-trimpath \
				-ldflags "\
					-s -w \
					-X main.version=$(VERSION)" \
				-o $(BIN_PATH) \
				./cmd ; \
		fi \
	elif [ "$(OS)" = "windows" ]; then \
		echo "Building windows/$(ARCH). Tags: $(TAGS) ( $(GO_BUILD_TAGS) )"; \
		CGO_ENABLED=0 \
		GOOS=$(OS) GOARCH=$(ARCH) \
		go build \
			$(GO_BUILD_TAGS) \
			-trimpath \
			-ldflags "\
				-s -w \
				-X main.version=$(VERSION)" \
			-o $(BIN_PATH) \
			./cmd ; \
	elif [ "$(OS)" = "darwin" ]; then \
		echo "Building darwin/$(ARCH). Tags: $(TAGS) ( $(GO_BUILD_TAGS) )"; \
		CGO_ENABLED=0 \
		GOOS=$(OS) GOARCH=$(ARCH) \
		go build \
			$(GO_BUILD_TAGS) \
			-trimpath \
			-ldflags "\
				-s -w \
				-X main.version=$(VERSION)" \
			-o $(BIN_PATH) \
			./cmd ; \
	else \
		echo "Unsupported OS $(OS)"; \
		exit 1; \
	fi
