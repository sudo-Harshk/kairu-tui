# Kairu TUI

A TUI time tracker inspired by Pomodoro technique with ASCII art timer and activity analytics.

## Features

- **ASCII Art Timer** - Large, beautiful digital display
- **Custom Session Duration** - Set time per session (mm or hh:mm)
- **Weekly Bar Chart** - Visualize your 7-day activity
- **Desktop Notifications** - Get notified locally when sessions start, end, or finish
- **Notification Settings Panel** - Toggle alerts and quiet hours from inside the app
- **Keyboard Controls** - Use Tab to switch fields or open stats, Space to pause, E to edit time, S for settings
- **Recent Task Recall** - Use Up/Down in the task field to reuse recently tracked tasks
- **Session Notes** - Add an optional note to each session before starting
- **Task Tags** - Add comma-separated tags to sessions for lighter grouping
- **Session Templates** - Save and reuse preset task setups with one shortcut
- **Help Overlay** - Press ? to view keybindings anytime
- **In-app Error Messages** - Runtime issues surface inside the UI
- **Work/Break Cycles** - Pomodoro-style productivity
- **Activity Dashboard** - Track streaks, ratios, and totals
- **Theme and Font Customization** - Switch color themes and timer styles in-app
- **Local Storage** - All data stays on your machine
- **Session Chaining** - Seamless workflow between tasks

## Quick Start

- Install Go 1.21+
- Clone and run:

```bash
git clone https://github.com/yourusername/kairu-tui.git
cd kairu-tui
go run main.go
```

Optional: install the binary to $GOPATH/bin

```bash
go install .
```

## Documentation

- Setup: [docs/setup.md](docs/setup.md)
- Usage: [docs/usage.md](docs/usage.md)
- Configuration: [docs/configuration.md](docs/configuration.md)
- Telegram Notifications: [docs/telegram-notifications.md](docs/telegram-notifications.md)
- Overview: [docs/overview.md](docs/overview.md)

## Configuration Snapshot

Create kairu.yaml in the project root and optionally .env for secrets. See full details in docs.

```bash
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

Theme options: `forest`, `ocean`, `ember`, `mono`

Font options: `ansi`, `block`, `thin`

Environment variables for Telegram delivery only (optional):

```bash
KAIRU_TELEGRAM_BOT_TOKEN=your_bot_token
KAIRU_TELEGRAM_CHAT_ID=your_chat_id
```

See the Telegram setup guide: [docs/telegram-notifications.md](docs/telegram-notifications.md)

