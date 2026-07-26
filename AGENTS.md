# AGENTS.md — Mass Email Campaign Sender

> AI agent configuration. Update as the project evolves. Keep it short.

## Project

Single-page app to send personalized email campaigns to 1M recipients per campaign via SMTP/SES, with queue-based processing, retries, and real-time progress.

- **Frontend**: Svelte 5 (runes), TailwindCSS v4, DaisyUI v5, Embed in Go binary.
- **Backend**: Go, Go Fiber v3, `wneessen/go-mail` for email sending, `slog` for logging.
- **Email Testing**: Mailtrap Local (`mailtrap-local`)
- **Config**: File-based YAML for server port, email provider, retry params
- **Dev mode**: `DEV_MODE=true` enables CORS + API-only backend; Vite proxies `/api` to backend on `:8080`.

## Spec Flow

```
ANALYZE → PLAN → CLARIFY (if needed) → EXECUTE → TEST
```

## API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/campaign/config` | Return default configuration |
| POST | `/api/campaign/preview` | Parse CSV + render N sample emails |
| POST | `/api/campaign/start` | Parse CSV + start worker pool |
| POST | `/api/campaign/pause` | Gracefully pause (wait for in-flight send, cancel ctx) |
| POST | `/api/campaign/resume` | Resume processing pending recipients only |
| GET | `/api/campaign/progress` | Poll sent/failed/pending counts + state |
| GET | `/api/campaign/log` | Return in-memory campaign log events |
| POST | `/api/campaign/reset` | Clear all campaign state (works on paused too) |

Preview and Start accept the full payload: `csv`, `subject`, `body`, `to`, `from`, `provider`, `smtp`, `ses`, `concurrency`, `max_retries`, `retry_backoff_base`, `retry_backoff_max`.

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
  handler/               # 8 HTTP handlers (Go Fiber v3)
  worker/                # worker pool, queue, retry logic, RunPending for resume
  email/                 # SMTP + SES senders, template rendering
  store/                 # in-memory recipient status store
  campaign/              # CSV parsing, campaign orchestration, file logging
frontend/
  src/
    lib/
      components/
        CampaignForm.svelte   # orchestrator: state, API calls, progress
        InputData.svelte      # file picker + manual toggle
        EmailTemplate.svelte  # To, Subject, Body + body preview modal
        ProviderConfig.svelte # SMTP/SES config, fieldset+label+input
        WorkerConfig.svelte   # concurrency, retries, backoff
        PreviewModal.svelte   # iframe sandbox render, code tab, next/prev
    App.svelte
    main.js
    app.css               # TailwindCSS v4 + DaisyUI v5
```

## Go Conventions

- `slog` for all logging: `LevelError`/`LevelInfo`/`LevelDebug`.
- Placeholders `{key}` rendered via `strings.NewReplacer`.
- Worker pool: context cancellation, exponential backoff retry.
- Pause: cancels context between sends (in-flight email completes gracefully). Resume: `RunPending` processes only `pending` recipients.
- Campaign log stored in-memory (max 500 events) + file-based JSON log per run.
- Test with `testify` (`assert`/`require`), each subtest gets its own `assert.New(t)`.
- Slices/maps initialized explicitly (never nil).
- Integration tests use `//go:build integration` + Mailtrap Local.

## Svelte 5 Conventions

- **Local state + onblur**: Child components with `$bindable` props use local `$state` copy (`_value`) bound to inputs. Sync to parent only on `onblur` via `flush()`. Avoids INP delays from per-keystroke parent re-renders.
- **Form pattern**: `<fieldset class="fieldset">` wrapping `<label class="label" for="id">` + `<input id="..." class="input w-full">`.
- **Iframe sandbox**: Preview modes use sandboxed iframes with `srcdoc` to isolate email content.
- **Log panel**: Collapsable below progress, auto-scrolls when scrolled to bottom, shows timestamped events from backend.
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
