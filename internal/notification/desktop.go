package notification

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// SendDesktopNotification sends a desktop notification.
func SendDesktopNotification(title, body string) error {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, body, title)
		return exec.Command("osascript", "-e", script).Run()
	case "linux":
		if err := exec.Command("notify-send", title, body).Run(); err == nil {
			return nil
		}
		return exec.Command("sh", "-c", fmt.Sprintf("printf '\\a'; printf '%s: %s\\n'", shellEscape(title), shellEscape(body))).Run()
	case "windows":
		script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$notify = New-Object System.Windows.Forms.NotifyIcon
$notify.Icon = [System.Drawing.SystemIcons]::Information
$notify.BalloonTipTitle = '%s'
$notify.BalloonTipText = '%s'
$notify.Visible = $true
$notify.ShowBalloonTip(3000)
$notify.Dispose()
`, psEscape(title), psEscape(body))
		return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script).Run()
	default:
		return fmt.Errorf("desktop notifications are not supported on %s", runtime.GOOS)
	}
}

func shellEscape(s string) string {
	s = strings.ReplaceAll(s, "'", "'\"'\"'")
	return "'" + s + "'"
}

func psEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
