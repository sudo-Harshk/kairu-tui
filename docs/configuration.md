# Configuration

Kairu reads kairu.yaml and .env from the project root. Missing files are allowed; defaults apply.

YAML (kairu.yaml):
- work_duration (int, minutes) — default 25
- break_duration (int, minutes) — default 5
- font (string) — default "ansi" (reserved for future font options)
- notifications (bool) — master switch for notifications
- desktop_notifications (bool) — enable local desktop notifications
- notify_work_complete (bool) — notify when a work session ends
- notify_break_complete (bool) — notify when a break ends
- notify_session_start (bool) — notify when a session starts
- notify_session_end (bool) — notify when a session is ended manually
- notify_pause_resume (bool) — notify when the timer is paused or resumed
- notify_ending_soon (bool) — notify when a session is almost done
- sound_command (string) — optional shell command to play a sound after notification
- auto_break (bool) — suggest a break automatically after N sessions
- sessions_before_break (int) — default 4

Env (.env):
- KAIRU_TELEGRAM_BOT_TOKEN — token from @BotFather
- KAIRU_TELEGRAM_CHAT_ID — chat ID (user or group)

Override rules:
- YAML loads first; env values override Telegram fields
- If notifications is true, Telegram env vars are required only for Telegram delivery

Storage:
- entries.json stores session history in JSON array form
- notification_outbox.json stores pending notification retries

Keyboard:
- Tab switches views/fields
- Space pauses/resumes
- Enter confirms
- E edits duration
- q quits
