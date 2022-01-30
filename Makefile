.PHONY: all build

all: build

build:
	@go build -o bin/example ./cmd/example/*.go
