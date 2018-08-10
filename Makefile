
.PHONY: all clean
.PHONY: gvite_linux gvite-linux-386 gvite-linux-amd64 gvite-darwin-amd64
.PHONY: gvite-darwin gvite-darwin-amd64
.PHONY: gvite-windows gvite-windows-386 gvite-windows-amd64
.PHONY: gitversion



GO ?= latest

MAINDIR = rpc
SERVERMAIN = $(shell pwd)/cmd/$(MAINDIR)/main.go
BUILDDIR = $(shell pwd)/build
GOBIN = $(BUILDDIR)/cmd/$(MAINDIR)

TESTCLIENTMAIN = $(shell pwd)/testdata/main/ipc_client.go

GITREV = $(shell git rev-parse HEAD)

gitversion:
	@echo "$(shell git rev-parse HEAD)" > $(shell pwd)/gitversion

gvite: gitversion
	go build -i -o $(GOBIN)/gvite $(SERVERMAIN)
	@echo "Build server done."
	@echo "Run \"$(GOBIN)/gvite\" to start gvite."

	go build -i -o $(GOBIN)/gvite-test-client $(TESTCLIENTMAIN)
	@echo "Build test client done."
	@echo "Run \"$(GOBIN)/gvite-test-client\" to start test client."

all: gvite-darwin gvite-windows gvite-linux

clean:
	rm -r $(BUILDDIR)/

gvite-linux: gvite-linux-386 gvite-linux-amd64
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/linux/gvite-linux-*

gvite-linux-386: gitversion
	env GOOS=linux GOARCH=386 go build -i -o $(GOBIN)/linux/gvite-linux-386 $(SERVERMAIN)
	@echo "Build server done."
	@ls -ld $(GOBIN)/linux/gvite-linux-386

	env GOOS=linux GOARCH=386 go build -i -o $(GOBIN)/linux/gvite-test-client-linux-386 $(TESTCLIENTMAIN)
	@echo "Build test client done."
	@ls -ld $(GOBIN)/linux/gvite-test-client-linux-386


gvite-linux-amd64: gitversion
	env GOOS=linux GOARCH=amd64 go build -i -o $(GOBIN)/linux/gvite-linux-amd64 $(SERVERMAIN)
	@echo "Build server done."
	@ls -ld $(GOBIN)/linux/gvite-linux-amd64

	env GOOS=linux GOARCH=amd64 go build -i -o $(GOBIN)/linux/gvite-test-client-linux-amd64 $(TESTCLIENTMAIN)
	@echo "Build test client done."
	@ls -ld $(GOBIN)/linux/gvite-test-client-linux-amd64


gvite-darwin: gitversion
	env GOOS=darwin GOARCH=amd64 go build -i -o $(GOBIN)/darwin/gvite-darwin $(SERVERMAIN)
	@echo "Build server done."
	@ls -ld $(GOBIN)/darwin/gvite-darwin

	env GOOS=darwin GOARCH=amd64 go build -i -o $(GOBIN)/darwin/gvite-test-client-darwin $(TESTCLIENTMAIN)
	@echo "Build test client done."
	@ls -ld $(GOBIN)/darwin/gvite-test-client-darwin


gvite-windows: gvite-windows-386 gvite-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/windows/gvite-windows-*

gvite-windows-386: gitversion
	env GOOS=windows GOARCH=386 go build -i -o $(GOBIN)/windows/gvite-windows-386.exe $(SERVERMAIN)
	@echo "Build server done."
	@ls -ld $(GOBIN)/windows/gvite-windows-386.exe

	env GOOS=windows GOARCH=386 go build -i -o $(GOBIN)/windows/gvite-test-client-windows-386.exe $(TESTCLIENTMAIN)
	@echo "Build test client done."
	@ls -ld $(GOBIN)/windows/gvite-test-client-windows-386.exe


gvite-windows-amd64: gitversion
	env GOOS=windows GOARCH=amd64 go build -i -o $(GOBIN)/windows/gvite-windows-amd64.exe $(SERVERMAIN)
	@echo "Build server done."
	@ls -ld $(GOBIN)/windows/gvite-windows-amd64.exe

	env GOOS=windows GOARCH=amd64 go build -i -o $(GOBIN)/windows/gvite-test-client-windows-amd64.exe $(TESTCLIENTMAIN)
	@echo "Build test client done."
	@ls -ld $(GOBIN)/windows/gvite-test-client-windows-amd64.exe


