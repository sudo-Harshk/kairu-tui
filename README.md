# Kairu TUI

A TUI time tracker inspired by Pomodoro technique with ASCII art timer and activity analytics.

## Features

- **ASCII Art Timer** - Large, beautiful digital display
- **Custom Session Duration** - Set time per session (mm or hh:mm)
- **Weekly Bar Chart** - Visualize your 7-day activity
- **Telegram Notifications** - Get notified on your phone when work sessions end
- **Keyboard Controls** - Use Tab to switch fields, Space to pause, E to edit time
- **Work/Break Cycles** - Pomodoro-style productivity
- **Activity Dashboard** - Track streaks, ratios, and totals
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
- Architecture: [docs/architecture.md](docs/architecture.md)

## Configuration Snapshot

Create kairu.yaml in the project root and optionally .env for secrets. See full details in docs.

```bash
work_duration: 25
break_duration: 5
font: ansi
notifications: false
sound_command: ""
auto_break: false
sessions_before_break: 4
```

Environment variables (optional):

```bash
KAIRU_TELEGRAM_BOT_TOKEN=your_bot_token
KAIRU_TELEGRAM_CHAT_ID=your_chat_id
```

See the Telegram setup guide: [docs/telegram-notifications.md](docs/telegram-notifications.md)

