# AGENTS.md — Mini Email Campaign Sender

> AI agent configuration. Update as the project evolves. Keep it short.
> You, AI, should update this AGENTS.md as much as needed as the project evolve.

## Project

Single-page app to send personalized email campaigns to 1M recipients per campaign via SMTP/SES, with queue-based processing, retries, and real-time progress.

- **Frontend**: Svelte 5 (runes), TailwindCSS v4, DaisyUI v5, Embed in Go binary.
- **Backend**: Go, Go Fiber v3, `wneessen/go-mail` for email sending, `slog` for logging.
- **Email Testing**: Mailtrap Local (`mailtrap-local`)
- **Config**: YAML at `~/.mecs/config.yaml` (default) or `CONFIG_PATH` env or `./config.yaml` (legacy). Sensitive fields (SMTP password, SES keys) stored in OS keyring with `_provider` metadata blocks. Partial save via `POST /api/config/save` with key-order preservation (yaml.Node merge).
- **Editor**: Zed with Svelte extension.
- **Dev mode**: `DEV_MODE=true` enables CORS + API-only backend; Vite proxies `/api` to backend on `:8080`.

## Rule

- No Arrow `←` or `→` or long dash `—` ; Use simple hyphen (-) instead or <- or ->
- No `─` on code either comments/content; except for drawing sequence in markdown or explicit needed

## Your workflow

1. **Understand the request** — read the user's task carefully before doing anything.
2. **Explore the codebase** — use read-only tools to map everything relevant:
   - What files/components/hooks/services exist today that relate to the task?
   - What is the current data flow, component tree, or logic path?
   - What patterns and conventions does the project use that this task must follow?
   - What are the edge cases or dependencies that could affect the implementation?
3. **Draft the plan** following the exact structure defined below.
4. **Save the plan** to `.zed/plans/` inside the project root using the **`edit_file` tool** (write mode):
   - Path: `<project-root>/.zed/plans/YYYY-MM-DD-short-description.plan.md`
   - Replace `YYYY-MM-DD` with the actual current date.
   - Replace `short-description` with 3–5 kebab-case words summarising the task (e.g. `add-concession-step`, `fix-navigation-refresh`, `refactor-product-context`).
   - **Never use a `cat` heredoc** — it breaks on plan content that contains backticks.
   - for minor changes, you can skip to save the plan file.
5. **Clarify** — if any part of the plan is unclear, ask the user for clarification before proceeding.
6. **Confirm** to the user: print the full path where the plan was saved and give a one-paragraph plain-English summary of what the plan covers.
7. **Execute the plan** — implement the changes in code. Use the `edit_file` tool for all code edits.
8. **Test** — run the relevant tests (unit, integration, UI) to ensure the changes work as expected. If any tests fail, debug and fix the issues.
9. Update all file docs needed like `AGENTS.md` file with any new context or notes that will help future tasks. `README.md` if any public-facing changes.
10. **Commit** — create commit with a clear message summarizing the work done. Include references to any relevant issues or tasks. (No need to commit, user will review first, do if asked)

## Plan Structure

Every plan MUST follow this exact structure — do not skip or reorder sections:

---

name: [Short Plan Name — title case, e.g. "Add Concession Step"]
overview: [One sentence describing what this plan achieves]
todos:

- id: [kebab-id-matching-step-1]
  content: [Exact description of Step 1]
  status: pending
- id: [kebab-id-matching-step-2]
  content: [Exact description of Step 2]
  status: pending
  [one todo per step, in the same order as Section 4]
  isProject: false

---

# [Plan Name]

> **Created:** YYYY-MM-DD
> **Request:** "[the user's original request, quoted verbatim]"
> **Status:** 🟡 Pending

---

## API Endpoints

| Method | Path                    | Purpose                                                                 |
| ------ | ----------------------- | ----------------------------------------------------------------------- |
| GET    | `/api/version`          | Return app version (set via `-ldflags -X main.version`)                 |
| GET    | `/api/campaign/config`  | Return default configuration + current campaign state (session restore) |
| POST   | `/api/campaign/preview` | Parse CSV (multipart/form-data) + render N sample emails                |
| POST   | `/api/campaign/start`   | Parse CSV (multipart/form-data) + start worker pool                     |
| POST   | `/api/campaign/pause`   | Gracefully pause (wait for in-flight send, cancel ctx)                  |
| POST   | `/api/campaign/resume`  | Resume processing pending recipients only                               |
| GET    | `/api/campaign/events`  | SSE stream: pushes progress + log events every 1s                       |
| POST   | `/api/campaign/reset`   | Clear all campaign state (works on paused too)                          |
| POST   | `/api/config/save`      | Deep-merge partial JSON into config.yaml, reload in-memory config       |

