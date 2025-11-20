# Help commands
default:
    @just --list

# Run all checks
check: fmt lint test

# Lint and fix code
lint:
    golangci-lint run

# Format code
fmt:
    gofumpt -w .

# Ejecutar tests
test: generate
    ginkgo -r -p

# Generate code
generate:
    go generate ./...

# Run tests and generate coverage report
test-coverage: generate
    go test -coverprofile=coverage.out ./...

bump:
	nix flake update
	nix develop --command go get -u -t -v ./...
	nix develop --command go mod tidy
	nix develop --command just rehash-package-nix

rehash-package-nix:
	sd 'vendorHash = ".*";' 'vendorHash = "";' package.nix; h="$((nix build -L --no-link .#default || true) 2>&1 | sed -nE 's/.*got:[[:space:]]+([^ ]+).*/\1/p' | tail -1)"; [ -n "$h" ] && sd 'vendorHash = ".*";' "vendorHash = \"$h\";" package.nix && echo "vendorHash -> $h"
