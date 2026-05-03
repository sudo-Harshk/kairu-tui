# Usage

Start the app:

```bash
go run main.go
```

Input mode:
- Type a task name
- Enter duration in minutes (e.g., 25) or hh:mm (e.g., 1:00)
- If templates exist, use Left/Right on the `Template:` selector to cycle presets
- Optional note field for quick session notes
- Optional tags field for comma-separated labels like `deep work, writing`
- Press Enter to advance through fields and start from the note field
- Tab switches fields
- Use Up/Down in the task field to cycle through recent tasks
- Press Ctrl+T to save the current form as a reusable template
- Press Ctrl+R to rename the selected template using the current task name
- Press Ctrl+D to delete the selected template
- Press ? to open the help overlay

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
- Tab to return
- S to open the settings panel
- ? for help

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
