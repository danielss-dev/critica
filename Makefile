GO := go
BINARY := critica
PKG := ./...

.PHONY: all build install test lint fmt vet clean ci

all: build install

build:
	$(GO) build -o $(BINARY)

install:
	$(GO) install ./...

test:
	$(GO) test $(PKG)

lint:
	@command -v golangci-lint >/dev/null 2>&1 || (echo "golangci-lint not found; run 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest'" && exit 1)
	golangci-lint run

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

clean:
	rm -f $(BINARY)

ci: fmt vet lint test


