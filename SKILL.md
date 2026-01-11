---
name: oura
description: Oura Ring CLI for sleep, activity, readiness, HRV, and workout data.
homepage: https://github.com/andrew-kurin/oura-cli
metadata: {"clawdbot":{"emoji":"💍","requires":{"bins":["oura"]}}}
---

# Oura Ring CLI

Query Oura Ring data from the command line. Requires OAuth2 setup with Oura developer account.

## Setup

1. Create app at [Oura Developer Portal](https://cloud.ouraring.com/oauth/applications)
2. Set redirect URI to `http://localhost:8081/callback`
3. Create config at `~/.config/oura/config.json`:
   ```json
   {
     "client_id": "your-client-id",
     "client_secret": "your-client-secret"
   }
   ```
4. Run `oura auth` to authenticate

## Commands

```bash
# Today's summary (all metrics)
oura today

# All metrics for specific date
oura all 2026-01-10

# Individual metrics
oura sleep [date]      # Main sleep + naps with details
oura readiness [date]  # Readiness score and contributors
oura activity [date]   # Steps, calories, activity breakdown
oura heartrate [date]  # HR min/max/average
oura hrv [date]        # Heart rate variability
oura stress [date]     # Daytime stress levels
oura workouts [date]   # Detected workouts
```

Date format: `YYYY-MM-DD` (defaults to today)

## Example Usage

**Morning check-in:**
```bash
oura today
```

**Check last night's sleep:**
```bash
oura sleep
```

**Review a specific day:**
```bash
oura all 2026-01-09
```

## Output

Shows all sleep periods (main sleep + naps) with:
- Time (local timezone)
- Duration and efficiency
- Sleep stages (deep, light, REM)
- HR, HRV, breath rate

Readiness includes contributor scores (resting HR, HRV, recovery, etc.)

## Files

| Path | Description |
|------|-------------|
| `~/.config/oura/config.json` | OAuth credentials |
| `~/.config/oura/token.json` | Access tokens (auto-managed) |
