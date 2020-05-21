.PHONY: all clean
.PHONY: gvite-linux-amd64
.PHONY: gvite-darwin
.PHONY: gvite-windows
.PHONY: build_version build

GO ?= latest

MAIN_DIR=gvite
WORK_DIR=$(shell pwd)
MAIN=$(shell pwd)/cmd/$(MAIN_DIR)/main.go
BUILD_ROOT_DIR=$(shell pwd)/build
VITE_GIT_COMMIT=$(shell git rev-parse HEAD)
VITE_VERSION=$(shell cat vite/version/buildversion)
VITE_VERSION_FILE=$(shell pwd)/vite/version/buildversion.go

BUILD_DIR=$(BUILD_ROOT_DIR)/cmd/$(MAIN_DIR)
BUILD_BIN=$(BUILD_DIR)/gvite

build_version:
	@echo "package version" > $(VITE_VERSION_FILE)
	@echo "const VITE_COMMIT_VERSION = "\"$(VITE_GIT_COMMIT)\" >> $(VITE_VERSION_FILE)
	@echo "const VITE_BUILD_VERSION = "\"$(VITE_VERSION)\" >> $(VITE_VERSION_FILE)
	@echo "gvite build version is "$(VITE_VERSION)", git commit is "$(VITE_GIT_COMMIT)"."

gvite:
	go build -i -o $(BUILD_BIN) $(MAIN)
	@echo "Build gvite done."
	@echo "Run $(BUILD_DIR)/gvite to start gvite."


build_linux_amd64:
	env GOOS=linux GOARCH=amd64 go build -i -o $(BUILD_DIR)/gvite-$(VITE_VERSION)-linux/gvite $(MAIN)

	@cp $(shell pwd)/conf/node_config.json $(BUILD_DIR)/gvite-$(VITE_VERSION)-linux/node_config.json
	@cp $(shell pwd)/bin/bootstrap_linux $(BUILD_DIR)/gvite-$(VITE_VERSION)-linux/bootstrap

	@ls -d $(BUILD_DIR)/gvite-$(VITE_VERSION)-linux/gvite
	@echo "Build linux version done."

build_darwin:
	env GOOS=darwin GOARCH=amd64 go build -i -o $(BUILD_DIR)/gvite-$(VITE_VERSION)-darwin/gvite $(MAIN)

	@cp  $(shell pwd)/conf/node_config.json $(BUILD_DIR)/gvite-$(VITE_VERSION)-darwin/node_config.json
	@ls -d $(BUILD_DIR)/gvite-$(VITE_VERSION)-darwin/gvite
	@echo "Build darwin version done"


build_windows:
	env GOOS=windows GOARCH=amd64 go build -i -o $(BUILD_DIR)/gvite-$(VITE_VERSION)-windows/gvite-windows-amd64.exe $(MAIN)

	@cp  $(shell pwd)/conf/node_config.json $(BUILD_DIR)/gvite-$(VITE_VERSION)-windows/node_config.json
	@ls -d $(BUILD_DIR)/gvite-$(VITE_VERSION)-windows/gvite-windows-amd64.exe
	@echo "Build windows version done."


gvite-darwin: build_version build_darwin

gvite-linux: build_version build_linux_amd64

gvite-windows: build_version build_windows

all: gvite-windows gvite-darwin gvite-linux

clean:
	rm -r $(BUILD_ROOT_DIR)/