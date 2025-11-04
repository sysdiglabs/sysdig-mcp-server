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
    go test ./...

# Generate code
generate:
    go generate ./...

# Run tests and generate coverage report
test-coverage: generate
    go test -coverprofile=coverage.out ./...
