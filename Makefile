VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build install test tidy

build:
	go build $(LDFLAGS) -o couloir ./cmd/couloir

install:
	go install $(LDFLAGS) ./cmd/couloir

test:
	go test ./...

tidy:
	go mod tidy
