---
name: Google Workspace
tags: [google, workspace, drive, gmail, calendar, sheets, productivity]
requires:
  bins: [gws]
---

# Google Workspace

Interact with Google Drive, Gmail, Calendar, Sheets, Docs, and more using the `gws` CLI.

## Installation

```bash
npm install -g @googleworkspace/cli
```

Requires Node.js 18+.

## Authentication

```bash
gws auth setup
```

This opens a browser for Google OAuth. Credentials are encrypted at rest using AES-256-GCM.
Alternatively, set `GOOGLE_WORKSPACE_CLI_TOKEN` with a pre-obtained access token.

## Capabilities

The `gws` CLI dynamically generates commands from Google's Discovery Service, covering:
- **Drive**: files, folders, uploads, downloads, sharing, search
- **Gmail**: send, read, search, label, thread management
- **Calendar**: events, calendars, attendees, reminders
- **Sheets**: spreadsheets, values, formulas, formatting
- **Docs**: documents, content, revisions
- **Chat**: spaces, messages, memberships
- **Admin**: users, groups, policies (requires Workspace admin)

## How to Use

Use the `exec` tool to run `gws` commands.

### Explore available commands
```bash
gws --help
gws drive --help
gws gmail --help
gws calendar --help
```

### Google Drive
```bash
# List files
gws drive files list --q "name contains 'report'"

# Upload a file
gws drive files create --upload-file /path/to/file.pdf --name "Report.pdf"

# Download a file
gws drive files get --file-id <ID> --alt media --output report.pdf
```

### Gmail
```bash
# Send an email
gws gmail users messages send --to recipient@example.com \
  --subject "Hello" --body "Message content"

# Search emails
gws gmail users messages list --q "from:boss@company.com is:unread"

# Read an email
gws gmail users messages get --id <MESSAGE_ID> --format full
```

### Google Calendar
```bash
# List upcoming events
gws calendar events list --calendar-id primary --time-min $(date -I)T00:00:00Z

# Create an event
gws calendar events insert --calendar-id primary \
  --summary "Meeting" --start "2026-03-10T10:00:00Z" --end "2026-03-10T11:00:00Z"
```

### Google Sheets
```bash
# Read cell values
gws sheets spreadsheets values get --spreadsheet-id <ID> --range "Sheet1!A1:D10"

# Write values
gws sheets spreadsheets values update --spreadsheet-id <ID> \
  --range "Sheet1!A1" --value-input-option USER_ENTERED --values '[[1,2,3]]'
```

## Advanced Features

```bash
# Auto-paginate results
gws drive files list --page-all

# Dry run (preview without executing)
gws gmail users messages send --dry-run ...

# Output as JSON for scripting
gws drive files list --format json
```

## Notes

- First run: `gws auth setup` to authenticate
- Credentials stored encrypted in `~/.config/gws/`
- Scope of access depends on scopes granted during OAuth
- For service accounts, set `GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE` to your JSON key path
- `gws schema <method>` shows the full schema for any API method
