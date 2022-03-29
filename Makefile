.PHONY: all build-example bin/c_compiler install_go install_go_modules test

export PATH := $(HOME)/go/bin:$(HOME)/.go/bin:$(PATH)

all: bin/c_compiler

build-example:
	@nex -o internal/firsttest/rp.nn.go internal/firsttest/rp.nex
	@goyacc -o internal/firsttest/y.go -v internal/firsttest/y.output internal/firsttest/rp.y
	@go build -o bin/example ./cmd/example/*.go

install_go_modules:
	@echo "Installing goyacc and nex via Golang install"
	GOPATH="$(HOME)/go" GOROOT="$(HOME)/.go" go install golang.org/x/tools/cmd/goyacc@v0.1.9
	GOPATH="$(HOME)/go" GOROOT="$(HOME)/.go" go install github.com/blynn/nex@master

install_go:
	@echo "Installing Golang 1.17"
	@./goinstall.sh

bin/c_compiler : install_go install_go_modules
	@GOPATH="$(HOME)/go" GOROOT="$(HOME)/.go" nex -o pkg/c90/c90.nn.go pkg/c90/c90.nex
	@GOPATH="$(HOME)/go" GOROOT="$(HOME)/.go" goyacc -o pkg/c90/y.go -v pkg/c90/y.output pkg/c90/grammar.y
	@GOPATH="$(HOME)/go" GOROOT="$(HOME)/.go" go build -o bin/see90 ./cmd/see90/*.go
	@cp bin/see90 bin/c_compiler
