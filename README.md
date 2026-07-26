# Mini Email Campaign Sender

Send personalized email campaigns via SMTP or Amazon SES, with real-time progress tracking, pause/resume, retry logic, and a clean web interface.

<p align="center">
  <img src="./docs/preview.png" alt="Preview">
</p>

## Quick Start

```bash
# 1. Edit default configuration (config.yaml has sensible defaults)
# 2. Build the application
make build

# 3. Run
./bin/server
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

## Configuration

Edit `config.yaml` to set defaults for server, email provider, and worker pool:

```yaml
server:
  port: 8080

email:
  provider: smtp           # "smtp" or "ses"
  from: "sender@example.com"
  smtp:
    host: "localhost"
    port: 1025
    username: ""
    password: ""
    tls: false
    batch_size: 50
  ses:
    region: "us-east-1"
    access_key_id: ""
    secret_access_key: ""

worker:
  concurrency: 10
  max_retries: 3
  retry_backoff_base: "1s"
  retry_backoff_max: "30s"

log:
  campaign:
    log_to_file: true
    verbose: false
```

All settings can be overridden per campaign in the web interface.

## Development

### Prerequisites

- Go 1.22+
- Node.js 20+
- [Mailtrap Local](https://github.com/mailtrap/mailtrap-local) (for email testing)

### Frontend + Backend dev mode

```bash
# Terminal 1: Backend (API only + CORS)
make dev-be

# Terminal 2: Frontend (Vite HMR, proxies /api to :8080)
make dev-fe
```

Open [http://localhost:5173](http://localhost:5173) in your browser.

### Email testing with Mailtrap Local

```bash
docker run -p 1025:1025 -p 8025:8025 mailtrap/mailtrap-local
```

Web interface at [http://localhost:8025](http://localhost:8025) to view captured emails.

### Running tests

```bash
make test                 # Unit tests
make test-integration     # Integration tests (requires Mailtrap Local)
```

## Build

```bash
make build
```

Produces a single binary at `bin/server` with the frontend embedded.

## Makefile Targets

| Target | Purpose |
|--------|---------|
| `make build` | Build frontend + embed + Go binary |
| `make dev` | Production mode single binary |
| `make dev-be` | Backend only (API + CORS for `:5173`) |
| `make dev-fe` | Vite dev server with HMR |
| `make test` | `go test ./...` |
| `make test-integration` | Integration tests |
| `make clean` | Remove build artifacts |

## Architecture

```
cmd/server/              # Entry point, embeds frontend, SSE
internal/
  config/                # YAML configuration
  handler/               # HTTP handlers (Go Fiber v3)
  worker/                # Worker pool + retry + RunPending for resume
  email/                 # SMTP & SES senders, template rendering
  store/                 # In-memory campaign state + event log
  campaign/              # CSV parsing, campaign orchestration, file logging
frontend/                # Svelte 5 + TailwindCSS v4 + DaisyUI v5
```

## API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/campaign/config` | Return default configuration |
| POST | `/api/campaign/preview` | Parse CSV + render sample emails |
| POST | `/api/campaign/start` | Parse CSV + start worker pool |
| POST | `/api/campaign/pause` | Gracefully pause campaign |
| POST | `/api/campaign/resume` | Resume processing pending recipients |
| GET | `/api/campaign/events` | SSE stream: progress + log every 1s |
| POST | `/api/campaign/reset` | Clear all campaign state |

## Campaign Flow

1. **Input Data** — Upload a .csv file or paste CSV manually. First row is headers, requires an `email` column.
2. **Email Template** — Configure To, Subject, and Body with `{placeholder}` variables. Click 🔍 to preview the body.
3. **Email Provider** — Select SMTP or SES (defaults pre-filled from `config.yaml`). Override as needed.
4. **Worker Config** — Adjust concurrency, max retries, and backoff settings.
5. **Dry-Run Preview** — Render sample emails in sandboxed iframes (Render/Code tabs) without sending.
6. **Start Campaign** — Begin sending. Real-time progress bar and log stream via SSE.
7. **Pause / Resume** — Pause gracefully (in-flight email completes). Resume processes only pending recipients.
8. **Reset** — Clear all state and start fresh.

## Campaign Logs

- **Console**: Human-readable log messages via `slog` during campaign run.
- **Frontend**: Collapsable log panel with auto-scroll, live-updated via SSE.
- **File**: Structured JSON log at `logs/campaign_<timestamp>.log` with full configuration, recipient statuses, and retry attempts.
