package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPetInitialization(t *testing.T) {
	pet := DefaultPet("Neko", "kitty")
	if pet.Name != "Neko" {
		t.Errorf("Expected Name to be Neko, got %s", pet.Name)
	}
	if pet.Type != "kitty" {
		t.Errorf("Expected Type to be kitty, got %s", pet.Type)
	}
	if pet.Level != 1 {
		t.Errorf("Expected Level to be 1, got %d", pet.Level)
	}
	if pet.Experience != 0 {
		t.Errorf("Expected Experience to be 0, got %d", pet.Experience)
	}
}

func TestPetXPScalingAndLevelUp(t *testing.T) {
	pet := DefaultPet("Neko", "kitty")

	// Level 1 threshold is 1 * 250 = 250 XP
	leveledUp := pet.AddXP(200)
	if leveledUp {
		t.Error("Should not have leveled up at 200 XP")
	}
	if pet.Level != 1 || pet.Experience != 200 {
		t.Errorf("Expected Level 1 with 200 XP, got Level %d with %d XP", pet.Level, pet.Experience)
	}

	// Trigger level up to Level 2
	leveledUp = pet.AddXP(100)
	if !leveledUp {
		t.Error("Expected level up to occur")
	}
	// Target is 250, so 300 - 250 = 50 XP in Level 2
	if pet.Level != 2 || pet.Experience != 50 {
		t.Errorf("Expected Level 2 with 50 XP, got Level %d with %d XP", pet.Level, pet.Experience)
	}

	// Level 2 threshold is 2 * 250 = 500 XP
	// Let's add 500 XP to trigger Level 3
	leveledUp = pet.AddXP(450)
	if !leveledUp {
		t.Error("Expected level up to Level 3")
	}
	if pet.Level != 3 || pet.Experience != 0 {
		t.Errorf("Expected Level 3 with 0 XP, got Level %d with %d XP", pet.Level, pet.Experience)
	}
}

func TestPetEvolutionStage(t *testing.T) {
	pet := DefaultPet("Neko", "kitty")
	if pet.EvolutionStage() != 1 {
		t.Errorf("Expected Level 1 to be Evolution Stage 1, got %d", pet.EvolutionStage())
	}

	pet.Level = 4
	if pet.EvolutionStage() != 2 {
		t.Errorf("Expected Level 4 to be Evolution Stage 2, got %d", pet.EvolutionStage())
	}

	pet.Level = 8
	if pet.EvolutionStage() != 3 {
		t.Errorf("Expected Level 8 to be Evolution Stage 3, got %d", pet.EvolutionStage())
	}
}

func TestPetMoodTransitions(t *testing.T) {
	pet := DefaultPet("Neko", "kitty")

	// Timer running work session
	pet.UpdateMood(true, "timer", time.Now())
	if pet.Mood != "working" {
		t.Errorf("Expected working mood during work timer, got %s", pet.Mood)
	}

	// Timer running break session
	pet.UpdateMood(true, "break", time.Now())
	if pet.Mood != "happy" {
		t.Errorf("Expected happy mood during break timer, got %s", pet.Mood)
	}

	// Paused timer
	pet.UpdateMood(false, "timer", time.Now())
	if pet.Mood != "grumpy" {
		t.Errorf("Expected grumpy mood when paused, got %s", pet.Mood)
	}

	// Idle state (recently fed)
	pet.LastFedTime = time.Now()
	pet.UpdateMood(false, "input", time.Now())
	// Should be idle or sleeping depending on local time
	hour := time.Now().Hour()
	if hour >= 23 || hour < 6 {
		if pet.Mood != "sleeping" {
			t.Errorf("Expected sleeping mood at late night, got %s", pet.Mood)
		}
	} else {
		if pet.Mood != "idle" {
			t.Errorf("Expected idle mood during normal hours, got %s", pet.Mood)
		}
	}

	// Neglect mode (inactive > 48 hours)
	pet.LastFedTime = time.Now().Add(-50 * time.Hour)
	pet.UpdateMood(false, "input", time.Now())
	if pet.Mood != "grumpy" {
		t.Errorf("Expected grumpy mood after 48h neglect, got %s", pet.Mood)
	}
}

func TestPetAnimations(t *testing.T) {
	pet := DefaultPet("Neko", "kitty")
	pet.Mood = "idle"
	pet.Level = 1

	// Check that frame 0 and frame 1 produce different ASCII art (frame 1 is blinking)
	ascii0 := pet.GetASCII(0)
	ascii1 := pet.GetASCII(1)

	if ascii0 == ascii1 {
		t.Error("Expected animated idle ASCII art frames to be different (blinking)")
	}

	if !strings.Contains(ascii1, "-.-") {
		t.Error("Expected frame 1 idle to contain blinking eyes (-.-)")
	}
}

func TestPetJSONPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pet-test")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "pet.json")
	originalPet := DefaultPet("Neko", "kitty")
	originalPet.Level = 3
	originalPet.Experience = 120

	err = SavePetState(filePath, originalPet)
	if err != nil {
		t.Fatalf("Failed to save pet state: %v", err)
	}

	loadedPet, err := LoadPetState(filePath)
	if err != nil {
		t.Fatalf("Failed to load pet state: %v", err)
	}

	if loadedPet.Name != originalPet.Name || loadedPet.Level != originalPet.Level || loadedPet.Experience != originalPet.Experience {
		t.Errorf("Loaded pet doesn't match original. Got %+v, want %+v", loadedPet, originalPet)
	}
}
