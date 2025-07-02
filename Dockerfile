FROM ghcr.io/astral-sh/uv:python3.12-bookworm-slim AS builder
ENV UV_COMPILE_BYTECODE=1 UV_LINK_MODE=copy

# Disable Python downloads, because we want to use the system interpreter
# across both images. If using a managed Python version, it needs to be
# copied from the build image into the final image; see `standalone.Dockerfile`
# for an example.

WORKDIR /app
COPY . /app
RUN apt update && apt install -y git
RUN --mount=type=cache,target=/root/.cache/uv \
    --mount=type=bind,source=uv.lock,target=uv.lock \
    --mount=type=bind,source=pyproject.toml,target=pyproject.toml \
    uv sync --locked --no-install-project --no-editable --no-dev
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --locked --no-editable --no-dev

RUN uv build
RUN mv ./dist/sysdig_mcp_server-*.tar.gz /tmp/sysdig_mcp_server.tar.gz

# Final image without uv
FROM python:3.12-slim
# It is important to use the image that matches the builder, as the path to the
# Python executable must be the same

WORKDIR /app

RUN apt update && apt install -y git
# Copy the application from the builder
COPY --from=builder --chown=app:app /tmp/sysdig_mcp_server.tar.gz /app
COPY --from=builder --chown=app:app /app/app_config.yaml /app

RUN pip install /app/sysdig_mcp_server.tar.gz

USER 1001:1001

ENTRYPOINT ["sysdig-mcp-server"]
