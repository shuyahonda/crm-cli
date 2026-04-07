# crm-cli Makefile

BINARY      := crm-cli
MODULE      := github.com/shuyahonda/crm-cli
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -s -w -X main.version=$(VERSION)

# Default: build for the host platform
.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

# Windows ARM64 (primary target for this project)
.PHONY: build-windows-arm64
build-windows-arm64:
	GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-windows-arm64.exe .

# Windows AMD64
.PHONY: build-windows-amd64
build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-windows-amd64.exe .

# Linux AMD64
.PHONY: build-linux-amd64
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-amd64 .

# macOS ARM64 (Apple Silicon)
.PHONY: build-macos-arm64
build-macos-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-macos-arm64 .

# Build all platforms
.PHONY: build-all
build-all: build-windows-arm64 build-windows-amd64 build-linux-amd64 build-macos-arm64

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: clean
clean:
	rm -f $(BINARY) $(BINARY)-*.exe $(BINARY)-linux-* $(BINARY)-macos-*

.PHONY: install
install:
	go install -ldflags "$(LDFLAGS)" .

# Install to Windows PATH-friendly location (requires WSL or cross-env)
.PHONY: install-windows
install-windows: build-windows-arm64
	@echo "Built: $(BINARY)-windows-arm64.exe"
	@echo "Copy to a directory in your Windows PATH, e.g. C:\\Windows\\System32\\"

.DEFAULT_GOAL := build
