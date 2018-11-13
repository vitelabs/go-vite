
.PHONY: all clean
.PHONY: gvite_linux gvite-linux-386 gvite-linux-amd64 gvite-darwin-amd64
.PHONY: gvite-darwin gvite-darwin-amd64
.PHONY: gvite-windows gvite-windows-386 gvite-windows-amd64
.PHONY: gitversion



GO ?= latest

MAINDIR = gvite
SERVERMAIN = $(shell pwd)/cmd/$(MAINDIR)/main.go
BUILDDIR = $(shell pwd)/build
GOBIN = $(BUILDDIR)/cmd/$(MAINDIR)

gitversion:
	@echo "package govite\nconst VITE_VERSION = \""$(shell git rev-parse HEAD)"\"" > $(shell pwd)/buildversion.go

gvite:
	@echo "package govite\nconst VITE_VERSION = \""$(shell git rev-parse HEAD)"\"" > $(shell pwd)/buildversion.go
	go build -i -o $(GOBIN)/gvite $(SERVERMAIN)
	@echo "Build server done."
	@echo "Run \"$(GOBIN)/gvite\" to start gvite."

all: gvite-windows gvite-darwin  gvite-linux

clean:
	rm -r $(BUILDDIR)/

gvite-linux: gvite-linux-386 gvite-linux-amd64
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/linux/gvite-linux-*

gvite-linux-386:
	@echo "package govite\nconst VITE_VERSION = \""$(shell git rev-parse HEAD)"\"" > $(shell pwd)/buildversion.go
	env GOOS=linux GOARCH=386 go build -i -o $(GOBIN)/linux/gvite-linux-386 $(SERVERMAIN)
	@echo "Build server done."
	@ls -ld $(GOBIN)/linux/gvite-linux-386


gvite-linux-amd64:
	@echo "package govite\nconst VITE_VERSION = \""$(shell git rev-parse HEAD)"\"" > $(shell pwd)/buildversion.go
	env GOOS=linux GOARCH=amd64 go build -i -o $(GOBIN)/linux/gvite-linux-amd64 $(SERVERMAIN)
	@echo "Build server done."
	@ls -ld $(GOBIN)/linux/gvite-linux-amd64

gvite-darwin:
	@echo "package govite\nconst VITE_VERSION = \""$(shell git rev-parse HEAD)"\"" > $(shell pwd)/buildversion.go
	env GOOS=darwin GOARCH=amd64 go build -i -o $(GOBIN)/darwin/gvite-darwin $(SERVERMAIN)
	@echo "Build server done."
	@ls -ld $(GOBIN)/darwin/gvite-darwin

gvite-windows: gvite-windows-386 gvite-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/windows/gvite-windows-*

gvite-windows-386:
	@echo "package govite\nconst VITE_VERSION = \""$(shell git rev-parse HEAD)"\"" > $(shell pwd)/buildversion.go
	env GOOS=windows GOARCH=386 go build -i -o $(GOBIN)/windows/gvite-windows-386.exe $(SERVERMAIN)
	@echo "Build server done."
	@ls -ld $(GOBIN)/windows/gvite-windows-386.exe

gvite-windows-amd64:
	@echo "package govite\nconst VITE_VERSION = \""$(shell git rev-parse HEAD)"\"" > $(shell pwd)/buildversion.go
	env GOOS=windows GOARCH=amd64 go build -i -o $(GOBIN)/windows/gvite-windows-amd64.exe $(SERVERMAIN)
	@echo "Build server done."
	@ls -ld $(GOBIN)/windows/gvite-windows-amd64.exe



