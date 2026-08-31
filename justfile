# Help commands
default:
    @just --list

# Run all checks
check: fmt fix check-vulnerabilities lint test

# Check for vulnerabilities in the project
check-vulnerabilities:
    govulncheck -show verbose ./...

# Lint and fix code
lint:
    golangci-lint run

# Fix code
fix:
    go fix ./...

# Format code
fmt: fix
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

# Update all dependencies
update:
	nix flake update
	nix develop --command go get -u -t -v ./...
	nix develop --command just update-client
	nix develop --command go mod tidy
	nix develop --command just rehash-package-nix
	nix develop --command pinact run -u
	nix develop --command prek autoupdate
	nix develop --command just update-base-images

# Re-calculate the vendorHash from the package.nix
rehash-package-nix:
	sd 'vendorHash = ".*";' 'vendorHash = "";' package.nix; h="$((nix build -L --no-link .#default || true) 2>&1 | sed -nE 's/.*got:[[:space:]]+([^ ]+).*/\1/p' | tail -1)"; [ -n "$h" ] && sd 'vendorHash = ".*";' "vendorHash = \"$h\";" package.nix && echo "vendorHash -> $h"

update-base-images:
    nix-prefetch-docker --arch amd64 quay.io/sysdig/sysdig-mini-ubi9 1 > docker-base-amd64.nix
    nix-prefetch-docker --arch arm64 quay.io/sysdig/sysdig-mini-ubi9 1 > docker-base-aarch64.nix

# Regenerate internal/infra/sysdig/client.gen.go from github.com/draios/api-spec
update-client: && generate fmt
    #!/usr/bin/env bash
    set -euo pipefail
    tmp_dir="$(mktemp -d)"
    trap 'rm -rf "$tmp_dir"' EXIT
    repo_root="$(pwd)"
    git clone --depth 1 https://github.com/draios/api-spec.git "$tmp_dir"
    (cd "$tmp_dir" && nix develop --command just merge-all)
    oapi-codegen -package sysdig -generate client,types -o "$repo_root/internal/infra/sysdig/client.gen.go" "$tmp_dir/openapi.yaml"
