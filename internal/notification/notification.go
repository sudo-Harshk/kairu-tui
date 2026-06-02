package notification

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

type NotificationJob struct {
	ID            string    `json:"id"`
	Event         string    `json:"event"`
	Title         string    `json:"title"`
	Body          string    `json:"body"`
	CreatedAt     time.Time `json:"created_at"`
	Attempts      int       `json:"attempts"`
	NextAttemptAt time.Time `json:"next_attempt_at"`
	LastError     string    `json:"last_error,omitempty"`
}

func LoadNotificationOutbox(path string) ([]NotificationJob, error) {
	var jobs []NotificationJob
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return jobs, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func SaveNotificationOutbox(path string, jobs []NotificationJob) error {
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

var NotificationBackoff = []time.Duration{
	2 * time.Second,
	5 * time.Second,
	15 * time.Second,
}

func NewNotificationJob(id, event, title, body string) NotificationJob {
	return NotificationJob{
		ID:        id,
		Event:     event,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now(),
	}
}

func ScheduleNextAttempt(job *NotificationJob) {
	delay := NotificationBackoff[len(NotificationBackoff)-1]
	if job.Attempts > 0 && job.Attempts <= len(NotificationBackoff) {
		delay = NotificationBackoff[job.Attempts-1]
	}
	job.NextAttemptAt = time.Now().Add(delay)
}

