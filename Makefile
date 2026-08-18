.PHONY: build run test lint vet demo seed clean

BINARY  := meshery-mcp-demo
CMD     := ./cmd/meshery-mcp-demo
VERSION ?= 0.1.0
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS := -X github.com/AkashNaickar/meshery-mcp-demo/internal/version.Version=$(VERSION) \
           -X github.com/AkashNaickar/meshery-mcp-demo/internal/version.CommitSHA=$(COMMIT)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(CMD)

run: build
	./bin/$(BINARY)

test:
	go test ./... -race

lint:
	golangci-lint run

vet:
	go vet ./...

demo: seed
	go run -ldflags "$(LDFLAGS)" $(CMD)

seed:
	./scripts/seed.sh

clean:
	rm -rf bin coverage.txt