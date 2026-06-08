GOMOD    := github.com/greyerof/ocpchart
CMD      := ./ocpchart
APP_MAIN := ./cmd/ocpchart

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS  := -X '$(GOMOD)/internal/commands.version=$(VERSION)' \
            -X '$(GOMOD)/internal/commands.commit=$(COMMIT)'

.PHONY: build lint test clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(CMD) $(APP_MAIN)

lint:
	golangci-lint run ./...

test:
	go test ./... -v

clean:
	rm -f $(CMD)
