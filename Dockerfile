FROM ghcr.io/astral-sh/uv:python3.12-bookworm-slim AS builder
ENV UV_COMPILE_BYTECODE=1 UV_LINK_MODE=copy

# Disable Python downloads, because we want to use the system interpreter
# across both images. If using a managed Python version, it needs to be
# copied from the build image into the final image; see `standalone.Dockerfile`
# for an example.

WORKDIR /app
COPY . /app
RUN --mount=type=cache,target=/root/.cache/uv \
--mount=type=bind,source=uv.lock,target=uv.lock \
--mount=type=bind,source=pyproject.toml,target=pyproject.toml \
uv sync --locked --no-install-project --no-editable
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --locked --no-editable

# Dinal image without uv
FROM python:3.12-slim
# It is important to use the image that matches the builder, as the path to the
# Python executable must be the same

# Copy the application from the builder
COPY --from=builder --chown=app:app /app /app

WORKDIR /app

# Place executables in the environment at the front of the path
ENV PATH="/app/.venv/bin:$PATH"

ENTRYPOINT ["/bin/sh", "entrypoint.sh"]
