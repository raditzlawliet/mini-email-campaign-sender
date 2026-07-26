# AGENTS.md — Mini Email Campaign Sender

> AI agent configuration. Update as the project evolves. Keep it short.

## Project

Single-page app to send personalized email campaigns to 1M recipients per campaign via SMTP/SES, with queue-based processing, retries, and real-time progress.

- **Frontend**: Svelte 5 (runes), TailwindCSS v4, DaisyUI v5, Embed in Go binary.
- **Backend**: Go, Go Fiber v3, `wneessen/go-mail` for email sending, `slog` for logging.
- **Email Testing**: Mailtrap Local (`mailtrap-local`)
- **Config**: File-based YAML for server port, email provider, retry params, log behavior
- **Dev mode**: `DEV_MODE=true` enables CORS + API-only backend; Vite proxies `/api` to backend on `:8080`.

## Spec Flow

```
ANALYZE → PLAN → CLARIFY (if needed) → EXECUTE → TEST
```

## API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/campaign/config` | Return default configuration + current campaign state (session restore) |
| POST | `/api/campaign/preview` | Parse CSV (multipart/form-data) + render N sample emails |
| POST | `/api/campaign/start` | Parse CSV (multipart/form-data) + start worker pool |
| POST | `/api/campaign/pause` | Gracefully pause (wait for in-flight send, cancel ctx) |
| POST | `/api/campaign/resume` | Resume processing pending recipients only |
| GET | `/api/campaign/events` | SSE stream: pushes progress + log events every 1s |
| POST | `/api/campaign/reset` | Clear all campaign state (works on paused too) |

Preview and Start use `multipart/form-data`. CSV sent as file (`csv_file`) or text field (`csv_text`). Other fields: `subject`, `body`, `to`, `from`, `provider`, `smtp_host`, `smtp_port`, `smtp_username`, `smtp_password`, `smtp_tls`, `smtp_batch_size`, `ses_region`, `ses_access_key_id`, `ses_secret_access_key`, `concurrency`, `max_retries`, `retry_backoff_base`, `retry_backoff_max`, `log_to_file`, `verbose`. Preview also accepts `count`.

## Naming Conventions

### Go (Backend)

| Concern | Convention | Example |
|---------|-----------|---------|
| Packages | lowercase, single word | `worker`, `email`, `config` |
| Files | snake_case | `email_sender.go`, `worker_pool.go` |
| Types | PascalCase | `Campaign`, `RecipientStatus` |
| Functions | PascalCase (exported), camelCase (private) | `SendEmail()`, `parseCSV()` |
| Errors | `Err` prefix | `ErrInvalidCSV` |
| HTTP handlers | `Handle` prefix (exported) | `HandlePreview`, `HandleProgress` |
| Test files | `*_test.go` | `worker_pool_test.go` |

### Svelte 5 (Frontend)

| Concern | Convention | Notes |
|---------|-----------|-------|
| Components | PascalCase | `CampaignForm.svelte`, `InputData.svelte` |
| Runes | `$state`, `$derived`, `$effect`, `$props`, `$bindable` | — |
| Props | camelCase | `let { toField } = $props()` |
| Events | callback props | `let { onreset } = $props()` |
| Slots | `{@render}` snippets | — |
| Input binding | `onblur` sync for `$bindable` | Local `_value` state, flush on blur |

## Directory Structure

```
cmd/server/              # main entrypoint, embedded frontend
internal/
  config/                # YAML config loading
  handler/               # 7 HTTP handlers (Go Fiber v3)
  worker/                # worker pool, queue, retry logic, RunPending for resume
  email/                 # SMTP + SES senders, template rendering, batched SMTP
  store/                 # in-memory recipient status store
  campaign/              # CSV parsing, campaign orchestration, file logging
frontend/
  src/
    lib/
      components/
        CampaignForm.svelte   # orchestrator: state, API calls, tabs, progress
        InputData.svelte      # file picker + manual toggle + multipart file emit
        EmailTemplate.svelte  # To, Subject, Body + body preview modal
        ProviderConfig.svelte # SMTP/SES config, batch_size, fieldset+label+input
        WorkerConfig.svelte   # concurrency, retries, backoff
        LogConfig.svelte      # log_to_file and verbose toggles
        PreviewModal.svelte   # iframe sandbox render, code tab, next/prev
    App.svelte
    main.js
    app.css               # TailwindCSS v4 + DaisyUI v5
```

