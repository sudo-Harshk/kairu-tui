# Usage

Start the app:

```bash
go run main.go
```

Input mode:
- Type a task name
- Enter duration in minutes (e.g., 25) or hh:mm (e.g., 1:00)
- Press Enter to start, Tab to switch fields
- Press ? to open the help overlay

Timer mode:
- Space to pause/resume
- E to edit total session duration
- Enter to end session early
- Tab to open the stats dashboard
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
- ? for help

Errors:
- Input validation errors show inline under the fields.
- Runtime errors (file I/O, notifications) appear in red in the UI.

Data:
- Sessions are appended to entries.json in the project directory
- Each entry includes task, start, end, duration (seconds), and type (work/break)
