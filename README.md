# Oura CLI

A command-line interface for the Oura Ring API. Displays sleep, activity, readiness, heart rate, HRV, stress, and workout data.

## Features

- OAuth2 authentication with automatic token refresh
- All sleep periods shown (main sleep + naps)
- Local timezone display
- Clean terminal output with emoji indicators
- Webhook subscription management (create/list/update/delete/renew)
- Tag / enhanced tag / session browsing (list + get by document_id)
- Shell completion scripts (bash/zsh/fish)

## Setup

### 1. Create an Oura Application

1. Go to [Oura Developer Portal](https://cloud.ouraring.com/oauth/applications)
2. Create a new application:
   - **App Name:** Whatever you want
   - **Redirect URI:** `http://localhost:8081/callback`
   - **Scopes:** Select all data types you want to access
3. Note your **Client ID** and **Client Secret**

### 2. Configure the CLI

Create the config file at `~/.config/oura/config.json`:

```json
{
  "client_id": "your-client-id-here",
  "client_secret": "your-client-secret-here"
}
```

### 3. Build

```bash
go build -o oura .
```

Optionally copy to your PATH:
```bash
cp oura ~/bin/oura
```

### 4. Authenticate

```bash
oura auth
```

This opens a browser for OAuth login. After authorizing, your token is saved to `~/.config/oura/token.json`.

## Usage

```bash
# Today's summary (readiness, sleep, activity, stress, HR)
oura today

# Output JSON to stdout (append --json or -j to any command)
oura sleep 2026-01-10 --json
oura all --json 2026-01-10
oura hrv 2026-01-10 --json

# All metrics for a specific date
oura all 2026-01-10

# Individual metrics
oura sleep [date]
oura activity [date]
oura readiness [date]
oura heartrate [date]
oura hrv [date]
oura stress [date]
oura workout [date]

# Back-compat alias (same as: all --json)
oura json [date]

# Re-authenticate
oura auth

# Webhook subscriptions (use your app credentials)
oura webhook list
oura webhook create --callback-url https://my-api.example/oura/webhook --verification-token 123 --event-type update --data-type sleep
oura webhook get <id>
oura webhook renew <id>
oura webhook delete <id>

# Personal info
oura personal-info

# Tags / enhanced tags / sessions
oura tag list --start-date 2026-01-01 --end-date 2026-01-31
oura tag get <document_id>

oura enhanced-tag list --start-date 2026-01-01 --end-date 2026-01-31
oura enhanced-tag get <document_id>

oura session list --start-date 2026-01-01 --end-date 2026-01-31
oura session get <document_id>

# Shell completion
oura completion bash
oura completion zsh
oura completion fish
```

Date format: `YYYY-MM-DD` (defaults to today if omitted)

## Shell Completion

The CLI can output completion scripts:

```bash
oura completion <bash|zsh|fish>
```

Common install locations:

```bash
# bash
oura completion bash > /etc/bash_completion.d/oura

# zsh
mkdir -p ~/.zsh/completions
oura completion zsh > ~/.zsh/completions/_oura

# fish
mkdir -p ~/.config/fish/completions
oura completion fish > ~/.config/fish/completions/oura.fish
```

## Example Output

```
╔══════════════════════════════════════╗
║      OURA METRICS - 2026-01-10       ║
╚══════════════════════════════════════╝

💪 Readiness - 2026-01-10
────────────────────────────────────────
Score:              86

🌙 Sleep - 2026-01-10
────────────────────────────────────────
Score:         84

🛏️  Main Sleep
Time:          10:07 PM → 5:21 AM
Total Sleep:   6h 3m
Deep Sleep:    52m

😴 Nap
Time:          3:01 PM → 4:14 PM
Total Sleep:   56m
Deep Sleep:    21m
```

## Files

| Path | Description |
|------|-------------|
| `~/.config/oura/config.json` | OAuth client credentials |
| `~/.config/oura/token.json` | Access/refresh tokens (auto-managed) |

## License

MIT
