FROM ghcr.io/astral-sh/uv:python3.12-bookworm-slim AS builder
ENV UV_COMPILE_BYTECODE=1 UV_LINK_MODE=copy

WORKDIR /app
COPY . /app
RUN apt update && apt install -y git
RUN --mount=type=cache,target=/root/.cache/uv \
    --mount=type=bind,source=uv.lock,target=uv.lock \
    --mount=type=bind,source=pyproject.toml,target=pyproject.toml \
    uv sync --locked --no-install-project --no-editable --no-dev
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --locked --no-editable --no-dev

RUN rm -rf ./dist
RUN uv build
RUN mv ./dist/sysdig_mcp_server-*.tar.gz /tmp/sysdig_mcp_server.tar.gz

# Final image with UBI
FROM quay.io/sysdig/sysdig-ubi:1

# Install Python 3.12 and git
RUN dnf update -y && \
    dnf install -y python3.12 python3.12-pip git && \
    dnf clean all

# Create a non-root user
RUN useradd -u 1001 -m appuser
WORKDIR /home/appuser

# Copy the application from the builder
COPY --from=builder --chown=appuser:appuser /tmp/sysdig_mcp_server.tar.gz .

# Install the application
RUN python3.12 -m pip install --no-cache-dir sysdig_mcp_server.tar.gz

USER appuser

ENTRYPOINT ["sysdig-mcp-server"]