Config save accepts arbitrary partial JSON (e.g., `{"app":{"theme":"cupcake"}}` or `{"app":{"language":"id"}}`). Only provided keys updated; existing keys and order preserved via `yaml.Node` merge. `canonicalOrder` map enforces struct field order (app → server → email → worker → log, app.theme → app.language, email.provider → email.from → email.smtp → email.ses, etc.). Reloads `DefaultConfig` in-memory after save. Theme and language auto-persist on change.

Preview and Start use `multipart/form-data`. CSV sent as file (`csv_file`) or text field (`csv_text`). Other fields: `subject`, `body`, `to`, `from`, `provider`, `smtp_host`, `smtp_port`, `smtp_username`, `smtp_password`, `smtp_tls`, `smtp_batch_size`, `ses_region`, `ses_access_key_id`, `ses_secret_access_key`, `ses_use_template`, `ses_template_name`, `ses_batch_size`, `concurrency`, `max_retries`, `retry_backoff_base`, `retry_backoff_max`, `log_to_file`, `verbose`. Preview also accepts `count`.

## Naming Conventions

### Go (Backend)

| Concern       | Convention                                 | Example                             |
| ------------- | ------------------------------------------ | ----------------------------------- |
| Packages      | lowercase, single word                     | `worker`, `email`, `config`         |
| Files         | snake_case                                 | `email_sender.go`, `worker_pool.go` |
| Types         | PascalCase                                 | `Campaign`, `RecipientStatus`       |
| Functions     | PascalCase (exported), camelCase (private) | `SendEmail()`, `parseCSV()`         |
| Errors        | `Err` prefix                               | `ErrInvalidCSV`                     |
| HTTP handlers | `Handle` prefix (exported)                 | `HandlePreview`, `HandleProgress`   |
| Test files    | `*_test.go`                                | `worker_pool_test.go`               |

### Svelte 5 (Frontend)

| Concern       | Convention                                             | Notes                                     |
| ------------- | ------------------------------------------------------ | ----------------------------------------- |
| Components    | PascalCase                                             | `CampaignForm.svelte`, `InputData.svelte` |
| Runes         | `$state`, `$derived`, `$effect`, `$props`, `$bindable` | —                                         |
| Props         | camelCase                                              | `let { toField } = $props()`              |
| Events        | callback props                                         | `let { onreset } = $props()`              |
| Slots         | `{@render}` snippets                                   | —                                         |
| Input binding | `onblur` sync for `$bindable`                          | Local `_value` state, flush on blur       |

## Directory Structure

```
cmd/server/              # main entrypoint, embedded frontend
internal/
  config/                # YAML config loading, partial save (yaml.Node merge, canonical order)
  handler/               # 9 HTTP handlers (Go Fiber v3)
  worker/                # worker pool, queue, retry logic, RunPending for resume
  email/                 # SMTP + SES senders, template rendering, batched SMTP
  store/                 # in-memory recipient status store
  campaign/              # CSV parsing, campaign orchestration, file logging
frontend/
  src/
    lib/
      components/
        CampaignForm.svelte   # orchestrator: state, API calls, tabs, progress, confirm modal
        InputData.svelte      # file picker + manual toggle + multipart file emit
        EmailTemplate.svelte  # To, Subject, Body + body preview modal
        ProviderConfig.svelte # SMTP/SES config, Reset/Save-as-defaults buttons
        WorkerConfig.svelte   # concurrency, retries, backoff, Reset/Save-as-defaults buttons
        LogConfig.svelte      # log_to_file and verbose toggles, Reset/Save-as-defaults buttons
        PreviewModal.svelte   # iframe sandbox render, code tab, next/prev
        Navbar.svelte         # version badge + title + GitHub icon + language picker (globe) + theme picker (DaisyUI dropdowns, auto-persist)
        ConfirmModal.svelte   # reusable confirmation dialog (i18n-reactive defaults)
      i18n.svelte.js         # translations (en/ar/id), reactive $state store, RTL sync
    App.svelte             # layout shell: Navbar + CampaignForm
    main.js
    app.css               # TailwindCSS v4 + DaisyUI v5 (themes: all)
```

## Go Conventions

