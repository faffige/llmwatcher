# llmWatcher

Open-source, self-hostable LLM observability platform. A lightweight alternative to Helicone and Langfuse — single binary, zero external dependencies.

## What it does

llmWatcher sits as a reverse proxy between your application and LLM providers. It transparently captures every API call, parses token usage and metadata, and persists it to a local SQLite database.

```
Your App  -->  llmWatcher (reverse proxy)  -->  OpenAI / Anthropic / Google
                    |
                    v
               SQLite + OTel metrics/traces/logs + Grafana dashboards
```

## Features

- **Reverse proxy** — routes requests to upstream LLM providers, passing auth headers through untouched
- **OpenAI chat completions parsing** — extracts model, token counts, and error info from both streaming (SSE) and non-streaming responses
- **Automatic streaming support** — injects `stream_options.include_usage: true` into streaming requests so token counts are always captured
- **SQLite storage** — persists call records with WAL mode for performance (pure Go, no CGO)
- **Async pipeline** — buffered channel with background worker, so recording never adds latency to your API calls
- **ULID identifiers** — time-sortable, unique IDs for every recorded call
- **OTel-native observability** — metrics, traces, and logs via OpenTelemetry (with Prometheus `/metrics` endpoint for Grafana)
- **Configurable** — YAML config file, environment variables, or CLI flags (via koanf)

## Quick start

```bash
# Build
make build

# Run (proxies OpenAI by default on :8080)
./bin/llmwatcher

# Or with a custom database path
./bin/llmwatcher --db /path/to/llmwatcher.db

# Or with a config file
./bin/llmwatcher --config config.yaml
```

Then point your OpenAI client at `http://127.0.0.1:8080/v1/openai/` instead of `https://api.openai.com/`.

```bash
curl http://127.0.0.1:8080/v1/openai/v1/chat/completions \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4o", "messages": [{"role": "user", "content": "Hello"}]}'
```

The request is proxied to OpenAI, and the call metadata (model, tokens, duration, status) is recorded to SQLite in the background.

## Configuration

llmWatcher uses koanf with the following precedence: **environment variables > config file > defaults**.

Environment variables are prefixed with `LLMWATCHER_` and use underscores for nesting:

```bash
LLMWATCHER_SERVER_HOST=0.0.0.0
LLMWATCHER_SERVER_PROXY_PORT=8080
```

Example YAML config:

```yaml
server:
  host: 127.0.0.1
  proxy_port: 8080
  metrics_port: 9090

providers:
  openai:
    upstream: https://api.openai.com
    enabled: true
  anthropic:
    upstream: https://api.anthropic.com
    enabled: false
  google:
    upstream: https://generativelanguage.googleapis.com
    enabled: false
```

## Development

```bash
make build    # Build binary to bin/llmwatcher
make test     # Run tests with race detection and coverage
make lint     # Run golangci-lint
make clean    # Remove build artefacts
```

## Architecture

```
cmd/llmwatcher/         Entry point
internal/
  config/               Configuration loading (koanf)
  proxy/                Reverse proxy + recorder middleware
  provider/             Parser interface + CallRecord data model
  provider/openai/      OpenAI response parser
  pipeline/             Async buffered channel → storage writer
  storage/              Store interface
  storage/sqlite/       SQLite backend (modernc.org/sqlite)
```

## Licence

Apache 2.0 — see [LICENCE](LICENCE).
