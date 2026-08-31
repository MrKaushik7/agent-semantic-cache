# Semantic Cache: AI Agent Serving Platform

## Summary

A public Apache-2.0 monorepo demonstrating production AI engineering through:

- An OpenAI-compatible Python/FastAPI inference gateway.
- Exact and guarded semantic caching backed by Redis.
- A fictional support incident-management workload.
- Retrieval, fine-tuned cache-safety classification, and LangGraph multi-agent orchestration.
- Fault tolerance, concurrency control, observability, evaluation, cost accounting, load testing, CI/CD, containers, and Kubernetes.
- A Go CLI for workload generation, fault scenarios, and reproducible benchmarks.

## Development

Requirements:

- Python 3.13
- [uv](https://docs.astral.sh/uv/)
- Go 1.25

Install the locked Python environment and run the smoke tests from the repository root:

```bash
uv sync --locked
uv run pytest

cd loadgen
go test ./...
```

Run the configured static checks from the repository root:

```bash
uv run ruff check .
uv run mypy src tests
```

## License

Licensed under the [Apache License 2.0](LICENSE).
