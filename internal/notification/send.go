package notification

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"kairu-tui/internal/config"
)

// SendNotification sends a notification represented by a NotificationJob.
func SendNotification(cfg config.Config, job NotificationJob) (string, error) {
	if strings.TrimSpace(job.Body) == "" {
		return "", fmt.Errorf("notification body is empty")
	}
	return DeliverNotification(cfg, job.Title, job.Body)
}

// DeliverNotification delivers the notification payload through enabled channels.
func DeliverNotification(cfg config.Config, title, body string) (string, error) {
	var successes []string
	var failures []string

	if cfg.DesktopNotifications {
		if err := SendDesktopNotification(title, body); err == nil {
			successes = append(successes, "desktop")
		} else {
			failures = append(failures, fmt.Sprintf("desktop: %v", err))
		}
	}

	if cfg.SoundCommand != "" {
		var soundErr error
		if runtime.GOOS == "windows" {
			soundErr = exec.Command("cmd", "/c", cfg.SoundCommand).Run()
		} else {
			soundErr = exec.Command("sh", "-c", cfg.SoundCommand).Run()
		}
		if soundErr == nil {
			successes = append(successes, "sound")
		} else {
			failures = append(failures, fmt.Sprintf("sound: %v", soundErr))
		}
	}

	if token := strings.TrimSpace(cfg.TelegramBotToken); token != "" && strings.TrimSpace(cfg.TelegramChatID) != "" {
		if err := SendTelegramMessage(token, strings.TrimSpace(cfg.TelegramChatID), body); err == nil {
			successes = append(successes, "telegram")
		} else {
			failures = append(failures, fmt.Sprintf("telegram: %v", err))
		}
	}

	if len(successes) > 0 {
		status := "Delivered via " + strings.Join(successes, ", ")
		if len(failures) > 0 {
			status += " (failed: " + strings.Join(failures, "; ") + ")"
		}
		return status, nil
	}

	if len(failures) > 0 {
		return "", fmt.Errorf("all channels failed: %s", strings.Join(failures, "; "))
	}
	return "No active notification channels", nil
}
