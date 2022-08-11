.PHONY: all clean
.PHONY: gvite-linux-amd64
.PHONY: gvite-linux-arm64
.PHONY: gvite-darwin
.PHONY: gvite-windows
.PHONY: build_version build
.PHONY: test


GO ?= latest

MAIN_DIR=gvite
WORK_DIR=$(shell pwd)
MAIN=$(shell pwd)/cmd/$(MAIN_DIR)/main.go
BUILD_ROOT_DIR=$(shell pwd)/build
VITE_GIT_COMMIT=$(shell git rev-parse HEAD)
VITE_VERSION=$(shell cat version/buildversion)
VITE_VERSION_FILE=$(shell pwd)/version/buildversion.go

BUILD_DIR=$(BUILD_ROOT_DIR)/cmd/$(MAIN_DIR)
BUILD_BIN=$(BUILD_DIR)/gvite

build_version:
	@echo "package version" > $(VITE_VERSION_FILE)
	@echo "const VITE_COMMIT_VERSION = "\"$(VITE_GIT_COMMIT)\" >> $(VITE_VERSION_FILE)
	@echo "const VITE_BUILD_VERSION = "\"$(VITE_VERSION)\" >> $(VITE_VERSION_FILE)
	@echo "gvite build version is "$(VITE_VERSION)", git commit is "$(VITE_GIT_COMMIT)"."

build:
	GO111MODULE=on go build -o $(BUILD_BIN) $(MAIN)
	@echo "Build gvite done."
	@echo "Run $(BUILD_DIR)/gvite to start gvite."

test:
	GO111MODULE=on go test ./common/upgrade
	GO111MODULE=on go test ./ledger/pipeline
	GO111MODULE=on go test ./tools/toposort
	GO111MODULE=on go test ./vm
	GO111MODULE=on go test ./wallet
	GO111MODULE=on go test ./ledger/onroad/pool

build_linux_amd64:
	env GOOS=linux CGO_ENABLED=0 GO111MODULE=on GOARCH=amd64 go build -o $(BUILD_DIR)/gvite-$(VITE_VERSION)-linux/gvite $(MAIN)

	@cp $(shell pwd)/conf/node_config.json $(BUILD_DIR)/gvite-$(VITE_VERSION)-linux/node_config.json
	@cp $(shell pwd)/bin/bootstrap_linux $(BUILD_DIR)/gvite-$(VITE_VERSION)-linux/bootstrap

	@ls -d $(BUILD_DIR)/gvite-$(VITE_VERSION)-linux/gvite
	@echo "Build linux version done."

build_linux_arm64:
	env GOOS=linux CGO_ENABLED=0 GO111MODULE=on GOARCH=arm64 go build -o $(BUILD_DIR)/gvite-$(VITE_VERSION)-linux-arm64/gvite $(MAIN)

	@cp $(shell pwd)/conf/node_config.json $(BUILD_DIR)/gvite-$(VITE_VERSION)-linux-arm64/node_config.json
	@cp $(shell pwd)/bin/bootstrap_linux $(BUILD_DIR)/gvite-$(VITE_VERSION)-linux-arm64/bootstrap

	@ls -d $(BUILD_DIR)/gvite-$(VITE_VERSION)-linux-arm64/gvite
	@echo "Build linux-arm64 version done."	

build_darwin:
	env GOOS=darwin CGO_ENABLED=0 GO111MODULE=on GOARCH=amd64 go build -o $(BUILD_DIR)/gvite-$(VITE_VERSION)-darwin/gvite $(MAIN)

	@cp  $(shell pwd)/conf/node_config.json $(BUILD_DIR)/gvite-$(VITE_VERSION)-darwin/node_config.json
	@ls -d $(BUILD_DIR)/gvite-$(VITE_VERSION)-darwin/gvite
	@echo "Build darwin version done"


build_windows:
	env GOOS=windows CGO_ENABLED=0 GO111MODULE=on GOARCH=amd64 go build -o $(BUILD_DIR)/gvite-$(VITE_VERSION)-windows/gvite-windows-amd64.exe $(MAIN)

	@cp  $(shell pwd)/conf/node_config.json $(BUILD_DIR)/gvite-$(VITE_VERSION)-windows/node_config.json
	@ls -d $(BUILD_DIR)/gvite-$(VITE_VERSION)-windows/gvite-windows-amd64.exe
	@echo "Build windows version done."


gvite-darwin: build_version build_darwin

gvite-linux: build_version build_linux_amd64

gvite-linux-arm64: build_version build_linux_arm64

gvite-windows: build_version build_windows

gvite: build_version build

all: gvite-windows gvite-darwin gvite-linux gvite-linux-arm64

clean:
	rm -r $(BUILD_ROOT_DIR)/