package notification

import (
	"strings"
	"testing"

	"kairu-tui/internal/config"
)

func TestDeliverNotificationConcurrent(t *testing.T) {
	// 1. Test case: No active channels enabled
	t.Run("NoChannels", func(t *testing.T) {
		cfg := config.Config{
			DesktopNotifications: false,
			SoundCommand:         "",
		}
		status, err := DeliverNotification(cfg, "Test", "Body")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status != "No active notification channels" {
			t.Fatalf("unexpected status: %q", status)
		}
	})

	// 2. Test case: Sound channel succeeds
	t.Run("SoundSuccess", func(t *testing.T) {
		cmd := "echo notification_test"
		cfg := config.Config{
			DesktopNotifications: false,
			SoundCommand:         cmd,
		}
		status, err := DeliverNotification(cfg, "Test", "Body")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(status, "Delivered via sound") {
			t.Fatalf("unexpected status: %q", status)
		}
	})

	// 3. Test case: Sound channel fails (executable not found)
	t.Run("SoundFailure", func(t *testing.T) {
		cfg := config.Config{
			DesktopNotifications: false,
			SoundCommand:         "invalid_command_nonexistent_xyz",
		}
		status, err := DeliverNotification(cfg, "Test", "Body")
		if err == nil {
			t.Fatalf("expected error, got nil status=%q", status)
		}
		if !strings.Contains(err.Error(), "all channels failed") || (!strings.Contains(err.Error(), "invalid_command_nonexistent_xyz") && !strings.Contains(err.Error(), "exit status 1") && !strings.Contains(err.Error(), "exit status 127")) {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}
