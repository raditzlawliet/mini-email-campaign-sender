# AGENTS.md — Mini Email Campaign Sender (Wails Desktop)

> AI agent configuration. Update as the project evolves. Keep it short.
> You, AI, should update this AGENTS.md as much as needed as the project evolve.

## Project

Desktop app (Wails v2) to send personalized email campaigns to 1M recipients per campaign via SMTP/SES, with queue-based processing, retries, and real-time progress.

- **Frontend**: Svelte 5 (runes), TailwindCSS v4, DaisyUI v5, rendered in Wails WebView.
- **Backend**: Go, Wails v2.13.0, `wneessen/go-mail` for email sending, `slog` for logging.
- **Email Testing**: Mailtrap Local (`mailtrap-local`)
- **Config**: YAML at `~/.mecs/config.yaml` (default) or `CONFIG_PATH` env or `./config.yaml` (legacy). Sensitive fields (SMTP password, SES keys) stored in OS keyring with `_provider` metadata blocks. Partial save via `SaveConfig` binding (yaml.Node merge).
- **Editor**: Zed with Svelte extension.
- **Dev mode**: `wails dev` (hot-reload for Go + Vite). No HTTP server, no CORS.

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

## Wails Bindings (App struct)

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

All methods live on `App` in `internal/app/app.go`, bound via `Bind: []interface{}{application}` in root `main.go`. Frontend imports from `frontend/src/lib/wailsjs/go/app/App` (generated by `wails generate module`, gitignored). Progress + log events stream via Wails Events: Go emits `campaign:progress` every 1s (`runtime.EventsEmit`), frontend listens with `EventsOn("campaign:progress", cb)`.

`CampaignInput` struct fields serialize as PascalCase (no json tags) — the frontend `buildInput()` must use exact keys: `CSVText`, `CSVFilePath`, `Subject`, `Body`, `To`, `From`, `Provider`, `SMTPHost`, `SMTPPort`, `SMTPUsername`, `SMTPPassword`, `SMTPTLS`, `SMTPBatchSize`, `SESRegion`, `SESAccessKeyID`, `SESSecretKey`, `SESUseTemplate`, `SESTemplateName`, `SESBatchSize`, `Concurrency`, `MaxRetries`, `BackoffBase`, `BackoffMax`, `LogToFile`, `Verbose`, `Count`.

Config save accepts arbitrary partial JSON (e.g., `{"app":{"theme":"cupcake"}}` or `{"app":{"language":"id"}}`). Only provided keys updated; existing keys and order preserved via `yaml.Node` merge. `canonicalOrder` map enforces struct field order (app → server → email → worker → log, app.theme → app.language, email.provider → email.from → email.smtp → email.ses, etc.). Reloads `DefaultConfig` in-memory after save. Theme and language auto-persist on change.

CSV: native file dialog via `PickCSVFile` (path string), Go reads the file with `os.ReadFile` (handles 100MB+ files without JS memory pressure). Manual input sent as `csvText` in `CampaignInput`.

## Naming Conventions

### Go (Backend)

| Concern       | Convention                                 | Example                             |
| ------------- | ------------------------------------------ | ----------------------------------- |
| Packages      | lowercase, single word                     | `worker`, `email`, `config`         |
| Files         | snake_case                                 | `email_sender.go`, `worker_pool.go` |
| Types         | PascalCase                                 | `Campaign`, `RecipientStatus`       |
| Functions     | PascalCase (exported), camelCase (private) | `SendEmail()`, `parseCSV()`         |
| Errors        | `Err` prefix                               | `ErrInvalidCSV`                     |
| Wails methods | PascalCase, exported on App struct         | `StartCampaign`, `GetVersion`       |
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
main.go                    # Wails entry point (wails.Run, asset server, bindings)
wails.json                 # Wails project config (wailsjsdir -> frontend/src/lib)
internal/
  app/                     # Wails App struct + bound methods (replaces old HTTP handlers)
  config/                  # YAML config loading, partial save (yaml.Node merge, canonical order)
  worker/                  # worker pool, queue, retry logic, RunPending for resume
  email/                   # SMTP + SES senders, template rendering, batched SMTP
  store/                   # in-memory recipient status store
  campaign/                # CSV parsing, campaign orchestration, file logging
frontend/
  src/
    lib/
      components/
        CampaignForm.svelte   # orchestrator: state, Wails binding calls, tabs, progress, confirm modal
        InputData.svelte      # native file dialog (PickCSVFile) + manual toggle
        EmailTemplate.svelte  # To, Subject, Body + body preview modal
        ProviderConfig.svelte # SMTP/SES config, Reset/Save-as-defaults buttons
        WorkerConfig.svelte   # concurrency, retries, backoff, Reset/Save-as-defaults buttons
        LogConfig.svelte      # log_to_file and verbose toggles, Reset/Save-as-defaults buttons
        PreviewModal.svelte   # iframe sandbox render, code tab, next/prev
        Navbar.svelte         # version badge + title + GitHub icon (BrowserOpenURL) + language picker (globe) + theme picker (DaisyUI dropdowns, auto-persist)
        ConfirmModal.svelte   # reusable confirmation dialog (i18n-reactive defaults)
      wailsjs/               # AUTO-GENERATED by wails generate module (gitignored)
      i18n.svelte.js         # translations (en/ar/id), reactive $state store, RTL sync
    App.svelte             # layout shell: Navbar + CampaignForm
    main.js
    app.css               # TailwindCSS v4 + DaisyUI v5 (themes: all)
