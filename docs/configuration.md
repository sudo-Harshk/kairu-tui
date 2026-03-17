# Configuration

Kairu reads kairu.yaml and .env from the project root. Missing files are allowed; defaults apply.

YAML (kairu.yaml):
- work_duration (int, minutes) — default 25
- break_duration (int, minutes) — default 5
- font (string) — default "ansi" (reserved for future font options)
- notifications (bool) — enable Telegram notifications for completed work sessions
- sound_command (string) — optional shell command to play a sound after notification
- auto_break (bool) — suggest a break automatically after N sessions
- sessions_before_break (int) — default 4

Env (.env):
- KAIRU_TELEGRAM_BOT_TOKEN — token from @BotFather
- KAIRU_TELEGRAM_CHAT_ID — chat ID (user or group)

Override rules:
- YAML loads first; env values override Telegram fields
- If notifications is true, both env vars are required for sending messages

Storage:
- entries.json stores session history in JSON array form

Keyboard:
- Tab switches views/fields
- Space pauses/resumes
- Enter confirms
- E edits duration
- q quits

