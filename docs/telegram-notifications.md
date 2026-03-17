# Telegram Notifications

This guide sets up phone notifications when a work session ends.

1) Create a bot
- Open Telegram and chat with @BotFather
- Send /newbot and follow prompts
- Copy the HTTP API token (looks like 123456:ABC-DEF...)

2) Find your chat ID
- Option A: Message @userinfobot and read your ID
- Option B: Start a chat with your bot, then visit:
  https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates
  Look for message.chat.id in the JSON

3) Configure Kairu
- Create .env in the project root:

```dotenv
KAIRU_TELEGRAM_BOT_TOKEN=123456:ABCDEF_your_token
KAIRU_TELEGRAM_CHAT_ID=123456789
```

- In kairu.yaml set:

```yaml
notifications: true
```

4) Test a message (optional)
- Use curl to verify your token/chat ID:

```bash
curl -X POST \
  -d "chat_id=123456789" \
  -d "text=Hello from Kairu setup" \
  https://api.telegram.org/bot123456:ABCDEF_your_token/sendMessage
```

5) Run Kairu
- Finish a work session; you should receive a Telegram message:
  "Session completed: <task> (<duration>)"

Notes
- Notifications send only for work sessions, not breaks
- Both env vars must be set when notifications is true
- Secrets are not checked into git; keep .env private

Troubleshooting
- 400/401 errors: token or chat ID invalid
- No messages: ensure you started a chat with your bot at least once
- Windows: leave sound_command empty or use a compatible shell

