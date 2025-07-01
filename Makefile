.PHONY: help init lint fmt test test-coverage

help:
	@echo "Available commands:"
	@echo "  make init    - Install dependencies"
	@echo "  make lint    - Lint and fix code"
	@echo "  make fmt     - Format code"
	@echo "  make test    - Run tests"
	@echo "  make test-coverage - Run tests and generate coverage report"

init:
	uv sync

lint:
	uvx ruff check --fix --config ruff.toml

fmt:
	uvx ruff format --config ruff.toml

test:
	uv run pytest --capture=tee-sys --junitxml=pytest.xml

test-coverage:
	uv run pytest --cov=. --cov-report=xml
