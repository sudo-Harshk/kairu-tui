package main

import (
	"strings"
	"testing"
	"time"
)

func TestTamagotchiDecayAndDeath(t *testing.T) {
	// Initialize default pet (Health: 100, Hunger: 20, Happiness: 80, Cleanliness: 100, Energy: 80)
	// We set Hunger to 20 so it never exceeds 20 during decay, which completely disables random poop generation.
	pet := DefaultPet("Testy", "kitty")
	pet.Hunger = 20

	// Set last tick to 4 hours ago to simulate active decay
	pet.LastTickTime = time.Now().Add(-4 * time.Hour)

	// Trigger decay to the current time
	pet.TickStateDecay(time.Now())

	// 4 hours is 16 ticks (15 mins each)
	// Hunger decays by 1 per tick (awake) => -16. Hunger should be 20 - 16 = 4
	if pet.Hunger != 4 {
		t.Errorf("Expected Hunger to be 4 after 4 hours decay, got %d", pet.Hunger)
	}

	// Happiness decays by 1 per tick => -16. Happiness should be 80 - 16 = 64
	if pet.Happiness != 64 {
		t.Errorf("Expected Happiness to be 64, got %d", pet.Happiness)
	}

	// Cleanliness decays by 1 per 2 ticks => -8. Cleanliness should be 100 - 8 = 92
	if pet.Cleanliness != 92 {
		t.Errorf("Expected Cleanliness to be 92, got %d", pet.Cleanliness)
	}

	// Energy decays by 1 per tick => -16. Energy should be 80 - 16 = 64
	if pet.Energy != 64 {
		t.Errorf("Expected Energy to be 64, got %d", pet.Energy)
	}

	// Let's force starvation and verify health decay
	// Reset other stats to perfect health so we only test Hunger=0 health decay deterministically
	pet.Health = 100
	pet.Hunger = 0
	pet.Poops = 0
	pet.IsSick = false
	pet.Cleanliness = 100
	pet.Happiness = 100
	pet.LastTickTime = time.Now().Add(-2 * time.Hour) // 8 ticks

	// Health was 100. Starving decays health by 2 per tick => -16 health
	pet.TickStateDecay(time.Now())
	if pet.Health != 84 {
		t.Errorf("Expected Health to be 84 after starving for 2 hours, got %d", pet.Health)
	}

	// Force death
	pet.Health = 5
	pet.Hunger = 0
	pet.Poops = 0
	pet.IsSick = false
	pet.Cleanliness = 100
	pet.Happiness = 100
	pet.LastTickTime = time.Now().Add(-1 * time.Hour) // 4 ticks => -8 health
	pet.TickStateDecay(time.Now())

	if !pet.IsDead {
		t.Error("Expected pet to be dead when health reaches 0")
	}
	if pet.Mood != "dead" {
		t.Errorf("Expected mood to be 'dead', got %s", pet.Mood)
	}
}

func TestOfflineSimulationCatchup(t *testing.T) {
	pet := DefaultPet("Neko", "kitty")

	// Put pet to sleep
	pet.IsSleeping = true
	// Energy starts at 80
	// Last tick set to 2 hours ago (8 ticks)
	pet.LastTickTime = time.Now().Add(-2 * time.Hour)

	pet.TickStateDecay(time.Now())

	// While sleeping, Energy increases by 4 per tick => 80 + 32 = 100 (capped)
	// It reaches 100 on tick 5, waking up automatically, and then decays by -1 on ticks 6, 7, 8 => 100 - 3 = 97
	if pet.Energy != 97 {
		t.Errorf("Expected Energy to be 97 after sleeping and waking up, got %d", pet.Energy)
	}
	// Sleeping automatically turns off when Energy reaches 100
	if pet.IsSleeping {
		t.Error("Expected pet to wake up automatically when fully charged")
	}
}

func TestPomoCoinEconomy(t *testing.T) {
	pet := DefaultPet("Testy", "kitty")
	pet.Coins = 20

	// 1. Buy valid item
	msg := pet.BuyItem("fish", 5)
	if pet.Coins != 15 {
		t.Errorf("Expected 15 coins left, got %d", pet.Coins)
	}
	if pet.Inventory["fish"] != 2 { // Started with 1 default fish, bought 1 => 2
		t.Errorf("Expected fish quantity to be 2, got %d", pet.Inventory["fish"])
	}
	if !strings.Contains(msg, "Successfully bought") {
		t.Errorf("Unexpected purchase message: %s", msg)
	}

	// 2. Try to buy when broke
	msg = pet.BuyItem("medicine", 25)
	if pet.Coins != 15 {
		t.Errorf("Coins changed on failed purchase")
	}
	if pet.Inventory["medicine"] != 0 {
		t.Errorf("Medicine added on failed purchase")
	}
	if !strings.Contains(msg, "Insufficient Pomo-Coins") {
		t.Errorf("Expected insufficient funds message, got: %s", msg)
	}
}

func TestFeedingHealCleanSleep(t *testing.T) {
	pet := DefaultPet("Neko", "kitty")

	// 1. Feeding Fish
	pet.Hunger = 50
	pet.Inventory["fish"] = 1
	msg, _ := pet.FeedItem("fish")
	if pet.Hunger != 80 {
		t.Errorf("Expected Hunger 80, got %d", pet.Hunger)
	}
	if pet.Inventory["fish"] != 0 {
		t.Error("Consumable was not deducted")
	}
	if !strings.Contains(msg, "delicious Fish") {
		t.Errorf("Unexpected message: %s", msg)
	}

	// 2. Clean Poops
	pet.Poops = 2
	pet.Cleanliness = 40
	msg = pet.CleanPoop()
	if pet.Poops != 0 {
		t.Errorf("Expected poops to be 0, got %d", pet.Poops)
	}
	if pet.Cleanliness != 100 {
		t.Errorf("Expected Cleanliness 100, got %d", pet.Cleanliness)
	}
	if !strings.Contains(msg, "Swept 2 poop") {
		t.Errorf("Unexpected message: %s", msg)
	}

	// 3. Cure Sickness
	pet.IsSick = true
	pet.Health = 50
	pet.Inventory["medicine"] = 1
	msg = pet.HealSick()
	if pet.IsSick {
		t.Error("Expected pet to be cured")
	}
	if pet.Health != 90 {
		t.Errorf("Expected Health 90, got %d", pet.Health)
	}
	if pet.Inventory["medicine"] != 0 {
		t.Error("Medicine was not consumed")
	}

	// 4. Toggle Sleep
	pet.IsSleeping = false
	msg = pet.ToggleSleep()
	if !pet.IsSleeping {
		t.Error("Expected pet to be sleeping")
	}
	if pet.Mood != "sleeping" {
		t.Errorf("Expected mood 'sleeping', got %s", pet.Mood)
	}
}
