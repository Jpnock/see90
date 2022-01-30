.PHONY: all build

all: build

build:
	@nex -o internal/firsttest/rp.nn.go internal/firsttest/rp.nex
	@goyacc -o internal/firsttest/y.go -v internal/firsttest/y.output internal/firsttest/rp.y
	@go build -o bin/example ./cmd/example/*.go
