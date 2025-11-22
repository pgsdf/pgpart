.PHONY: build run clean install deps test

# Build the application
build:
	@echo "Building PGPart..."
	go build -o pgpart .

# Run the application (requires root)
run:
	@echo "Running PGPart (requires root privileges)..."
	@if [ `id -u` -ne 0 ]; then \
		echo "Please run as root: sudo make run"; \
		exit 1; \
	fi
	./pgpart

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f pgpart
	go clean

# Install to /usr/local/bin
install: build
	@echo "Installing to /usr/local/bin..."
	install -m 755 pgpart /usr/local/bin/

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Build for release
release:
	@echo "Building release binary..."
	CGO_ENABLED=1 go build -ldflags="-s -w" -o pgpart .

# Show help
help:
	@echo "PGPart Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build    - Build the application"
	@echo "  run      - Run the application (requires root)"
	@echo "  clean    - Clean build artifacts"
	@echo "  install  - Install to /usr/local/bin"
	@echo "  deps     - Download and tidy dependencies"
	@echo "  test     - Run tests"
	@echo "  release  - Build optimized release binary"
	@echo "  help     - Show this help message"
