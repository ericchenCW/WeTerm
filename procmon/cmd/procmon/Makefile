BINARY := procmon
PKG := .
LDFLAGS := -s -w -X main.version=$(shell git describe --tags --dirty --always 2>/dev/null || echo dev)

.PHONY: build build-linux test clean

# Local build (for the current platform — convenient for development only;
# the binary is only useful on Linux because it reads /proc).
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

# Production build: static Linux/amd64 binary. This is the shippable artifact.
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BINARY)-linux-amd64 $(PKG)
	@file $(BINARY)-linux-amd64 || true

test:
	go test ./...

clean:
	rm -f $(BINARY) $(BINARY)-linux-amd64
