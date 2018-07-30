# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: all test clean
.PHONY: gvite-linux-386 gvite-linux-amd64 gvite-linux-mips64 gvite-linux-mips64le
.PHONY: gvite-linux-arm gvite-linux-arm-5 gvite-linux-arm-6 gvite-linux-arm-7 gvite-linux-arm64
.PHONY: gvite-darwin gvite-darwin-386 gvite-darwin-amd64
.PHONY: gvite-windows gvite-windows-386 gvite-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

gvite:
	build/env.sh go run build/ci.go install ./cmd/gvite
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gvite\" to launch gvite."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install


test: all
	build/env.sh go run build/ci.go test

lint: ## Run linters.
	build/env.sh go run build/ci.go lint

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

gvite-cross: gvite-linux gvite-darwin gvite-windows gvite-android gvite-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gvite-*

gvite-linux: gvite-linux-386 gvite-linux-amd64 gvite-linux-arm gvite-linux-mips64 gvite-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gvite-linux-*

gvite-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gvite
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gvite-linux-* | grep 386

gvite-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gvite
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gvite-linux-* | grep amd64

gvite-linux-arm: gvite-linux-arm-5 gvite-linux-arm-6 gvite-linux-arm-7 gvite-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gvite-linux-* | grep arm

gvite-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gvite
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gvite-linux-* | grep arm-5

gvite-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gvite
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gvite-linux-* | grep arm-6

gvite-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gvite
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gvite-linux-* | grep arm-7

gvite-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gvite
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gvite-linux-* | grep arm64

gvite-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gvite
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gvite-linux-* | grep mips

gvite-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gvite
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gvite-linux-* | grep mipsle

gvite-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gvite
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gvite-linux-* | grep mips64

gvite-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gvite
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gvite-linux-* | grep mips64le

gvite-darwin: gvite-darwin-386 gvite-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gvite-darwin-*

gvite-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gvite
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gvite-darwin-* | grep 386

gvite-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gvite
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gvite-darwin-* | grep amd64

gvite-windows: gvite-windows-386 gvite-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gvite-windows-*

gvite-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gvite
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gvite-windows-* | grep 386

gvite-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gvite
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gvite-windows-* | grep amd64
