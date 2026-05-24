# Notification Design Notes

This file tracks the next notification system we want to build for Kairu.

## Goal

Make notifications feel reliable, useful, and native to the desktop app instead of bolting on a single external channel.

## What we want to implement

- A notification engine that handles session events like start, pause, end, and break reminders.
- Desktop notifications as the primary delivery path.
- Optional fallback channels later, such as terminal alerts or Telegram.
- A small queue so notifications are not lost if the app restarts or the desktop API fails.
- Retry logic with backoff for temporary failures.
- Deduping so the same event does not notify twice.
- User settings for sound, quiet hours, and which events should notify.

## Design direction

- Keep the messages short and actionable.
- Make success states gentle and failure states obvious.
- Prefer local desktop notifications first, remote channels second.
- Keep the notification layer separate from the timer UI so it is easier to maintain.

## Next step

Define the event list and the notification state model before implementing anything.
