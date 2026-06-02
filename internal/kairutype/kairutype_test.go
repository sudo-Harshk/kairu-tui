package kairutype

import (
	"strings"
	"testing"
	"time"
)

func TestKairuTypeInitialization(t *testing.T) {
	state := InitKairuType("time", 30)

	if state.Active {
		t.Error("Expected test to not be active on initialization")
	}
	if state.Finished {
		t.Error("Expected test to not be finished on initialization")
	}
	if state.Mode != "time" {
		t.Errorf("Expected mode to be 'time', got %s", state.Mode)
	}
	if state.TargetTime != 30 {
		t.Errorf("Expected target time to be 30, got %d", state.TargetTime)
	}
	if len(state.WordsList) == 0 {
		t.Error("Expected generated words list to not be empty")
	}
	if len(state.TargetText) == 0 {
		t.Error("Expected generated target text to not be empty")
	}
}

func TestKairuTypeTypingAccuracy(t *testing.T) {
	state := InitKairuType("words", 10)
	state.TargetText = "hello world"

	// Type first correct character
	state.AddChar("h")
	if !state.Active {
		t.Error("Expected test to become active on typing first character")
	}
	if state.TypedText != "h" {
		t.Errorf("Expected typed text to be 'h', got %s", state.TypedText)
	}
	if state.Accuracy != 100.0 {
		t.Errorf("Expected accuracy to be 100%%, got %.1f%%", state.Accuracy)
	}

	// Type correct and incorrect characters
	state.AddChar("e")
	state.AddChar("x") // Should be 'l'
	if state.TypedText != "hex" {
		t.Errorf("Expected typed text to be 'hex', got %s", state.TypedText)
	}
	// 'h' (correct), 'e' (correct), 'x' (incorrect, target was 'l') -> 2/3 correct = 66.67%
	expectedAccuracy := (2.0 / 3.0) * 100.0
	if state.Accuracy < expectedAccuracy-0.1 || state.Accuracy > expectedAccuracy+0.1 {
		t.Errorf("Expected accuracy to be around %.2f%%, got %.2f%%", expectedAccuracy, state.Accuracy)
	}
	if state.ErrorCount != 1 {
		t.Errorf("Expected error count to be 1, got %d", state.ErrorCount)
	}
}

func TestKairuTypeBackspaceRollback(t *testing.T) {
	state := InitKairuType("words", 10)
	state.TargetText = "hello world"

	state.AddChar("h")
	state.AddChar("e")
	state.AddChar("x") // Error

	if state.ErrorCount != 1 {
		t.Errorf("Expected error count to be 1, got %d", state.ErrorCount)
	}

	// Trigger Backspace
	state.HandleBackspace()
	if state.TypedText != "he" {
		t.Errorf("Expected typed text to be rolled back to 'he', got %s", state.TypedText)
	}
	if state.Accuracy != 100.0 {
		t.Errorf("Expected accuracy to be restored to 100%%, got %.1f%%", state.Accuracy)
	}
}

func TestKairuTypeWordCompletion(t *testing.T) {
	state := InitKairuType("words", 2)
	state.WordsList = []string{"hello", "world"}
	state.TargetText = "hello world"

	// Simulate completing the words
	chars := strings.Split(state.TargetText, "")
	for _, ch := range chars {
		state.AddChar(ch)
	}

	if !state.Finished {
		t.Error("Expected test to finish when typing matches target words text length")
	}
}

func TestKairuTypeWPMSampling(t *testing.T) {
	state := InitKairuType("time", 15)
	state.TargetText = "hello world focus terminal typing monkeytype code"
	state.StartTest()

	// Simulate typing 20 correct characters in 1 second
	state.TypedText = "hello world focus te"
	state.Keystrokes = 20

	// Mocking elapsed time by setting start time back by 6 seconds
	state.StartTime = time.Now().Add(-6 * time.Second)

	state.SampleWPM(6)

	if len(state.WPMHistory) != 1 {
		t.Errorf("Expected WPM history length to be 1, got %d", len(state.WPMHistory))
	}
	if len(state.TimeSamples) != 1 {
		t.Errorf("Expected time samples length to be 1, got %d", len(state.TimeSamples))
	}
	if state.WPMHistory[0] <= 0 {
		t.Errorf("Expected sampled WPM to be positive, got %d", state.WPMHistory[0])
	}
}
