<div align="center">

# Kairu

**A keyboard-first Pomodoro timer for the terminal.**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-22c55e?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-6366f1?style=flat-square)]()
[![Storage](https://img.shields.io/badge/Storage-Local%20Only-f59e0b?style=flat-square)]()
[![Built With](https://img.shields.io/badge/Built%20with-Bubbletea-ec4899?style=flat-square)](https://github.com/charmbracelet/bubbletea)

</div>

---

Kairu is a compact, local-first TUI application that combines a Pomodoro timer, task capture, session templates, ambient soundscapes, and productivity analytics - all without leaving your terminal.

> No accounts. No cloud. No distractions. Just work.

---

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Usage](#usage)
- [Keybindings](#keybindings)
- [Soundscapes](#soundscapes)
- [Template Management](#template-management)
- [Configuration](#configuration)
- [Notifications](#notifications)
- [Data & Storage](#data--storage)
- [CLI Reference](#cli-reference)
- [Contributing](#contributing)
- [Documentation](#documentation)

---

## Features

| Category | Capabilities |
|---|---|
| **Timer** | ASCII art countdown, pause/resume, inline duration edit, auto-break cycling |
| **Task Capture** | Task name, note, comma-separated tags, recent task recall overlay |
| **Templates** | Save, apply, rename, delete, duplicate, and undo — full browser with `Ctrl+P` |
| **Soundscapes** | Ambient audio during work sessions, live track indicator in timer header |
| **Analytics** | Daily totals, streak tracking, recovery mode, work/break ratio, weekly chart |
| **Heatmap** | Year-at-a-glance activity heatmap (GitHub-style) |
| **Reports** | Session timeline, daily markdown report with export |
| **Notifications** | Desktop, sound command, and Telegram — with quiet hours and an outbox |
| **Personalization** | 4 themes (`forest`, `ocean`, `ember`, `mono`), 3 timer fonts, in-app settings |
| **Data Safety** | One-command backup and restore; all data is local JSON/YAML |

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      kairu-tui                          │
│                                                         │
│  ┌──────────────┐    ┌───────────────────────────────┐  │
│  │   Config     │    │          TUI (Bubbletea)       │  │
│  │  kairu.yaml  │───▶│                               │  │
│  └──────────────┘    │  Input ──▶ Timer ──▶ Break    │  │
│                      │    │         │                 │  │
│  ┌──────────────┐    │    ▼         ▼                 │  │
│  │   Session    │◀──▶│  Templates  Stats Dashboard   │  │
│  │ entries.json │    │             │                 │  │
│  └──────────────┘    │             ▼                 │  │
│                      │           Heatmap             │  │
│  ┌──────────────┐    │           Timeline            │  │
│  │  Templates   │◀──▶│           Report              │  │
│  │templates.json│    └───────────────────────────────┘  │
│  └──────────────┘                  │                    │
│                                    ▼                    │
│  ┌─────────────────────────────────────────────────┐   │
│  │             Notification Pipeline               │   │
│  │  outbox.json ──▶ Desktop / Sound / Telegram     │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

**Key design decisions:**

- **Single binary** - the entire application lives in one Go file with zero CGO dependencies.
- **Local-first** - all state is stored in plain JSON/YAML files next to the binary. No network calls except optional Telegram delivery.
- **Keyboard-driven** - every action is reachable from the keyboard; the mouse is never required.
- **Bubbletea model** - state flows through a single immutable `model` struct updated by an `Update` function, making the app deterministic and easy to reason about.

---

## Quick Start

**Requirements:** Go 1.21+

```bash
# Clone and run
git clone https://github.com/sudo-Harshk/kairu-tui.git
cd kairu-tui
go run main.go
```

```bash
# Or install to $GOPATH/bin
go install .
```

The app creates `kairu.yaml`, `entries.json`, and `templates.json` on first run. No setup required.

---

## Usage

### Starting a session

1. Type a task name in the **Task** field
2. Enter a duration - `25` (minutes) or `1:30` (hh:mm)
3. Optionally fill in a note and tags (comma-separated)
4. Press `Enter` to start the timer

### During a session

- `Space` - pause or resume
- `E` - edit the planned duration without stopping
- `Enter` - end the session early and save it
- `Tab` - open the analytics dashboard
- `Ctrl+M` - open the soundscape selector

### Reviewing your work

Press `Tab` while the timer is running to step through the analytics pipeline:

```
Timer → Dashboard → Analytics → Heatmap → Timeline → Report → Timer
```

The **Daily Report** can be exported as a Markdown file with `E`.

### Streak & Recovery

Kairu tracks consecutive work days as a **streak**, shown in the timer header:

| Indicator | Meaning |
|---|---|
| `🔥 N` | Active streak — worked N days in a row including today |
| `✦ recoverable` | Missed yesterday, but the streak can still be saved today |
| `◌ rebuild` | Streak lost — start fresh today |

When recovery is available, the input screen shows a prompt so you never silently lose progress.

---

## Keybindings

### Input Screen

| Key | Action |
|---|---|
| `Tab` | Cycle between fields |
| `Enter` | Advance field / start session |
| `Up` / `Down` | Browse recent tasks (shows 5-item overlay) |
| `Left` / `Right` | Cycle templates (when Template field is focused) |
| `Ctrl+P` | Open Template Browser |
| `Ctrl+T` | Save current form as a template |
| `Ctrl+M` | Open Soundscape selector |
| `?` | Help overlay |
| `q` | Quit |

### Timer & Break

| Key | Action |
|---|---|
| `Space` | Pause / Resume |
| `E` | Edit session duration |
| `Enter` | End session early |
| `Tab` | Open analytics dashboard |
| `S` | Open settings |
| `Ctrl+M` | Open Soundscape selector |
| `?` | Help overlay |
| `q` | Quit (saves current session) |

### Template Browser (`Ctrl+P`)

| Key | Action |
|---|---|
| `Tab` / `Up` / `Down` | Navigate templates |
| `Enter` | Apply selected template |
| `Ctrl+T` | Save current form as new template |
| `Ctrl+R` | Rename selected template |
| `Ctrl+D` | Delete selected template |
| `Ctrl+Z` | Undo last delete (10 s window) |
| `Ctrl+Y` | Duplicate selected template |
| `Ctrl+P` / `Esc` | Return to input screen |

### Analytics Views

| Key | Action |
|---|---|
| `Tab` | Advance to next view |
| `Esc` | Return to timer |
| `E` (Report only) | Export markdown report |
| `S` | Open settings |

---

## Soundscapes

Kairu can play looping ambient audio during work sessions.

### Setup

1. Create a `soundscapes/` directory in the project root (or set `soundscapes_dir` in config)
2. Drop audio files into it - `.mp3`, `.wav`, `.ogg`, `.flac`, `.aac`
3. Install a command-line player:

```bash
# macOS
brew install mpv

# Ubuntu / Debian
sudo apt install mpv

# Windows (winget)
winget install mpv
```

### How it works

- Press `Ctrl+M` to open the selector at any time
- Navigate with `Up` / `Down`, confirm with `Enter`
- Select **None** to stop playback
- Playback is scoped to work sessions only - it pauses on break and stops on session end
- The active track name appears in the timer header as `🎵 Track Name`

You can use any player that accepts a file path as its final argument:

```yaml
soundscape_player: ffplay -loop 0 -nodisp -autoexit
```

---

## Template Management

Templates store a full session configuration - task, duration, note, and tags - for one-keystroke reuse.

### Creating a template

Fill in the input form and press `Ctrl+T`. If a template with the same task name already exists it is updated in place.

### Applying a template

- Press `Ctrl+P` to open the Template Browser and select from a searchable list
- Or use `Left` / `Right` while the **Template** field is focused to cycle inline

### Template Browser

Each entry shows **Name**, **Duration**, and **Tags**. Use the browser to apply, rename, delete, undo, or duplicate templates without leaving the keyboard.

---

## Configuration

Create `kairu.yaml` in the project root. All keys are optional and fall back to the defaults shown below.

```yaml
# Session durations (minutes)
work_duration: 25
break_duration: 5

# Appearance
theme: forest         # forest | ocean | ember | mono
font: ansi            # ansi | block | thin

# Auto-break cycling
auto_break: false
sessions_before_break: 4

# Notifications
notifications: false
desktop_notifications: true
notify_work_complete: true
notify_break_complete: true
notify_session_start: false
notify_session_end: false
notify_pause_resume: false
notify_ending_soon: false

# Quiet hours (-1 disables)
quiet_hours_start: -1
quiet_hours_end: -1

# Sound hook (runs after desktop notification fails)
sound_command: ""

# Soundscapes
soundscapes_dir: soundscapes
soundscape_player: mpv --loop --no-video
```

### Themes

| Value | Description |
|---|---|
| `forest` | Green accent, natural tones |
| `ocean` | Blue accent, cool palette |
| `ember` | Amber accent, warm palette |
| `mono` | No color, terminal default |

### Telegram Notifications

Add credentials to a `.env` file (never committed):

```bash
KAIRU_TELEGRAM_BOT_TOKEN=your_bot_token
KAIRU_TELEGRAM_CHAT_ID=your_chat_id
```

See [docs/telegram-notifications.md](docs/telegram-notifications.md) for setup instructions.

---

## Notifications

Kairu attempts notification delivery in order: **Desktop → Sound command → Telegram**. The first successful channel wins. Failed attempts are queued in `notification_outbox.json` and retried automatically.

**Events that can trigger a notification:**

- Session start / end
- Work complete
- Break complete
- Pause / resume
- Ending soon (configurable threshold)

Quiet hours suppress all notifications between `quiet_hours_start` and `quiet_hours_end` (24-hour integers, e.g. `22` and `7`).

---

## Data & Storage

All data lives next to the binary. No installation to system directories.

| File | Contents |
|---|---|
| `kairu.yaml` | Application configuration |
| `entries.json` | Session history |
| `templates.json` | Saved session templates |
| `notification_outbox.json` | Pending notification retries |
| `backup.json` | Full state snapshot (created on demand) |

### Session schema

```json
{
  "task": "Write design doc",
  "note": "Focus on the API section",
  "tags": ["writing", "deep work"],
  "start": "2026-05-17T09:00:00Z",
  "end": "2026-05-17T09:25:00Z",
  "duration": 1500,
  "type": "work"
}
```

---

## CLI Reference

```bash
# Backup full state (config + sessions + templates + outbox)
go run main.go --backup backup.json

# Restore from backup
go run main.go --restore backup.json

# Export session history only
go run main.go --export entries.json

# Import and merge session history
go run main.go --import entries.json
```

`--backup`, `--restore`, `--export`, and `--import` are mutually exclusive.

In-app backup and restore are also available from the **Settings** screen.

---

## Contributing

1. Open an issue before starting larger changes - alignment up front saves everyone time.
2. Keep pull requests focused and small.
3. Run `go test ./...` before submitting.
4. All UI changes must remain keyboard-first and compatible with `charmbracelet/bubbletea`.
5. No new CGO or cloud dependencies — keep the binary self-contained and lightweight.

---

## Documentation

| Document | Description |
|---|---|
| [docs/setup.md](docs/setup.md) | Detailed installation and environment setup |
| [docs/usage.md](docs/usage.md) | Full usage guide including streaks and soundscapes |
| [docs/configuration.md](docs/configuration.md) | Every configuration key explained |
| [docs/telegram-notifications.md](docs/telegram-notifications.md) | Telegram bot setup walkthrough |

---

<div align="center">

MIT License · Built with [Bubbletea](https://github.com/charmbracelet/bubbletea) · No telemetry, no cloud

</div>
