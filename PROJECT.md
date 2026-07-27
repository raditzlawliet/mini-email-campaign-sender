# Mini Email Campaign Sender

## Overview

System to send a single promotional email campaign to **1,000,000 recipients**, where each email shares the same template body but is personalized with the recipient's name with easy configuration.

## Core Requirements & Features

- Simple Email Campaign Page for that has a form for inputting campaign details, configure Email Provider and worker, and tracking progress.
  - Support data input CSV
  - Support template input with personalization placeholders by data header (e.g. `{name}`, `{email}`)
  - Real-time progress tracking and status updates using SSE/Event
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
- Simple File Configuration for Application Settings such as web server port.
- UI default features
  - Support theme picker from daisy UI default theme collection, default is `dark`
  - Support multilingual and rtl display: en (default), id, ar
  - Has App Version and Github link

## Tech Stack

- Frontend: Svelte 5 latest, TailwindCSS 4 latest, DaisyUI 5 latest (for UI components), Embed in Go binary, `lucide` for icons.
- Backend: Golang (Go Fiber v3 or latest for web serve static Frontend, `https://github.com/wneessen/go-mail` for email sending, `slog` for logging)
- Email Provider: Email (SMTP), Amazon SES
- Build Tool: Makefile

## Development

- Use Mailpit Local for email testing (https://github.com/axllent/mailpit)
- Include Unit Tests for backend logic and integration tests
- Include build scripts for building the application and running tests using Makefile
- Include Readme with development instruction and build instructions
- Frontend development can be done separately (e.g., hot reloading) without needing to rebuild the frontend and embedding it in the Go binary.
- Always use recommended tools when development. Do not add or edit dependencies from lock or sum file or auto-generated files.

## Project scoped

- No need to support multiple campaigns at the same time.
- No need to persist campaign state across runs or stored in a database.
- One page web interface show up the form to sending campaign.
- Reset state every time the application is run.

## Notes

- System support sending 1M emails.
- SPF, DKIM, DMARC must be configured on the sending domain to avoid landing in spam.
- Final build can be run on Windows or Linux, User can run the application with minimum configuration and web server up to the point where the server is running and the web page is accessible.
- CSV data can be big size (e.g., 100MB+), passing from the frontend to the backend should be processed gracefully using HTTP multipart form data.
