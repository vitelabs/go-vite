
.PHONY: all clean
.PHONY: gvite_linux  gvite-linux-amd64 gvite-darwin-amd64
.PHONY: gvite-darwin gvite-darwin-amd64
.PHONY: gvite-windows gvite-windows-386 gvite-windows-amd64
.PHONY: gitversion


GO ?= latest

MAINDIR = gvite
SERVERMAIN = $(shell pwd)/cmd/$(MAINDIR)/main.go
BUILDDIR = $(shell pwd)/build
GOBIN = $(BUILDDIR)/cmd/$(MAINDIR)
VITE_VERSION = $(shell cat buildversion)

gitversion:
	@echo "package govite" > $(shell pwd)/buildversion.go
	@echo "const VITE_VERSION = \""$(shell git rev-parse HEAD)"\"" >> $(shell pwd)/buildversion.go
	@echo "const VITE_BUILD_VERSION = \""$(VITE_VERSION)"\"" >> $(shell pwd)/buildversion.go

gvite:
	@echo "package govite" > $(shell pwd)/buildversion.go
	@echo "const VITE_VERSION = \""$(shell git rev-parse HEAD)"\"" >> $(shell pwd)/buildversion.go
	@echo "const VITE_BUILD_VERSION = \""$(VITE_VERSION)"\"" >> $(shell pwd)/buildversion.go
	go build -i -o $(GOBIN)/gvite $(SERVERMAIN)
	@echo "Build server done."
	@echo "Run \"$(GOBIN)/gvite\" to start gvite."

all: gvite-windows gvite-darwin  gvite-linux


clean:
	rm -r $(BUILDDIR)/

gvite-linux: gvite-linux-amd64
	@echo "Linux cross compilation done:"


gvite-linux-amd64:
	@echo "version package version is "$(VITE_VERSION)"."
	@echo "package govite" > $(shell pwd)/buildversion.go
	@echo "const VITE_VERSION = \""$(shell git rev-parse HEAD)"\"" >> $(shell pwd)/buildversion.go
	@echo "const VITE_BUILD_VERSION = \""$(VITE_VERSION)"\"" >> $(shell pwd)/buildversion.go
	env GOOS=linux GOARCH=amd64 go build -i -o $(GOBIN)/gvite-$(VITE_VERSION)-linux/gvite $(SERVERMAIN)
	@cp  $(shell pwd)/node_config.json $(GOBIN)/gvite-$(VITE_VERSION)-linux/node_config.json
	@cp  $(shell pwd)/bootstrap $(GOBIN)/gvite-$(VITE_VERSION)-linux/bootstrap
	@echo "Build server done."
	@ls -ld $(GOBIN)/gvite-$(VITE_VERSION)-linux/gvite

gvite-darwin:
	@echo "version package version is "$(VITE_VERSION)"."
	@echo "package govite" > $(shell pwd)/buildversion.go
	@echo "const VITE_VERSION = \""$(shell git rev-parse HEAD)"\"" >> $(shell pwd)/buildversion.go
	@echo "const VITE_BUILD_VERSION = \""$(VITE_VERSION)"\"" >> $(shell pwd)/buildversion.go
	env GOOS=darwin GOARCH=amd64 go build -i -o $(GOBIN)/gvite-$(VITE_VERSION)-darwin/gvite $(SERVERMAIN)
	@cp  $(shell pwd)/node_config.json $(GOBIN)/gvite-$(VITE_VERSION)-darwin/node_config.json
	@cp  $(shell pwd)/bootstrap $(GOBIN)/gvite-$(VITE_VERSION)-darwin/bootstrap
	@echo "Build server done."
	@ls -ld $(GOBIN)/gvite-$(VITE_VERSION)-darwin/gvite

gvite-windows: gvite-windows-386 gvite-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gvite-$(VITE_VERSION)-windows/gvite-windows-*

gvite-windows-386:
	@echo "package govite" > $(shell pwd)/buildversion.go
	@echo "const VITE_VERSION = \""$(shell git rev-parse HEAD)"\"" >> $(shell pwd)/buildversion.go
	@echo "const VITE_BUILD_VERSION = \""$(VITE_VERSION)"\"" >> $(shell pwd)/buildversion.go
	env GOOS=windows GOARCH=386 go build -i -o $(GOBIN)/gvite-$(VITE_VERSION)-windows/gvite-windows-386.exe $(SERVERMAIN)
	@cp  $(shell pwd)/node_config.json $(GOBIN)/gvite-$(VITE_VERSION)-windows/node_config.json
	@echo "Build server done."
	@ls -ld $(GOBIN)/gvite-$(VITE_VERSION)-windows/gvite-windows-386.exe

gvite-windows-amd64:
	@echo "package govite" > $(shell pwd)/buildversion.go
	@echo "const VITE_VERSION = \""$(shell git rev-parse HEAD)"\"" >> $(shell pwd)/buildversion.go
	@echo "const VITE_BUILD_VERSION = \""$(VITE_VERSION)"\"" >> $(shell pwd)/buildversion.go
	env GOOS=windows GOARCH=amd64 go build -i -o $(GOBIN)/gvite-$(VITE_VERSION)-windows/gvite-windows-amd64.exe $(SERVERMAIN)
	@cp  $(shell pwd)/node_config.json $(GOBIN)/gvite-$(VITE_VERSION)-windows/node_config.json
	@echo "Build server done."ls
	@ls -ld $(GOBIN)/gvite-$(VITE_VERSION)-windows/gvite-windows-amd64.exe



