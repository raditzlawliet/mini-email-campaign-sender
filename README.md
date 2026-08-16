# MECS - Mini Email Campaign Sender

Desktop app (Wails v2) to send personalized email campaigns via SMTP or Amazon SES, with real-time progress tracking, pause/resume, retry logic, and a clean desktop UI.

<p align="center">
  <img src="./docs/preview.gif" alt="Preview">
</p>

## Features

- **CSV input** via native file picker or manual paste, with header-based `{placeholder}` personalization
- **Pre-configured defaults** from `config.yaml` — all settings overridable per campaign in the UI
- **Dry-run preview** — render sample emails in sandboxed iframes (Render/Code tabs) without sending
- **Email providers** — SMTP, SES, and SES Templates (see [Supported Providers](#supported-providers) below)
- **Worker pool** — concurrent goroutines with per-worker dedicated sender instances
- **Retry logic** — exponential backoff with configurable max attempts and base/max durations
- **Pause / Resume** — graceful pause (in-flight email completes), resume processes only pending recipients
- **Real-time progress** — Wails Events push progress bar + log events to frontend every 1s
- **Session restore** — app restart restores in-progress campaign state
- **Campaign logging** — optional per-run JSON log file with full configuration and delivery tracking
- **Verbose mode** — per-email debug details in frontend and log file
- **Multi-language** — English, Arabic (RTL), and Bahasa Indonesia
- **Theme picker** — choose from all DaisyUI themes, auto-persisted to your config
- **Secure credentials** — SMTP and SES credentials automatically stored in OS keyring
- **MCP server (AI agents)** — embedded Model Context Protocol server on localhost so AI agents can prepare, dry-run, start, pause, resume, monitor, and clear campaigns (see [MCP Server](#mcp-server-ai-agents))

## Supported Providers

| Provider          | Method                   | Batch                   | Description                                                                                    |
| ----------------- | ------------------------ | ----------------------- | ---------------------------------------------------------------------------------------------- |
| **SMTP**          | `DialAndSend`            | ✅ configurable (1–500) | Direct SMTP connection, batched per connection for throughput                                  |
| **SES**           | `SendRawEmail`           | ❌ per-email            | Raw email sending via Amazon SES API                                                           |
| **SES Templates** | `SendBulkTemplatedEmail` | ✅ configurable (1–50)  | Marketing emails via SES templates; subject/body from template, CSV data as template variables |

## Download

You can try latest pre-built binaries from the [releases page](https://github.com/raditzlawliet/mini-email-campaign-sender/releases).

## Quick Start

```bash
# Run on Linux
chmod +x ./mecs
./mecs

# Run on Windows
./mecs.exe
```

## Configuration

By default if no config provided, it will automatically create `config.yaml` on user's home directory on `.mecs` folder.
You can edit `config.yaml` to set defaults for email provider, worker pool, and logging:

```yaml
email:
  provider: smtp # "smtp" or "ses"
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
    use_template: false
    template_name: ""
    batch_size: 50

worker:
  concurrency: 10
  max_retries: 3
  retry_backoff_base: "1s"
  retry_backoff_max: "30s"

log:
  campaign:
    log_to_file: true # write campaign events to logs/campaign_*.log
    verbose: false # include per-email debug detail

mcp:
  enabled: true # embedded MCP server for AI agents (localhost only)
  host: "127.0.0.1"
  port: 18799
  token: "" # optional bearer token; empty = no auth
```

| Section        | Key                  | Default     | Description                                                    |
| -------------- | -------------------- | ----------- | -------------------------------------------------------------- |
| `email`        | `provider`           | `smtp`      | Email provider: `smtp` or `ses`                                |
| `email`        | `from`               | —           | Sender email address                                           |
| `email.smtp`   | `host`               | `localhost` | SMTP server hostname                                           |
| `email.smtp`   | `port`               | `1025`      | SMTP server port                                               |
| `email.smtp`   | `username`           | —           | SMTP auth username (optional)                                  |
| `email.smtp`   | `password`           | —           | SMTP auth password (optional), automatically stored to keyring |
| `email.smtp`   | `tls`                | `false`     | Enable TLS                                                     |
| `email.smtp`   | `batch_size`         | `50`        | Emails per SMTP connection                                     |
| `email.ses`    | `region`             | —           | AWS region (e.g. `us-east-1`)                                  |
| `email.ses`    | `access_key_id`      | —           | AWS access key ID, automatically stored to keyring             |
| `email.ses`    | `secret_access_key`  | —           | AWS secret access key, automatically stored to keyring         |
| `email.ses`    | `use_template`       | `false`     | Use SES template (marketing)                                   |
| `email.ses`    | `template_name`      | —           | SES template name                                              |
| `email.ses`    | `batch_size`         | `50`        | Emails per Bulk API call (1–50)                                |
| `worker`       | `concurrency`        | `10`        | Parallel worker goroutines                                     |
| `worker`       | `max_retries`        | `3`         | Max retry attempts per email                                   |
| `worker`       | `retry_backoff_base` | `1s`        | Initial retry delay                                            |
| `worker`       | `retry_backoff_max`  | `30s`       | Max retry delay cap                                            |
| `log.campaign` | `log_to_file`        | `true`      | Write per-run JSON log file                                    |
| `log.campaign` | `verbose`            | `false`     | Enable debug-level detail                                      |
| `mcp`          | `enabled`            | `true`      | Enable the embedded MCP server (binds to `host`)              |
| `mcp`          | `host`               | `127.0.0.1` | Listen address (keep localhost for security)                  |
| `mcp`          | `port`               | `18799`     | MCP endpoint port                                             |
| `mcp`          | `token`              | —           | Optional bearer token; clients must send `Authorization: Bearer <token>` |

## Development

### Prerequisites

- Go 1.25+
- Node.js 20+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation) v2.13.0 (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0`)
- [Mailpit](https://github.com/axllent/mailpit) (for email testing)
- **Windows**: WebView2 runtime (pre-installed on Windows 11)
- **Linux**: WebKitGTK (`libwebkit2gtk-4.1-dev`) + GTK3 dev packages. Build with `-tags webkit2_41` (4.0 is dropped on Ubuntu 24.04+).

### Run in development mode

```bash
make dev
```

Runs `wails dev` — hot-reload for both Go backend and Svelte frontend (Vite).

### Email testing with Mailpit

```bash
docker run -d -p 1025:1025 -p 8025:8025 --name mailpit axllent/mailpit
```

Web interface at [http://localhost:8025](http://localhost:8025) to view captured emails.

### Running tests

```bash
make test                 # Unit tests
make test-integration     # Integration tests (requires Mailpit)
```

## Build

```bash
make build                # Current OS
make build-windows        # Windows amd64 + NSIS installer
make build-linux          # Linux amd64
make build-all            # Both platforms
```

Produces binaries in `build/bin/` with the frontend embedded.

## Makefile Targets

| Target              | Purpose                               |
| ------------------- | ------------------------------------- |
| `make build`        | Current-OS Wails build                |
| `make build-windows`| Windows amd64 + NSIS installer        |
| `make build-linux`  | Linux amd64 binary                    |
| `make build-all`    | Build both platforms                  |
| `make dev`          | `wails dev` (Go + Vite hot-reload)    |
| `make test`         | `go test ./...`                       |
| `make test-integration` | Integration tests                 |
| `make clean`        | Remove build artifacts                |

## Architecture

```
main.go                  # Wails entry point (bindings, asset server)
internal/
  app/                   # Wails App struct + bound methods (replaces HTTP handlers)
  config/                # YAML configuration
  worker/                # Worker pool, retry, RunPending for resume
  email/                 # SMTP (batched) & SES senders, template rendering
  store/                 # In-memory campaign state, event log, verbose flag
  campaign/              # CSV parsing, campaign orchestration, file logging
  mcp/                   # Embedded MCP server (go-sdk, streamable HTTP, 9 tools)
frontend/                # Svelte 5 + TailwindCSS v4 + DaisyUI v5 (Wails WebView)
```

## Wails Bindings

| Method              | Purpose                                                                 |
| ------------------- | ----------------------------------------------------------------------- |
| `GetVersion`        | Return app version (set via `-ldflags -X main.version`)                 |
| `GetCampaignConfig` | Return default configuration + current campaign state (session restore) |
| `Preview`           | Parse CSV (text or file path) + render N sample emails                  |
| `StartCampaign`     | Parse CSV (text or file path) + start worker pool                       |
| `PauseCampaign`     | Gracefully pause (wait for in-flight send, cancel ctx)                  |
| `ResumeCampaign`    | Resume processing pending recipients only                               |
| `ResetCampaign`     | Clear all campaign state (works on paused too)                          |
| `SaveConfig`        | Deep-merge partial JSON into config.yaml, reload in-memory config       |
| `PickCSVFile`       | Native file dialog, returns CSV path (Go reads file)                    |
| `ParseCSVFile`      | Parse picked CSV, return headers + recipient count                      |

Progress + log events stream via Wails Events (`campaign:progress`), emitted from Go every 1s.

## Campaign Flow

1. **Input Data** — Pick a `.csv` file via native dialog or paste CSV manually. First row is headers, requires an `email` column. Available headers shown as copyable badges.
2. **Email Template** — Configure To, Subject, and Body with `{placeholder}` variables matching CSV headers.
3. **Email Provider** — Select SMTP or SES (defaults from `config.yaml`). Override host, port, credentials, TLS, and batch size per campaign. SES supports template mode for marketing emails — when enabled, subject/body are defined by the SES template and CSV data becomes template variables, sent via batched `SendBulkTemplatedEmail`.
4. **Worker Config** — Adjust concurrency, max retries, and backoff settings.
5. **Log Config** — Toggle per-run file logging and verbose debug output.
6. **Dry-Run Preview** — Render sample emails in sandboxed iframes (Render/Code tabs) without sending.
7. **Start Campaign** — Begin sending. Real-time progress bar and log stream via Wails Events.
8. **Pause / Resume** — Pause gracefully (in-flight email completes). Resume processes only pending recipients.
9. **Reset** — Clear all state and start fresh.

## Campaign Logs

Each campaign run can write a structured JSON log file at `logs/campaign_<timestamp>.log`:

- Set `log.campaign.log_to_file: false` to disable file logging
- Set `log.campaign.verbose: true` to include debug-level per-email events

## MCP Server (AI Agents)

MECS embeds a [Model Context Protocol](https://modelcontextprotocol.io) server (official `github.com/modelcontextprotocol/go-sdk`) served over streamable HTTP on localhost, so AI agents (Claude Code, Zed, any MCP client) can drive campaigns programmatically. The endpoint is `http://127.0.0.1:18799/mcp` (configurable via the `mcp` section, see [Configuration](#configuration)). A plain health check is available at `/mcp/health`.

AI actions share the same state as the UI: a campaign started via MCP shows live in the frontend and vice versa. Secrets (SMTP password, SES keys, MCP token) are always masked in read responses.

### Tools

| Tool | Purpose |
| --- | --- |
| `mecs_prepare_campaign` | Stage a campaign without sending: CSV (text or file path) + partial template, provider, worker, and log settings (empty values keep current staging) |
| `mecs_get_current_campaign` | Current state: lifecycle, progress counters, staged template/config, optional recent events |
| `mecs_set_config` | Deep-merge partial JSON into `config.yaml` (same as the UI's Save as defaults) |
| `mecs_get_config` | Current global config, optionally filtered by section (`app`, `email`, `worker`, `log`, `mcp`) |
| `mecs_dry_run_campaign` | Render sample emails (max 5) from the prepared campaign without sending |
| `mecs_start_campaign` | Start sending the prepared campaign (global defaults fill any gaps) |
| `mecs_pause_campaign` | Gracefully pause the running campaign |
| `mecs_resume_campaign` | Resume a paused campaign (pending recipients only) |
| `mecs_clear_campaign` | Clear all campaign state back to idle |

### Example client setup (Claude Code)

Add a local MCP server entry pointing at MECS (run MECS first):

```json
{
  "mcpServers": {
    "mecs": {
      "url": "http://127.0.0.1:18799/mcp"
    }
  }
}
```

If a token is configured, add `"headers": { "Authorization": "Bearer <token>" }` to the `mecs` entry.

### Security notes

- The server binds to `127.0.0.1` by default - only processes on the same machine can connect. Set `mcp.host` to a different interface only if you understand the risk.
- Set `mcp.token` to require `Authorization: Bearer <token>` on every request.
- Changing `mcp.*` via `mecs_set_config` or the UI restarts the server with the new settings immediately.
- `csv_path` allows an agent to read any file the MECS process can read (same user).

## Contributors

Contributions are welcome! Check the [Issues](https://github.com/raditzlawliet/mini-email-campaign-sender/issues) page or open a Pull Request.

<div align="center">
    <a href="https://github.com/raditzlawliet/mini-email-campaign-sender/graphs/contributors">
        <img src="https://contrib.rocks/image?repo=raditzlawliet/mini-email-campaign-sender" />
    </a>
</div>

## License

MIT
