# Usage

Start the app:

```bash
go run main.go
```

Input mode:
- Type a task name
- Enter duration in minutes (e.g., 25) or hh:mm (e.g., 1:00)
- Press **Ctrl+P** to open the template manager (see below)
- Optional note field for quick session notes
- Optional tags field for comma-separated labels like `deep work, writing`
- Press Enter to advance through fields and start from the note field
- Tab switches fields
- Use Up/Down in the task field to cycle through **suggested tasks** (pinned, file-based, or recent)
- Press **Ctrl+T** to save the current form as a reusable template
- Press ? to open the help overlay

## Template Manager

Press `Ctrl+P` from the input screen to manage your session templates.

- **Navigation:** Use `Up` / `Down` to select a template.
- **Select:** Press `Enter` to load the selected template into the input form.
- **Rename:** Press `Ctrl+R` to rename the selected template using the current task name in the input field.
- **Duplicate:** Press `Ctrl+Y` to create a copy of the selected template.
- **Delete:** Press `Ctrl+D` to delete the selected template.
- **Undo Delete:** Press `Ctrl+Z` immediately after deleting to restore the template.
- **Exit:** Press `Esc` or `Ctrl+P` to return to the input screen.

Timer mode:
- Space to pause/resume
- E to edit total session duration
- Enter to end session early
- Tab to open the stats dashboard
- S to open the settings panel
- ? for help
- q to quit (saves session if running)

Break mode:
- Behaves like timer mode but tracks break time

Stats dashboard:
- Shows today’s total work time
- Current and longest streaks
- Weekly activity bar chart (last 7 days)
- Work/break ratio for this run
- Recovery status (see below)
- Tab to return
- S to open the settings panel
- ? for help

## Streak & Recovery

A **streak** counts the number of consecutive calendar days on which you completed at least one work session.

### Streak indicator

The timer header always shows your current streak state:

| Display | Meaning |
| --- | --- |
| `🔥 N` | Active streak — you have worked N days in a row including today |
| `✦ recoverable` | You missed yesterday but worked the day before; complete a session today to save the streak |
| `◌ rebuild` | The streak has been lost; start fresh today |

### Recovery mode

If you miss exactly one day, Kairu enters **Recovery mode**. The input screen will show an encouraging prompt:

> ✦ Recovery mode — complete a session today to save your streak!

Complete any work session before midnight and your streak is preserved.

If you miss two or more consecutive days the streak resets and the prompt changes to encourage you to start a new streak.

### Stats dashboard

The Analytics/Stats screen (press `Tab` during a timer session) shows:

- **Current streak** — consecutive days including today
- **Best streak** — your all-time longest streak
- **Recovery status** — whether recovery is available, active today, or the streak is lost

## Soundscapes

Press `Ctrl+M` from the input screen or while a timer is running to open the Soundscape selector.

- Use `Up` / `Down` to navigate the list of audio files found in `soundscapes_dir`
- Press `Enter` to activate the highlighted track (or select `None` to stop playback)
- Press `Esc` or `Ctrl+M` again to cancel
- Playback only runs during work sessions; it stops automatically on break, end, or quit
- The active track is shown in the timer header as `🎵 Track Name`

To add tracks, drop `.mp3`, `.wav`, `.ogg`, `.flac`, or `.aac` files into the configured `soundscapes_dir` directory (default: `soundscapes/`).

Errors:
- Input validation errors show inline under the fields.
- Runtime errors (file I/O, notifications) appear in red in the UI.

Data:
- Sessions are appended to entries.json in the project directory
- Each entry includes task, optional note, optional tags, start, end, duration (seconds), and type (work/break)
- Session templates are stored in templates.json in the project directory

CLI data utilities:
- Export entries to a file:

```bash
go run main.go --export backup.json
```

- Import entries from a file (merged with existing data, duplicates removed):

```bash
go run main.go --import backup.json
```
