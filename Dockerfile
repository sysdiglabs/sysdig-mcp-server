FROM nixos/nix:latest AS builder

# Enable flakes
RUN echo "experimental-features = nix-command flakes" >> /etc/nix/nix.conf

WORKDIR /app
COPY . /app

# Build the default package from the flake
# This will produce a 'result' symlink in the working directory
RUN nix build .#default

# Final image
# quay.io/sysdig/sysdig-mini-ubi9:1
FROM quay.io/sysdig/sysdig-mini-ubi9@sha256:fa959bec7353ce38ed2d35190d5ec4445b3c525f1d4ad00e26afca387360a253

# Copy the binary from the builder stage
COPY --from=builder /app/result/bin/sysdig-mcp-server /usr/local/bin/sysdig-mcp-server

# Run as non-root user (numeric ID)
USER 1000

ENTRYPOINT ["sysdig-mcp-server"]
