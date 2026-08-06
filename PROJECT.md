# Mini Email Campaign Sender

## Overview

System to send a single promotional email campaign to **1,000,000 recipients**, where each email shares the same template body but is personalized with the recipient's name with easy configuration.

## Core Requirements & Features

- Simple Email Campaign Page for that has a form for inputting campaign details, configure Email Provider and worker, and tracking progress.
  - Support data input CSV
  - Support template input with personalization placeholders by data header (e.g. `{name}`, `{email}`)
  - Real-time progress tracking and status updates using Wails Events
  - Support overwrite select and configure Email Provider configuration that already configure.
  - Support overwrite configure worker that already configure.
  - Dry-run / preview mode to test email content without sending.
  - Reset button to clear the campaign and start over.
  - Pause/Resume button to pause and resume the campaign.
- Queue-based processing to handle the volume of emails efficiently that can be resumed.
  - Worker pool or async I/O to maximize throughput.
  - In-memory store to mark per-recipient status (pending, sent, failed) for final delivery tracking report.
  - Retry strategy for transient failures with maximum retry attempts and exponential backoff.
  - Campaign log including the form, configurationm, and delivery tracking report stored in the file logging to disk each fresh campaign run. Attempt retrying failed emails appending to the log each retry attempt
- Simple File Configuration for Application Settings.
  - Priority: CONFIG_PATH env -> ./config.yaml (if exists) -> ~/.mecs/config.yaml (default)
  - Use keyring to store sensitive data such as password, api key or similars.
- UI default features
  - Support theme picker from daisy UI default theme collection, default is `dark`
  - Support multilingual and rtl display: en (default), id, ar
  - Has App Version and Github link

## Tech Stack

- Frontend: Svelte 5 latest, TailwindCSS 4 latest, DaisyUI 5 latest (for UI components), rendered in Wails WebView, `lucide` for icons.
- Backend: Golang (Wails v2.13.0 for desktop app, `https://github.com/wneessen/go-mail` for email sending, `slog` for logging)
- Email Provider: Email (SMTP), Amazon SES
- Build Tool: Makefile
- Desktop: Wails v2.13.0 (Windows WebView2 / Linux WebKitGTK)

## Development

- Use Mailpit Local for email testing (https://github.com/axllent/mailpit)
- Include Unit Tests for backend logic and integration tests
- Include build scripts for building the application and running tests using Makefile
- Include Readme with development instruction and build instructions
- Desktop development with `wails dev` (Go + Vite hot-reload), no HTTP server or CORS needed
- Always use recommended tools when development. Do not add or edit dependencies from lock or sum file or auto-generated files.

## Project scoped

- No need to support multiple campaigns at the same time.
- No need to persist campaign state across runs or stored in a database.
- One page desktop UI show up the form to sending campaign.
- Reset state every time the application is run.

## Notes

- System support sending 1M emails.
- SPF, DKIM, DMARC must be configured on the sending domain to avoid landing in spam.
- Final build can be run on Windows or Linux (Wails cross-platform build, see Makefile targets).
- CSV data can be big size (e.g., 100MB+): native file dialog returns a path, Go reads the file directly (`os.ReadFile`) avoiding JS memory pressure.
