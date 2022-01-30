.PHONY: all build-example build-see90

all: build-example build-see90

build-example:
	@nex -o internal/firsttest/rp.nn.go internal/firsttest/rp.nex
	@goyacc -o internal/firsttest/y.go -v internal/firsttest/y.output internal/firsttest/rp.y
	@go build -o bin/example ./cmd/example/*.go

build-see90:
	@nex -o pkg/c90/c90.nn.go pkg/c90/c90.nex
	@goyacc -o pkg/c90/y.go -v pkg/c90/y.output pkg/c90/grammar.y
	@go build -o bin/see90 ./cmd/see90/*.go