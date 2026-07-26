# Mass Email Campaign Sender

Send personalized email campaigns to millions of recipients via SMTP or Amazon SES, with real-time progress tracking, retry logic, and a clean web interface.

## Quick Start

```bash
# 1. Edit default configuration
cp config.yaml config.yaml  # already exists with defaults, edit as needed

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
  ses:
    region: "us-east-1"
    access_key_id: ""
    secret_access_key: ""

worker:
  concurrency: 10
  max_retries: 3
  retry_backoff_base: "1s"
  retry_backoff_max: "30s"
```

All settings can be overridden per campaign in the web interface.

## Development

### Prerequisites

- Go 1.22+
- Node.js 20+
- [Mailtrap Local](https://github.com/mailtrap/mailtrap-local) (for email testing)

### Run in development mode

```bash
# Terminal 1: Start Mailtrap Local (optional, for email testing)
docker run -p 1025:1025 -p 8025:8025 mailtrap/mailtrap-local

# Terminal 2: Build frontend
cd frontend && npm install && npm run build && cd ..

# Terminal 3: Run server
go run ./cmd/server
```

### Running tests

```bash
# Unit tests
make test

# Integration tests (requires Mailtrap Local running)
make test-integration
```

## Build

```bash
make build
```

This produces a single binary at `bin/server` with the frontend embedded.

## Architecture

```
cmd/server/         # Entry point, embeds frontend
internal/
  config/           # YAML configuration
  handler/          # HTTP handlers (Go Fiber)
  worker/           # Worker pool + retry logic
  email/            # SMTP & SES senders, template rendering
  store/            # In-memory campaign state
  campaign/         # CSV parsing, campaign orchestration, logging
frontend/           # Svelte 5 + TailwindCSS + DaisyUI
```

## Campaign Flow

1. Paste CSV data with headers (requires an `email` column)
2. Write email template with `{placeholder}` variables
3. Configure email provider (defaults pre-filled from `config.yaml`)
4. Configure worker pool (concurrency, retries)
5. Click **Dry-Run Preview** to test without sending
6. Click **Start Campaign** to begin sending
7. Watch real-time progress: sent / failed / pending

## Campaign Logs

Each campaign run creates a structured JSON log file in `logs/campaign_<timestamp>.log` containing configuration, recipient statuses, and retry attempts.
