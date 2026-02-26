BINARY := lurk
VERSION := $(shell v=$$(git describe --tags --always --dirty 2>/dev/null | sed 's/^v//'); [ -n "$$v" ] && echo "$$v" || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
GOFILES := $(shell find . -name '*.go' -type f)

.PHONY: build clean install test all

build: $(BINARY)

$(BINARY): $(GOFILES)
	go build $(LDFLAGS) -o $(BINARY) .

all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 build-windows-arm64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 .

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 .

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 .

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 .

build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe .

build-windows-arm64:
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-arm64.exe .

install: build
	cp $(BINARY) $(HOME)/.local/bin/$(BINARY)

install-skill: build
	mkdir -p $(HOME)/.claude/skills/lurk
	cp $(BINARY) $(HOME)/.claude/skills/lurk/$(BINARY)
	cp skill/SKILL.md $(HOME)/.claude/skills/lurk/SKILL.md
	@echo "Installed lurk skill to ~/.claude/skills/lurk/"

clean:
	rm -f $(BINARY)
	rm -rf dist/

test:
	go test ./...
