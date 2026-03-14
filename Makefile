.PHONY: build install clean test

VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GOOGLE_OAUTH_CLIENT_ID ?= PLACEHOLDER.apps.googleusercontent.com
GOOGLE_OAUTH_CLIENT_SECRET ?= PLACEHOLDER

LDFLAGS := -s -w \
	-X github.com/xtam-ai/xtam-cli/cmd.Version=$(VERSION) \
	-X github.com/xtam-ai/xtam-cli/cmd.Commit=$(COMMIT) \
	-X github.com/xtam-ai/xtam-cli/cmd.Date=$(DATE) \
	-X github.com/xtam-ai/xtam-cli/internal/auth.ClientID=$(GOOGLE_OAUTH_CLIENT_ID) \
	-X github.com/xtam-ai/xtam-cli/internal/auth.ClientSecret=$(GOOGLE_OAUTH_CLIENT_SECRET)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/xtam ./main.go

install: build
	cp bin/xtam /usr/local/bin/xtam

clean:
	rm -rf bin/

test:
	go test ./...