build/                     # Wails build output (gitignored)
```

## Go Conventions

- `slog` for all logging: `LevelError`/`LevelInfo`/`LevelDebug`.
- Placeholders `{key}` rendered via `strings.NewReplacer`.
- Worker pool: context cancellation, exponential backoff retry. Each worker goroutine creates its own `EmailSender` via `SenderFactory`. SMTP senders batch messages using a single `DialAndSend` call per batch (configurable `batch_size`, default 50). SES template mode buffers entries and flushes via `SendBulkTemplatedEmail` (configurable `batch_size`, default 50, max 50). Flush on worker completion.
- Pause: cancels context between sends (in-flight email completes gracefully). Resume: `RunPending` processes only `pending` recipients.
- Unified logging: `Store.LogAndEvent(level, msg)` logs to both `slog` (console) and in-memory events (frontend). `CampaignLogger.Log(level, msg)` writes the same message to a JSON file per run.
- Campaign log file per run: `logs/campaign_<timestamp>.log` with simple JSON lines `{"time":"...","level":"...","msg":"..."}`. Controlled by `log.campaign.log_to_file` in config.
- Verbose logging: `log.campaign.verbose: true` enables debug-level per-email events in frontend (Wails Events) and campaign log file. Terminal output follows slog handler level (Info by default suppresses debug).
- CSV: file dialog returns path (no file content through JS). Go reads file with `os.ReadFile`. Manual input passed as `csvText` string.
- Config partial save: `SavePartial(path, partialJSON)` reads existing YAML → `yaml.Node`, deep-merges partial JSON (via `mapToNode` with `canonicalOrder`), marshals without document header. Sensitive fields detected by `sensitiveKeyPaths` → routed to OS keyring (`github.com/zalando/go-keyring`, service `raditzlawliet.mecs`) → `_provider` metadata written to YAML. `Load()` calls `loadSecretsFromKeyring()` to hydrate fields with `_provider.type == "keyring"`.
- Wails: App struct stores `ctx` from `OnStartup`. Bound methods return `error` for failures (JS promise rejects). External links open via `runtime.BrowserOpenURL`.
- Test with `testify` (`assert`/`require`), each subtest gets its own `assert.New(t)`.
- Slices/maps initialized explicitly (never nil).
- Integration tests use `//go:build integration` + Mailtrap Local.

## Svelte 5 Conventions

- **i18n**: Zero-dependency reactive module (`i18n.svelte.js`). `$state` store + `t(key)` function used in all components. `$effect.root` syncs `document.documentElement.dir` (RTL for `ar`) and `lang` attribute. Language persisted to `config.yaml` via `SaveConfig` binding. Adding a language: add translations block + entry in `LANGS` array. RTL languages: add code to `RTL_LANGS` Set.
- **Dropdown close**: Content wrapped in `{#if}` (not just DaisyUI CSS hide) to avoid hidden DOM stealing clicks. `composedPath()` for click-outside detection — survives Svelte DOM recycling.
- **Theme**: Explicit `document.documentElement.dataset.theme` set on change, not relying on CSS `:has(.theme-controller:checked)` which breaks when radio inputs leave DOM.
- **Local state + onblur**: Child components with `$bindable` props use local `$state` copy (`_value`) bound to inputs. Sync to parent only on `onblur` via `flush()`. Avoids INP delays from per-keystroke parent re-renders.
- **Wails bindings**: Import from `../wailsjs/go/app/App` (component dir) or `../wailsjs/runtime/runtime` for events/`BrowserOpenURL`. Errors reject as JS promises; strip `Error: ` prefix via `bindingError(e)` helper in CampaignForm.
- **Form pattern**: `<fieldset class="fieldset">` wrapping `<label class="label" for="id">` + `<input id="..." class="input w-full">`.
- **Iframe sandbox**: Preview modes use sandboxed iframes with `srcdoc` to isolate email content.
- **Log panel**: Collapsable below progress, auto-scrolls when scrolled to bottom, shows timestamped events from backend.
- **Events**: Single `EventsOn("campaign:progress", cb)` registered on mount via `$effect`, never disconnected. Streams progress + log every 1s.
- **Session restore**: On app start, `GetCampaignConfig()` returns current campaign state (template, config, progress, events). Form restores to in-progress state if campaign is running/paused/completed.
- **Form lock**: Input data, template, provider, and worker config are disabled when campaign is running or paused. Reset button disabled only when `running` (enabled on paused/completed).
- **Save as defaults**: Each config tab has a "Save as defaults" button → `ConfirmModal` → `SaveConfig(JSON.stringify(payload))`. Theme picker auto-persists on change. All persist to `config.yaml`.
- **Icons**: Use `@lucide/svelte` for icons (Check, ChevronRight/Down/Up, X, ChevronLeft/Right, PauseIcon, PlayIcon, CircleXIcon).
- **Literal braces**: Use `{'{name}'}` syntax in template HTML to output literal `{name}`.

## Testing Rules

1. Every Go package MUST have unit tests.
2. Table-driven tests with `t.Run`.
3. `go test ./...` and `go vet ./...` must pass before done.
4. Frontend: handle loading, empty, error states visually.

## Makefile Targets

| Target          | Purpose                              |
| --------------- | ------------------------------------ |
| `make build`    | Current-OS Wails build               |
| `make build-windows` | Windows amd64 + NSIS installer  |
| `make build-linux`   | Linux amd64 binary              |
| `make build-all`     | Both platforms                   |
| `make dev`      | `wails dev` (Go + Vite hot-reload)   |
| `make test`     | `go test ./...`                       |
| `make clean`    | Remove build artifacts                |
