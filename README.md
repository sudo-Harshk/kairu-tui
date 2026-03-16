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

## Installation

### From Source

```bash
git clone https://github.com/yourusername/kairu-tui.git
cd kairu-tui
go install .
```

## Run Directly

```bash
go run main.go
```

## Configuration

### Create kairu.yaml in the project root:

```bash
work_duration: 25          # minutes
break_duration: 5          # minutes
notifications: true        # enable Telegram notifications
auto_break: false          # auto-suggest breaks
sessions_before_break: 4   # trigger break after N sessions
```

### Create .env in the project root (secrets)

```bash
KAIRU_TELEGRAM_BOT_TOKEN=your_bot_token
KAIRU_TELEGRAM_CHAT_ID=your_chat_id
```

### Telegram setup

Create a bot with @BotFather, then set `KAIRU_TELEGRAM_BOT_TOKEN`. Get your chat ID by messaging the bot and checking updates from the Telegram API, then set `KAIRU_TELEGRAM_CHAT_ID`.

##