- `slog` for all logging: `LevelError`/`LevelInfo`/`LevelDebug`.
- Placeholders `{key}` rendered via `strings.NewReplacer`.
- Worker pool: context cancellation, exponential backoff retry. Each worker goroutine creates its own `EmailSender` via `SenderFactory`. SMTP senders batch messages using a single `DialAndSend` call per batch (configurable `batch_size`, default 50). SES template mode buffers entries and flushes via `SendBulkTemplatedEmail` (configurable `batch_size`, default 50, max 50). Flush on worker completion.
- Pause: cancels context between sends (in-flight email completes gracefully). Resume: `RunPending` processes only `pending` recipients.
- Unified logging: `Store.LogAndEvent(level, msg)` logs to both `slog` (console) and in-memory events (frontend). `CampaignLogger.Log(level, msg)` writes the same message to a JSON file per run.
- Campaign log file per run: `logs/campaign_<timestamp>.log` with simple JSON lines `{"time":"...","level":"...","msg":"..."}`. Controlled by `log.campaign.log_to_file` in config.
- Verbose logging: `log.campaign.verbose: true` enables debug-level per-email events in frontend (SSE) and campaign log file. Terminal output follows slog handler level (Info by default suppresses debug).
- CSV upload: large files (100MB+) sent via `multipart/form-data` (`csv_file` field) to `parseMultipart()` in handler. Manual input sent as `csv_text` text field. Avoids JSON body size limits.
- Config partial save: `SavePartial(path, partialJSON)` reads existing YAML → `yaml.Node`, deep-merges partial JSON (via `mapToNode` with `canonicalOrder`), marshals without document header. Sensitive fields detected by `sensitiveKeyPaths` → routed to OS keyring (`github.com/zalando/go-keyring`, service `raditzlawliet.mecs`) → `_provider` metadata written to YAML. `Load()` calls `loadSecretsFromKeyring()` to hydrate fields with `_provider.type == "keyring"`.
- Test with `testify` (`assert`/`require`), each subtest gets its own `assert.New(t)`.
- Slices/maps initialized explicitly (never nil).
- Integration tests use `//go:build integration` + Mailtrap Local.

## Svelte 5 Conventions

- **i18n**: Zero-dependency reactive module (`i18n.svelte.js`). `$state` store + `t(key)` function used in all components. `$effect.root` syncs `document.documentElement.dir` (RTL for `ar`) and `lang` attribute. Language persisted to `config.yaml` via `POST /api/config/save`. Adding a language: add translations block + entry in `LANGS` array. RTL languages: add code to `RTL_LANGS` Set.
- **Dropdown close**: Content wrapped in `{#if}` (not just DaisyUI CSS hide) to avoid hidden DOM stealing clicks. `composedPath()` for click-outside detection — survives Svelte DOM recycling.
- **Theme**: Explicit `document.documentElement.dataset.theme` set on change, not relying on CSS `:has(.theme-controller:checked)` which breaks when radio inputs leave DOM.
- **Local state + onblur**: Child components with `$bindable` props use local `$state` copy (`_value`) bound to inputs. Sync to parent only on `onblur` via `flush()`. Avoids INP delays from per-keystroke parent re-renders.
- **Multipart upload**: CSV file sent via `FormData` (no manual `Content-Type` — browser sets boundary). `csvFile` prop tracked in `InputData`, used by `CampaignForm.buildFormData()` for preview/start.
- **Form pattern**: `<fieldset class="fieldset">` wrapping `<label class="label" for="id">` + `<input id="..." class="input w-full">`.
- **Iframe sandbox**: Preview modes use sandboxed iframes with `srcdoc` to isolate email content.
- **Log panel**: Collapsable below progress, auto-scrolls when scrolled to bottom, shows timestamped events from backend.
- **SSE**: Single `EventSource` connected on mount via `GET /api/campaign/events`, never disconnected. Streams progress + log every 1s.
- **Session restore**: On page refresh, `GET /api/campaign/config` returns current campaign state (template, config, progress, events). Form restores to in-progress state if campaign is running/paused/completed.
- **Form lock**: Input data, template, provider, and worker config are disabled when campaign is running or paused. Reset button disabled only when `running` (enabled on paused/completed).
- **Save as defaults**: Each config tab has a "Save as defaults" button → `ConfirmModal` → `POST /api/config/save`. Theme picker auto-persists on change. All persist to `config.yaml`.
- **Icons**: Use `@lucide/svelte` for icons (Check, ChevronRight/Down/Up, X, ChevronLeft/Right, PauseIcon, PlayIcon, CircleXIcon).
- **Literal braces**: Use `{'{name}'}` syntax in template HTML to output literal `{name}`.

## Testing Rules

1. Every Go package MUST have unit tests.
2. Table-driven tests with `t.Run`.
3. `go test ./...` and `go vet ./...` must pass before done.
4. Frontend: handle loading, empty, error states visually.

## Makefile Targets

| Target        | Purpose                               |
| ------------- | ------------------------------------- |
| `make build`  | Build frontend + embed + Go binary    |
| `make dev`    | Production mode single binary         |
| `make dev-be` | Backend only (API + CORS for `:5173`) |
| `make dev-fe` | Vite dev server with HMR              |
| `make test`   | `go test ./...`                       |
| `make clean`  | Remove build artifacts                |