## Go Conventions

- `slog` for all logging: `LevelError`/`LevelInfo`/`LevelDebug`.
- Placeholders `{key}` rendered via `strings.NewReplacer`.
- Worker pool: context cancellation, exponential backoff retry. Each worker goroutine creates its own `EmailSender` via `SenderFactory`. SMTP senders batch messages using a single `DialAndSend` call per batch (configurable `batch_size`, default 50). Flush on worker completion.
- Pause: cancels context between sends (in-flight email completes gracefully). Resume: `RunPending` processes only `pending` recipients.
- Unified logging: `Store.LogAndEvent(level, msg)` logs to both `slog` (console) and in-memory events (frontend). `CampaignLogger.Log(level, msg)` writes the same message to a JSON file per run.
- Campaign log file per run: `logs/campaign_<timestamp>.log` with simple JSON lines `{"time":"...","level":"...","msg":"..."}`. Controlled by `log.campaign.log_to_file` in config.
- Verbose logging: `log.campaign.verbose: true` enables debug-level per-email events in frontend (SSE) and campaign log file. Terminal output follows slog handler level (Info by default suppresses debug).
- CSV upload: large files (100MB+) sent via `multipart/form-data` (`csv_file` field) to `parseMultipart()` in handler. Manual input sent as `csv_text` text field. Avoids JSON body size limits.
- Test with `testify` (`assert`/`require`), each subtest gets its own `assert.New(t)`.
- Slices/maps initialized explicitly (never nil).
- Integration tests use `//go:build integration` + Mailtrap Local.

## Svelte 5 Conventions

- **Local state + onblur**: Child components with `$bindable` props use local `$state` copy (`_value`) bound to inputs. Sync to parent only on `onblur` via `flush()`. Avoids INP delays from per-keystroke parent re-renders.
- **Multipart upload**: CSV file sent via `FormData` (no manual `Content-Type` — browser sets boundary). `csvFile` prop tracked in `InputData`, used by `CampaignForm.buildFormData()` for preview/start.
- **Form pattern**: `<fieldset class="fieldset">` wrapping `<label class="label" for="id">` + `<input id="..." class="input w-full">`.
- **Iframe sandbox**: Preview modes use sandboxed iframes with `srcdoc` to isolate email content.
- **Log panel**: Collapsable below progress, auto-scrolls when scrolled to bottom, shows timestamped events from backend.
- **SSE**: Single `EventSource` connected on mount via `GET /api/campaign/events`, never disconnected. Streams progress + log every 1s.
- **Session restore**: On page refresh, `GET /api/campaign/config` returns current campaign state (template, config, progress, events). Form restores to in-progress state if campaign is running/paused/completed.
- **Form lock**: Input data, template, provider, and worker config are disabled when campaign is running or paused. Reset button disabled only when `running` (enabled on paused/completed).
- **Icons**: Use `@lucide/svelte` for icons (ChevronRight/Down, X, ChevronLeft/Right).
- **Literal braces**: Use `{'{name}'}` syntax in template HTML to output literal `{name}`.

## Testing Rules

1. Every Go package MUST have unit tests.
2. Table-driven tests with `t.Run`.
3. `go test ./...` and `go vet ./...` must pass before done.
4. Frontend: handle loading, empty, error states visually.

## Makefile Targets

| Target | Purpose |
|--------|---------|
| `make build` | Build frontend + embed + Go binary |
| `make dev` | Production mode single binary |
| `make dev-be` | Backend only (API + CORS for `:5173`) |
| `make dev-fe` | Vite dev server with HMR |
| `make test` | `go test ./...` |
| `make clean` | Remove build artifacts |
