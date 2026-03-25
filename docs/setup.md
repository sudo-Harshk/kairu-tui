# Setup

Prerequisites:
- Go 1.21+ installed
- Git and a terminal

Clone:

```bash
git clone https://github.com/yourusername/kairu-tui.git
cd kairu-tui
```

Run without modules:

```bash
go run main.go
```

Install binary to $GOPATH/bin:

```bash
go install .
```

Configuration files:
- kairu.yaml — non-sensitive configuration, placed in the project root (ignored by git)
- .env — secrets like Telegram token/chat id (ignored by git)

Create kairu.yaml:

```yaml
work_duration: 25
break_duration: 5
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

Create .env for Telegram (optional):

```dotenv
KAIRU_TELEGRAM_BOT_TOKEN=your_bot_token
KAIRU_TELEGRAM_CHAT_ID=your_chat_id
```

Windows notes:
- Use PowerShell or Windows Terminal
- Desktop notifications use a PowerShell toast fallback on Windows
- Sound command executes via sh -c; leave empty on Windows or use a compatible shell
