# Kairu TUI

Kairu TUI is a local-first, keyboard-driven Pomodoro timer for people who want focused work sessions, lightweight task capture, and useful activity analytics without leaving the terminal.

It combines a visual ASCII timer, reusable templates, notes and tags, session history, daily reporting, and notification support in a compact Go TUI.

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-CLI%20%2F%20TUI-111827)]()
[![Local First](https://img.shields.io/badge/Storage-Local%20Only-16a34a)]()

## Why Kairu

Most timer apps focus on counting down. Kairu focuses on helping you work.

It is designed for people who want:

- a distraction-free terminal workflow
- quick capture of tasks, notes, and tags
- reusable session templates for repeat work
- useful history instead of a pile of raw timestamps
- lightweight reporting without leaving the app
- local control over data, notifications, and formatting

## Highlights

- Local-first session tracking
- Fast keyboard workflow
- Pomodoro-style work and break cycles
- Reusable session templates
- Notes, tags, and recent task recall
- Activity dashboard with streaks and charts
- Analytics snapshot with task, tag, and session breakdowns
- Session timeline grouped by day
- Daily markdown report export
- Desktop, sound, and Telegram notifications
- Theme and timer font customization

## Screens and Features

### Focus Workflow

- Start a task with a duration, optional note, and optional tags
- Use recent task recall to reuse past entries quickly
- Save the current form as a template for repeated workflows
- Browse, apply, rename, delete, or duplicate templates

### Timer and Break Modes

- Large ASCII art timer display
- Pause and resume sessions
- Edit the running session duration
- End a session early
- Automatic break switching after a configurable number of work sessions

### Analytics and Review

- Today's work total
- Current and best streaks
- Recovery status
- Work/break ratio
- Weekly activity chart
- Streak history chart
- Session analytics snapshot with top tasks, tags, and busiest day
- Top tags summary
- Session timeline grouped by day
- Daily report view with export

### Notifications

- Session start/end notifications
- Work complete and break complete notifications
- Pause/resume notifications
- Ending soon reminders
- Quiet hours support
- Optional sound command hook
- Optional Telegram delivery

### Personalization

- Theme selection: `forest`, `ocean`, `ember`, `mono`
- Timer fonts: `ansi`, `block`, `thin`
- Notification toggles and quiet hours in-app

## Keybindings

| Screen | Keys | Action |
| --- | --- | --- |
| Input | `Tab` | Move between fields |
| Input | `Enter` | Advance or start the session |
| Input | `Ctrl+P` | Open template manager |
| Input | `Ctrl+T` | Save current form as template |
| Input | `Ctrl+R` | Rename selected template |
| Input | `Ctrl+D` | Delete selected template |
| Input | `Ctrl+Y` | Duplicate selected template |
| Input | `Up` / `Down` | Cycle recent tasks |
| Timer / Break | `Space` | Pause or resume |
| Timer / Break | `E` | Edit total duration |
| Timer / Break | `Enter` | End session early |
| Timer / Break | `Tab` | Open activity dashboard |
| Dashboard | `Tab` | Open analytics snapshot |
| Analytics | `Tab` | Open session timeline |
| Timeline | `Tab` | Open daily report |
| Report | `E` | Export markdown report |
| Any main screen | `S` | Open settings |
| Any main screen | `?` | Open help overlay |
| Any main screen | `q` | Quit |

## Quick Start

Requirements:

- Go 1.21+

Run from source:

```bash
git clone https://github.com/sudo-Harshk/kairu-tui.git
cd kairu-tui
go run main.go
```

Optional install:

```bash
go install .
```

## Usage

### 1. Create a session

From the input screen:

- Enter a task name
- Enter a duration in minutes, such as `25`, or in `hh:mm`, such as `1:00`
- Optionally add a note
- Optionally add comma-separated tags such as `deep work, writing`
- Press `Enter` to start

### 2. Manage the timer

While a session is active:

- Pause or resume with `Space`
- Edit the planned duration with `E`
- End early with `Enter`
- Open the dashboard with `Tab`

### 3. Review your work

From the dashboard, press `Tab` to move through:

- Activity dashboard
- Analytics snapshot
- Session timeline
- Daily report

The daily report can be exported as a markdown file for external notes or journaling.

## Data Files

Kairu stores everything locally in the project directory:

- `entries.json` - session history
- `templates.json` - saved templates
- `kairu.yaml` - configuration
- `notification_outbox.json` - queued notification retries

## Configuration

Create `kairu.yaml` in the project root. Missing values fall back to defaults.

```yaml
work_duration: 25
break_duration: 5
theme: forest
font: ansi
notifications: false
desktop_notifications: true
notify_work_complete: true
notify_break_complete: true
notify_session_start: false
notify_session_end: false
notify_pause_resume: false
notify_ending_soon: false
quiet_hours_start: -1
quiet_hours_end: -1
sound_command: ""
auto_break: false
sessions_before_break: 4
```

Theme options:

- `forest`
- `ocean`
- `ember`
- `mono`

Font options:

- `ansi`
- `block`
- `thin`

Telegram notifications use environment variables in `.env`:

```bash
KAIRU_TELEGRAM_BOT_TOKEN=your_bot_token
KAIRU_TELEGRAM_CHAT_ID=your_chat_id
```

## CLI Commands

Export session data:

```bash
go run main.go --export backup.json
```

Import session data and merge it with the existing file:

```bash
go run main.go --import backup.json
```

`--export` and `--import` cannot be used together.

## Notifications

Kairu can notify on:

- session start
- session end
- work complete
- break complete
- pause/resume
- ending soon

Notification delivery can use:

- desktop notifications
- a local sound command
- Telegram

Quiet hours can be configured to suppress notifications during specific hours.

## Session Fields

Each stored session includes:

- task
- optional note
- optional tags
- start time
- end time
- duration in seconds
- type (`work` or `break`)

## Development

Run tests:

```bash
go test ./...
```

## Contributing

Contributions are welcome.

- Open an issue for bugs, ideas, or feature requests before larger changes.
- Keep changes focused and small when possible.
- Run `go test ./...` before submitting a pull request.
- Match the existing code style and keep the TUI keyboard-first workflow intact.

## Documentation

Additional docs:

- [docs/setup.md](docs/setup.md)
- [docs/usage.md](docs/usage.md)
- [docs/configuration.md](docs/configuration.md)
- [docs/telegram-notifications.md](docs/telegram-notifications.md)
