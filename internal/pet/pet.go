package pet

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// PetState represents the persistent status of the companion
type PetState struct {
	Name             string         `json:"name"`
	Type             string         `json:"type"`                // Always "kitty" (Neko)
	Level            int            `json:"level"`               // 1 to 10
	Experience       int            `json:"experience"`          // Current XP in this level
	TotalSessionsFed int            `json:"total_sessions_fed"`  // Completed Pomodoros
	Mood             string         `json:"mood"`                // "idle", "working", "happy", "sleeping", "grumpy", "sick", "dead"
	LastFedTime      time.Time      `json:"last_fed_time"`       // Track to detect inactivity/neglect
	ActiveItem       string         `json:"active_item"`         // Discovered cosmetic items (e.g., "Wizard Hat", "Cyber Visor")
	
	// Interactive Cyber-Tamagotchi fields
	Hunger       int            `json:"hunger"`         // 0 to 100
	Happiness    int            `json:"happiness"`      // 0 to 100
	Cleanliness  int            `json:"cleanliness"`    // 0 to 100
	Energy       int            `json:"energy"`         // 0 to 100
	Health       int            `json:"health"`         // 0 to 100
	IsSick       bool           `json:"is_sick"`        // Sick status
	Poops        int            `json:"poops"`          // Number of poops on screen (0 to 3)
	IsSleeping   bool           `json:"is_sleeping"`    // Light switch (sleeping state)
	IsDead       bool           `json:"is_dead"`        // Dead status
	AgeDays      int            `json:"age_days"`       // Pet lifetime in virtual days
	Coins        int            `json:"coins"`          // In-game currency
	Inventory    map[string]int `json:"inventory"`      // Food & medical items
	LastTickTime time.Time      `json:"last_tick_time"` // For offline decay calculations

	// Transient fields for active focus session interventions (ignored in JSON)
	SessionElapsed int            `json:"-"`
	TimerRunning   bool           `json:"-"`
	TimerMode      string         `json:"-"`
}

// DefaultPet creates a basic starting pet (Neko the Robo-Kitty)
func DefaultPet(name string, petType string) PetState {
	if name == "" {
		name = "Neko"
	}
	inventory := map[string]int{
		"fish":     1,
		"treat":    0,
		"drink":    0,
		"medicine": 0,
	}
	return PetState{
		Name:             name,
		Type:             "kitty",
		Level:            1,
		Experience:       0,
		TotalSessionsFed: 0,
		Mood:             "idle",
		LastFedTime:      time.Now(),
		ActiveItem:       "",
		Hunger:           80,
		Happiness:        80,
		Cleanliness:      100,
		Energy:           80,
		Health:           100,
		IsSick:           false,
		Poops:            0,
		IsSleeping:       false,
		IsDead:           false,
		AgeDays:          0,
		Coins:            10,
		Inventory:        inventory,
		LastTickTime:     time.Now(),
	}
}

// LoadPetState loads the companion data from JSON
func LoadPetState(path string) (PetState, error) {
	var state PetState
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state, err
		}
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	return state, nil
}

// SavePetState writes the companion data to JSON
func SavePetState(path string, state PetState) error {
	state.Type = "kitty" // Force type to kitty
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// AddXP adds XP to the pet and returns true if a Level Up occurred
func (p *PetState) AddXP(amount int) bool {
	p.Experience += amount
	leveledUp := false

	// Threshold for next level is Level * 250
	for p.Experience >= p.Level*250 {
		p.Experience -= p.Level * 250
		p.Level++
		leveledUp = true
		if p.Level > 10 {
			p.Level = 10
			p.Experience = 0
			break
		}
	}
	return leveledUp
}

// Feed awards a focus session's nutrition, adding XP and updating stats
func (p *PetState) Feed(durationMinutes int) bool {
	p.TotalSessionsFed++
	p.LastFedTime = time.Now()
	p.Mood = "happy"

	// Award 10 XP per minute of focus session, capped at 300 XP
	xpGained := durationMinutes * 10
	if xpGained > 300 {
		xpGained = 300
	}
	if xpGained <= 0 {
		xpGained = 50 // Minimum session reward
	}

	return p.AddXP(xpGained)
}

// UpdateMood determines the current pet mood based on app state and time
func (p *PetState) UpdateMood(timerRunning bool, timerMode string, lastActiveSession time.Time) {
	if p.IsDead {
		p.Mood = "dead"
		return
	}
	if p.IsSick {
		p.Mood = "sick"
		return
	}
	if p.IsSleeping {
		p.Mood = "sleeping"
		return
	}

	if timerRunning {
		if timerMode == "break" {
			p.Mood = "happy"
		} else {
			p.Mood = "working"
		}
		return
	}

	// If paused during an active timer, it's slightly grumpy/impatient
	if timerMode == "timer" || timerMode == "break" {
		p.Mood = "grumpy"
		return
	}

	// Calculate time since last fed/focused
	hoursSinceActivity := time.Since(p.LastFedTime).Hours()
	if hoursSinceActivity > 48 {
		p.Mood = "grumpy"
	} else if hoursSinceActivity > 24 {
		p.Mood = "idle"
	} else {
		// Late night sleeping logic
		hour := time.Now().Hour()
		if hour >= 23 || hour < 6 {
			p.Mood = "sleeping"
		} else {
			p.Mood = "idle"
		}
	}
}

// TickStateDecay simulates the decay of stats over elapsed time (both online and offline)
func (p *PetState) TickStateDecay(currentTime time.Time) {
	if p.IsDead {
		p.Mood = "dead"
		p.LastTickTime = currentTime
		return
	}
	if p.LastTickTime.IsZero() {
		p.LastTickTime = currentTime
		return
	}

	elapsed := currentTime.Sub(p.LastTickTime)
	interval := 15 * time.Minute
	if elapsed < interval {
		return // Not enough time passed for a 15-minute simulation step
	}

	numTicks := int(elapsed / interval)
	if numTicks > 480 {
		// Cap simulation at 5 days to avoid instant death from long absences
		numTicks = 480
	}

	// Simple LCG pseudo-random generator to remain deterministic and fast
	randSeed := currentTime.UnixNano()

	for i := 0; i < numTicks; i++ {
		p.LastTickTime = p.LastTickTime.Add(interval)

		if p.IsDead {
			break
		}

		// Increase AgeDays once per 24 hours (roughly when crossing midnight)
		if p.LastTickTime.Hour() == 0 && p.LastTickTime.Minute() < 15 {
			p.AgeDays++
		}

		// 1. Energy Decay/Recovery
		if p.IsSleeping {
			p.Energy += 4
			if p.Energy >= 100 {
				p.Energy = 100
				p.IsSleeping = false // Wakes up automatically when fully charged
			}
		} else {
			p.Energy -= 1
			if p.Energy < 0 {
				p.Energy = 0
			}
		}

		// 2. Hunger Decay
		if p.IsSleeping {
			// Decays 4x slower while asleep (1 per hour)
			if i%4 == 0 {
				p.Hunger -= 1
			}
		} else {
			// Decays 1 per interval (4 per hour)
			p.Hunger -= 1
		}
		if p.Hunger < 0 {
			p.Hunger = 0
		}

		// 3. Happiness Decay
		if p.IsSleeping {
			// No decay during peaceful sleep
		} else {
			decayAmt := 1
			// Neglected bonus decay: if no focus session in > 24 hours
			if time.Since(p.LastFedTime) > 24*time.Hour {
				decayAmt = 2
			}
			p.Happiness -= decayAmt
			if p.Happiness < 0 {
				p.Happiness = 0
			}
		}

		// 4. Cleanliness Decay
		if i%2 == 0 {
			p.Cleanliness -= 1
			if p.Cleanliness < 0 {
				p.Cleanliness = 0
			}
		}

		// Random Poop Generation (2% chance per 15 mins if awake and not starving)
		randSeed = (randSeed*1103515245 + 12345) & 0x7fffffff
		if !p.IsSleeping && p.Hunger > 20 && p.Poops < 3 && int(randSeed%100) < 2 {
			p.Poops++
			p.Cleanliness -= 15
			if p.Cleanliness < 0 {
				p.Cleanliness = 0
			}
		}

		// 5. Sickness Triggering
		if !p.IsSick {
			// Chance based on poops on screen
			randSeed = (randSeed*1103515245 + 12345) & 0x7fffffff
			if p.Poops > 0 && int(randSeed%100) < (3*p.Poops) {
				p.IsSick = true
			}
			// Chance based on filth
			randSeed = (randSeed*1103515245 + 12345) & 0x7fffffff
			if p.Cleanliness < 30 && int(randSeed%100) < 10 {
				p.IsSick = true
			}
		}

		// 6. Health Calculations
		healthChange := 0
		if p.IsSick {
			healthChange -= 2
		}
		if p.Hunger == 0 {
			healthChange -= 2
		}
		if p.Happiness == 0 {
			healthChange -= 1
		}
		if p.Poops > 0 {
			healthChange -= 1
		}

		// Natural Health Recovery
		if healthChange == 0 && p.Hunger > 50 && p.Happiness > 50 && p.Cleanliness > 50 && !p.IsSick {
			healthChange += 1
		}

		p.Health += healthChange
		if p.Health > 100 {
			p.Health = 100
		}
		if p.Health <= 0 {
			p.Health = 0
			p.IsDead = true
			p.IsSleeping = false
		}
	}

	p.LastTickTime = currentTime

	// Resolve final mood state
	if p.IsDead {
		p.Mood = "dead"
	} else if p.IsSick {
		p.Mood = "sick"
	} else if p.IsSleeping {
		p.Mood = "sleeping"
	} else if p.Hunger < 20 || p.Happiness < 20 || p.Poops > 0 {
		p.Mood = "grumpy"
	} else if p.Hunger > 70 && p.Happiness > 70 {
		p.Mood = "happy"
	} else {
		p.Mood = "idle"
	}
}

// FeedItem feeds an item from the inventory, returning feedback and whether a level up occurred
func (p *PetState) FeedItem(item string) (string, bool) {
	if p.IsDead {
		return "Your companion is dead. Rebirth them first.", false
	}
	if p.IsSleeping {
		return "Shhh! Your companion is sleeping. Wake them first!", false
	}
	if p.Inventory == nil {
		p.Inventory = make(map[string]int)
	}
	if p.Inventory[item] <= 0 {
		return fmt.Sprintf("You don't have any %s left!", item), false
	}

	p.Inventory[item]--
	p.LastFedTime = time.Now()

	switch item {
	case "fish":
		p.Hunger += 30
		if p.Hunger > 100 {
			p.Hunger = 100
		}
		p.Happiness += 5
		if p.Happiness > 100 {
			p.Happiness = 100
		}
		// A feed might cause poop!
		p.Poops++
		if p.Poops > 3 {
			p.Poops = 3
		}
		p.Cleanliness -= 10
		if p.Cleanliness < 0 {
			p.Cleanliness = 0
		}
		leveledUp := p.AddXP(15)
		return fmt.Sprintf("Fed %s delicious Fish! Hunger +30, Happiness +5.", p.Name), leveledUp

	case "treat":
		p.Hunger += 20
		if p.Hunger > 100 {
			p.Hunger = 100
		}
		p.Happiness += 30
		if p.Happiness > 100 {
			p.Happiness = 100
		}
		p.Energy += 10
		if p.Energy > 100 {
			p.Energy = 100
		}
		leveledUp := p.AddXP(25)
		return fmt.Sprintf("Gave %s a Cyber Cookie! Hunger +20, Happiness +30, Energy +10.", p.Name), leveledUp

	case "drink":
		p.Energy += 40
		if p.Energy > 100 {
			p.Energy = 100
		}
		p.Happiness += 10
		if p.Happiness > 100 {
			p.Happiness = 100
		}
		p.Cleanliness -= 10
		if p.Cleanliness < 0 {
			p.Cleanliness = 0
		}
		leveledUp := p.AddXP(20)
		return fmt.Sprintf("Offered %s a Cyber Cola! Energy +40, Happiness +10, Cleanliness -10.", p.Name), leveledUp

	default:
		p.Inventory[item]++ // Refund
		return "Unknown consumable.", false
	}
}

// CleanPoop sweeps poop off screen and resets cleanliness
func (p *PetState) CleanPoop() string {
	if p.IsDead {
		return "Your companion is dead. Rebirth them first."
	}
	if p.Poops == 0 {
		return "The screens are already sparkling clean!"
	}
	cleaned := p.Poops
	p.Poops = 0
	p.Cleanliness = 100
	p.Happiness += 10
	if p.Happiness > 100 {
		p.Happiness = 100
	}
	return fmt.Sprintf("Swept %d poop(s) away! Cleanliness is back to 100%%.", cleaned)
}

// HealSick administers medicine to cure sickness
func (p *PetState) HealSick() string {
	if p.IsDead {
		return "Your companion is dead. Rebirth them first."
	}
	if !p.IsSick {
		return fmt.Sprintf("%s is perfectly healthy!", p.Name)
	}
	if p.Inventory == nil {
		p.Inventory = make(map[string]int)
	}
	if p.Inventory["medicine"] <= 0 {
		return "You don't have any medicine left!"
	}
	p.Inventory["medicine"]--
	p.IsSick = false
	p.Health += 40
	if p.Health > 100 {
		p.Health = 100
	}
	return fmt.Sprintf("Injected Anti-Virus medicine! %s is fully cured. Health +40.", p.Name)
}

// ToggleSleep toggles the sleep lighting state
func (p *PetState) ToggleSleep() string {
	if p.IsDead {
		return "Your companion is dead. Rebirth them first."
	}
	p.IsSleeping = !p.IsSleeping
	if p.IsSleeping {
		p.Mood = "sleeping"
		return fmt.Sprintf("Turned off the lights. %s went to sleep. Zzz...", p.Name)
	} else {
		p.Mood = "idle"
		return fmt.Sprintf("Turned on the lights. %s woke up!", p.Name)
	}
}

// BuyItem purchases a food/medicine item from the shop
func (p *PetState) BuyItem(item string, cost int) string {
	if p.Coins < cost {
		return fmt.Sprintf("Insufficient Pomo-Coins! Costs %d, but you only have %d.", cost, p.Coins)
	}
	p.Coins -= cost
	if p.Inventory == nil {
		p.Inventory = make(map[string]int)
	}
	p.Inventory[item]++
	return fmt.Sprintf("Successfully bought %s for %d Pomo-Coins!", item, cost)
}

// RebirthPet resets the companion while preserving currency and cosmetic loot
func (p *PetState) RebirthPet(newName string) string {
	if newName == "" {
		newName = "Neko"
	}
	p.Name = newName
	p.Level = 1
	p.Experience = 0
	p.TotalSessionsFed = 0
	p.Mood = "idle"
	p.LastFedTime = time.Now()
	p.Hunger = 80
	p.Happiness = 80
	p.Cleanliness = 100
	p.Energy = 80
	p.Health = 100
	p.IsSick = false
	p.Poops = 0
	p.IsSleeping = false
	p.IsDead = false
	p.AgeDays = 0
	p.LastTickTime = time.Now()
	return fmt.Sprintf("%s has been reborn from the digital ashes!", p.Name)
}

// EvolutionStage determines the visual phase based on level
func (p *PetState) EvolutionStage() int {
	if p.Level >= 8 {
		return 3 // Cyber-Ascended
	}
	if p.Level >= 4 {
		return 2 // Adolescent
	}
	return 1 // Baby
}

// GetASCII returns the multi-line visual representation of the pet based on the animation frame
func (p *PetState) GetASCII(frame int) string {
	stage := p.EvolutionStage()
	baseArt := getKittyASCII(stage, p.Mood, frame)

	// Hook active cosmetic item onto the top of the ASCII art
	if p.ActiveItem != "" {
		hatLines := getCosmeticHatASCII(p.ActiveItem)
		return hatLines + "\n" + baseArt
	}

	return baseArt
}

// getCosmeticHatASCII returns small custom ASCII overlays for earned items
func getCosmeticHatASCII(item string) string {
	switch item {
	case "Wizard Hat":
		return "    /\\    \n   /  \\   \n  /____\\  \n (======)"
	case "Golden Crown":
		return "  /\\/\\/\\  \n  [====] "
	case "Cyber Visor":
		return "  [======]\n  |🕶 🕶|"
	case "Laser Goggles":
		return "  [🔴-🔴] "
	case "Mini Cape":
		return "  ( o_o) \n  /\\___/\\"
	default:
		return ""
	}
}

// GetDialogue returns a dynamic, character-fitting dialogue line
func (p *PetState) GetDialogue() string {
	if p.TimerRunning && p.TimerMode == "timer" {
		minute := p.SessionElapsed / 60
		second := p.SessionElapsed % 60

		// Show custom interactive focus interventions for the first 30 seconds of key minutes!
		if second < 30 {
			switch minute {
			case 5:
				return "*brings you a virtual hot tea* 🍵 Sip some tea, human, and keep coding!"
			case 10:
				return "*purrs softly* Focus frequencies fully synchronized. You're doing amazing! 🎧"
			case 15:
				return "*stretches paws* We've been typing for 15 minutes. Remember to sit up straight! 🐾"
			case 20:
				return "*adjusts cyber-visor* My sensors detect intense productivity! Finish strong! ⚡️"
			case 25:
				return "*wiggles tail* Almost at the 25-minute mark! Let's push this code! 💻"
			case 30:
				return "*curls up on your terminal* 30 minutes of solid work! Neko is so proud of you! ❤️"
			case 40:
				return "*does a happy flip* 40 minutes! You are a master programmer! 🏆"
			case 50:
				return "*eyes glowing green* 50 minutes of deep focus! A legendary session! 🚀"
			}
		}
	}

	var quotes []string

	switch p.Mood {
	case "working":
		quotes = []string{
			"Intense focus meow! Let's code!",
			"I'm keeping an eye on your syntax, purr.",
			"No distractions, human! Purr-fect work ahead.",
			"My sensors detect heavy productivity...",
			"*wiggles ears* You're in the flow zone!",
			"*blinks green lens* Refactoring looks optimal.",
		}
	case "happy":
		quotes = []string{
			"Purrr... That session was delicious!",
			"You did so well! Meow! Hugs!",
			"Mrow! Break time! Time to stretch!",
			"I am so proud of you! *purrs*",
			"*chases a virtual cursor* Yay, a commit!",
			"*does a backflip* Level up in concentration!",
		}
	case "sleeping":
		quotes = []string{
			"Zzz... soft paws coding... zzz...",
			"Mew... just 5 more minutes of sleep...",
			"Zzz... dreaming of clean pull requests... zzz...",
			"*twitching whiskers* Zzz... compiling... zzz...",
		}
	case "grumpy":
		quotes = []string{
			"Hiss! Why did we stop coding?",
			"My binary bowl is empty. Focus please!",
			"Mew... Don't ignore me for too long...",
			"Hey! Tap those keys, I'm waiting!",
			"*scratching terminal* Hmph, productivity lost...",
		}
	default: // "idle"
		quotes = []string{
			"Mrow? Ready to start a Pomodoro?",
			"Let's build a streak, human!",
			"Purring softly... standard focus routines loaded.",
			"Which file are we refactoring today?",
			"*wiggles tail* Ready to push to production?",
			"*yawns* Did you remember to handle that err != nil?",
			"*stretches claws* I'm watching your import statements...",
			"*chases a bug* Is that a compilation error I see?",
		}
	}

	// Pick a random quote
	if len(quotes) == 0 {
		return "Let's focus!"
	}
	// Use standard deterministic mapping to prevent dialogue bubble flickering every animation frame.
	// Changing every 15 seconds makes the pet feel much more alive and real-time!
	timeHash := int(time.Now().Unix() / 15)
	return quotes[timeHash%len(quotes)]
}

// ---------------------------------------------------------
// REAL-TIME ANIMATED ASCII TEMPLATES FOR NEKO THE ROBO-KITTY
// ---------------------------------------------------------

func getKittyASCII(stage int, mood string, frame int) string {
	switch stage {
	case 2: // Adolescent (Teen)
		switch mood {
		case "working":
			if frame == 0 {
				return "   /\\_/\n  ( ⌐.⌐ ) ~\n  / >💻<\\\n (_______)"
			}
			return "   /\\_/\n  ( ⌐.⌐ ) ~ ⚡️\n  / >⌨️<\\\n (_______)"
		case "sleeping":
			if frame == 0 {
				return "   /\\_/\n  ( -.- ) z~\n  / > < \\ Z\n (_______)"
			}
			return "   /\\_/\n  ( -.- ) ~\n  / > < \\ zZ\n (_______)"
		case "happy":
			if frame == 0 {
				return "   /\\_/\n  ( ^.^ ) ~ *\n  / >♥< \\\n (_______)"
			}
			return "   /\\_/\n  ( ^.^ ) ~ ✨\n  / >♥< \\\n (_______)"
		case "grumpy":
			if frame == 0 {
				return "   /\\_/\n  ( ಠ_ಠ ) ~\n  / >🗙< \\\n (_______)"
			}
			return "   /\\_/\n  ( ಠ_ಠ ) _~\n  / >🗙< \\\n (_______)"
		default: // "idle"
			if frame == 0 {
				return "   /\\_/\n  ( o.o ) ~\n  / > < \\\n (_______)"
			}
			return "   /\\_/\n  ( -.- ) ~\n  / > < \\\n (_______)" // Blinking frame
		}

	case 3: // Cyber-Ascended God
		switch mood {
		case "working":
			if frame == 0 {
				return "     /\\_____/\\\n   [(= ⌐ . ⌐ =)]\n    /  \\ ⚡️ /  \\\n   (  [ 💻 ]  )\n    \\________/"
			}
			return "     /\\_____/\\\n   [(= ⌐ . ⌐ =)]\n    /  \\ ✨ /  \\\n   (  [ ⚡️ ]  )\n    \\________/"
		case "sleeping":
			if frame == 0 {
				return "     /\\_____/\\\n   [(= - . - =)] z\n    /  \\   /  \\ Z\n   (  [ Z ]  )\n    \\________/"
			}
			return "     /\\_____/\\\n   [(= - . - =)] Z\n    /  \\   /  \\ z\n   (  [ z ]  )\n    \\________/"
		case "happy":
			if frame == 0 {
				return "     /\\_____/\\\n   [(= ^ . ^ =)] *\n    /  \\ ✨ /  \\\n   (  [ ♥ ]  )\n    \\________/"
			}
			return "     /\\_____/\\\n   [(= ^ . ^ =)] ✨\n    /  \\ ♥ /  \\ *\n   (  [ ✨ ]  )\n    \\________/"
		case "grumpy":
			if frame == 0 {
				return "     /\\_____/\\\n   [(= ಠ . ಠ =)] ⚠\n    /  \\ 🗙 /  \\\n   (  [ 🗙 ]  )\n    \\________/"
			}
			return "     /\\_____/\\\n   [(= ಠ . ಠ =)] 🚨\n    /  \\ 🗙 /  \\\n   (  [ 🗙 ]  )\n    \\________/"
		default: // "idle"
			if frame == 0 {
				return "     /\\_____/\\\n   [(= o . o =)]\n    /  \\ ⚙️ /  \\\n   (  [ ⚙️ ]  )\n    \\________/"
			}
			return "     /\\_____/\\\n   [(= - . - =)]\n    /  \\ ⚙️ /  \\\n   (  [ ⚙️ ]  )\n    \\________/" // Blinking frame
		}

	default: // Stage 1 (Baby)
		switch mood {
		case "working":
			if frame == 0 {
				return "   /\\_/\n  ( ⌐.⌐ )\n  =[💻]="
			}
			return "   /\\_/\n  ( ⌐.⌐ )\n  =[⌨️]="
		case "sleeping":
			if frame == 0 {
				return "   /\\_/\n  ( -.- )z\n  =~  ~= Z"
			}
			return "   /\\_/\n  ( -.- )\n  =~  ~=zZ"
		case "happy":
			if frame == 0 {
				return "   /\\_/\n  ( >.< )\n  =~*~*~="
			}
			return "   /\\_/\n  ( ^.^ )\n  =~*~*~="
		case "grumpy":
			if frame == 0 {
				return "   /\\_/\n  ( ಠ_ಠ )\n  =~🗙~🗙="
			}
			return "   /\\_/\n  ( ಠ_ಠ )\n  =~-~-~=" // wiggles tail
		default: // "idle"
			if frame == 0 {
				return "   /\\_/\n  ( o.o )\n  =~`~`~="
			}
			return "   /\\_/\n  ( -.- )\n  =~`~`~=" // Blinking frame
		}
	}
}

// centerLines pads each line of a multi-line string to center it inside a width block
func centerLines(text string, width int) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth < width {
			padding := (width - lineWidth) / 2
			lines[i] = strings.Repeat(" ", padding) + line
		}
	}
	return strings.Join(lines, "\n")
}

// RenderPetBox draws the pet with its name, level, progress bar, and dialogue bubble
func RenderPetBox(pet PetState, width int) string {
	// Dynamically calculate the active frame (cycles every 750ms between 0 and 1 using high-resolution millisecond clocks)
	frame := int((time.Now().UnixNano() / int64(time.Millisecond)) / 750) % 2

	ascii := pet.GetASCII(frame)
	dialogue := pet.GetDialogue()

	dialogWidth := 26
	boxWidth := dialogWidth + 4 // Total width of dialogue box including border padding

	// Format dialogue bubble (wrapped)
	wrappedDialogue := wrapText(dialogue, dialogWidth)
	bubbleBorderTop := "╭" + strings.Repeat("─", dialogWidth+2) + "╮"
	bubbleBorderBot := "╰" + strings.Repeat("─", dialogWidth+2) + "╯"

	var bubbleLines []string
	bubbleLines = append(bubbleLines, bubbleBorderTop)
	for _, line := range wrappedDialogue {
		padding := dialogWidth - len(line)
		bubbleLines = append(bubbleLines, fmt.Sprintf("│ %s%s │", line, strings.Repeat(" ", padding)))
	}
	bubbleLines = append(bubbleLines, bubbleBorderBot)
	bubbleLines = append(bubbleLines, "  ╰─v") // Clean, sleek speech bubble tail pointing downwards

	bubbleBlock := strings.Join(bubbleLines, "\n")

	// Format pet info banner
	barWidth := 10
	stage := pet.EvolutionStage()
	stageName := "Baby"
	if stage == 2 {
		stageName = "Teen"
	} else if stage == 3 {
		stageName = "Cyber-Ascended"
	}

	xpTarget := pet.Level * 250
	pct := float64(pet.Experience) / float64(xpTarget)
	filled := int(pct * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled
	xpBar := fmt.Sprintf("[%s%s]", strings.Repeat("█", filled), strings.Repeat("░", empty))

	statsBlock := fmt.Sprintf(
		"🐾 %s (Lv.%d)\n%s %d%%\nEvolution: %s\nMood: %s\nFed: 🍖 %d",
		pet.Name,
		pet.Level,
		xpBar,
		int(pct*100),
		stageName,
		cases.Title(language.English).String(pet.Mood),
		pet.TotalSessionsFed,
	)

	// Center-align the animated ASCII art and stats under the speech bubble!
	centeredASCII := centerLines(ascii, boxWidth)
	centeredStats := centerLines(statsBlock, boxWidth)

	// Render layout vertical stack
	finalRender := fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		bubbleBlock,
		centeredASCII,
		centeredStats,
	)

	return finalRender
}

// Simple text wrapping helper
func wrapText(text string, limit int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) > limit {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine += " " + word
		}
	}
	lines = append(lines, currentLine)
	return lines
}

// RenderTamagotchiScreen draws the interactive console screen of the companion
func RenderTamagotchiScreen(pet PetState, width int, activeMenu string, menuSelection int, feedbackMsg string, accentColor string, primaryColor string) string {
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(accentColor))
	primaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(primaryColor))
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // Red
	goldStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // Gold

	// Build the pet display section
	frame := int((time.Now().UnixNano() / int64(time.Millisecond)) / 500) % 2
	petASCII := pet.GetASCII(frame)

	if pet.IsDead {
		// Ghost / Tombstone ASCII
		petASCII = "     .-.     \n    (q.p)    \n  _  \\\"/  _  \n ( \\_/^\\_/ ) \n  \\_______/  \n    R.I.P.   "
	} else if pet.IsSleeping {
		// Sleep state
		petASCII = strings.ReplaceAll(petASCII, "o.o", "-.-")
		petASCII = strings.ReplaceAll(petASCII, "^.^", "-.-")
	}

	// Dynamic dialogue bubble or feedback message
	dialogueText := feedbackMsg
	if dialogueText == "" {
		dialogueText = pet.GetDialogue()
	}
	if pet.IsDead {
		dialogueText = "G H O S T L Y  S I L E N C E..."
	}

	// 1. STATS INNER BARS
	healthBar := renderSmallBar(pet.Health, 10, primaryColor)
	hungerBar := renderSmallBar(pet.Hunger, 10, primaryColor)
	happyBar := renderSmallBar(pet.Happiness, 10, primaryColor)
	cleanBar := renderSmallBar(pet.Cleanliness, 10, primaryColor)
	energyBar := renderSmallBar(pet.Energy, 10, primaryColor)

	// Status lines
	var statsLeft string
	if pet.IsDead {
		statsLeft = warningStyle.Render("  [ STATUS: DECEASED 🪦 ]\n  Run rebirth to reset.")
	} else {
		statsLeft = fmt.Sprintf(
			"  ❤️ HEALTH: %s %d%%\n  🍖 HUNGER: %s %d%%\n  💖 HAPPY:  %s %d%%\n  🧹 CLEAN:  %s %d%%\n  ⚡ ENERGY: %s %d%%",
			healthBar, pet.Health,
			hungerBar, pet.Hunger,
			happyBar, pet.Happiness,
			cleanBar, pet.Cleanliness,
			energyBar, pet.Energy,
		)
	}

	// XP & Coins Display
	xpTarget := pet.Level * 250
	pctXP := float64(pet.Experience) / float64(xpTarget)
	xpBar := renderSmallBar(int(pctXP*100), 10, accentColor)
	
	statusRight := fmt.Sprintf(
		"  🐾 Name: %s\n  ⭐ Level: %d\n  🧬 Evolution: %s\n  📈 XP: %s %d%%\n  🪙 Coins: %s",
		goldStyle.Bold(true).Render(pet.Name),
		pet.Level,
		accentStyle.Render(getEvolutionStageName(pet.Level)),
		xpBar, int(pctXP*100),
		goldStyle.Bold(true).Render(fmt.Sprintf("%d", pet.Coins)),
	)
	
	if pet.IsSick {
		statusRight += "\n  " + warningStyle.Bold(true).Blink(true).Render("⚠️ SICK (anti-virus needed!)")
	} else if pet.Poops > 0 {
		statusRight += "\n  " + warningStyle.Render(fmt.Sprintf("💩 POOPS ON SCREEN: %d", pet.Poops))
	} else {
		statusRight += "\n  " + primaryStyle.Render("🟢 State: Healthy & clean")
	}

	// 2. MAIN INNER DISPLAY CONTAINER
	innerWidth := 58
	lcdBorderTop := "┌" + strings.Repeat("─", innerWidth) + "┐"
	lcdBorderBot := "└" + strings.Repeat("─", innerWidth) + "┘"

	// Format dialogue block
	wrappedDiag := wrapText(dialogueText, innerWidth-4)
	var diagLines []string
	for _, l := range wrappedDiag {
		diagLines = append(diagLines, fmt.Sprintf("│  %s%s  │", accentStyle.Italic(true).Render(l), strings.Repeat(" ", innerWidth-4-lipgloss.Width(l))))
	}
	diagBlock := strings.Join(diagLines, "\n")

	// Render Neko and poops inside LCD
	petLines := strings.Split(petASCII, "\n")
	var petRenderedLines []string
	for i, pl := range petLines {
		// Draw poop on right side if there are poops!
		poopStr := ""
		if !pet.IsDead {
			if pet.Poops > 0 && i == len(petLines)-1 {
				poopStr = "  💩"
			}
			if pet.Poops > 1 && i == len(petLines)-2 {
				poopStr = "  💩"
			}
			if pet.Poops > 2 && i == len(petLines)-3 {
				poopStr = "        💩"
			}
		}
		
		padLeft := (innerWidth - lipgloss.Width(pl) - lipgloss.Width(poopStr)) / 2
		padRight := innerWidth - lipgloss.Width(pl) - lipgloss.Width(poopStr) - padLeft
		petRenderedLines = append(petRenderedLines, fmt.Sprintf("│%s%s%s%s│", strings.Repeat(" ", padLeft), pl, poopStr, strings.Repeat(" ", padRight)))
	}
	petBlock := strings.Join(petRenderedLines, "\n")

	// Assemble LCD Content
	var lcdContent []string
	lcdContent = append(lcdContent, lcdBorderTop)
	lcdContent = append(lcdContent, fmt.Sprintf("│  %-*s│", innerWidth-2, primaryStyle.Bold(true).Render("C Y B E R - P E T   S C R E E N")))
	lcdContent = append(lcdContent, "├"+strings.Repeat("─", innerWidth)+"┤")
	lcdContent = append(lcdContent, petBlock)
	lcdContent = append(lcdContent, "├"+strings.Repeat("─", innerWidth)+"┤")
	lcdContent = append(lcdContent, diagBlock)
	lcdContent = append(lcdContent, lcdBorderBot)
	lcdBlock := strings.Join(lcdContent, "\n")

	// 3. MENU / ACTION PANELS
	var actionBlock string
	switch activeMenu {
	case "feed":
		options := []string{
			fmt.Sprintf("1. Feed Fish Kibble (Restores +30 Hunger, +5 Happiness) - Qty: %d", pet.Inventory["fish"]),
			fmt.Sprintf("2. Feed Cyber Cookie (Restores +20 Hunger, +30 Happiness, +10 Energy) - Qty: %d", pet.Inventory["treat"]),
			fmt.Sprintf("3. Feed Cyber Cola (Restores +40 Energy, +10 Happiness, -10 Cleanliness) - Qty: %d", pet.Inventory["drink"]),
		}
		actionBlock = renderSubMenu("F E E D I N G   I N V E N T O R Y 🍖", options, menuSelection, innerWidth, primaryColor)
	case "shop":
		options := []string{
			"1. Buy Fish Kibble (Cost: 5 coins)",
			"2. Buy Cyber Cookie (Cost: 10 coins)",
			"3. Buy Cyber Cola (Cost: 8 coins)",
			"4. Buy Anti-Virus Medicine (Cost: 15 coins)",
		}
		actionBlock = renderSubMenu("C Y B E R   S H O P P I N G 🪙", options, menuSelection, innerWidth, primaryColor)
	case "play":
		options := []string{
			"1. Play Pomo-Type Challenge (Typing Speed Game)",
			"2. Play Binary Guessing Game (Higher/Lower Math Game)",
		}
		actionBlock = renderSubMenu("M I N I - G A M E S   L O B B Y 🎮", options, menuSelection, innerWidth, primaryColor)
	default:
		// Standard main operations menu selection
		options := []string{
			"FEED companion 🍖",
			"PLAY mini-game 🎮",
			"CLEAN poops 🧹",
			"HEAL sickness 💊",
			"SLEEP lights toggle ⚡",
			"SHOP item store 🪙",
			"REBIRTH reset 🧬",
			"EXIT console 🚪",
		}
		actionBlock = renderMainButtonsGrid(options, menuSelection, innerWidth, primaryColor)
	}

	// 4. ASSEMBLE ENTIRE HANDHELD CASING
	shellBorderTop := accentStyle.Render("    .----------------------------------------------------------.")
	shellHeader    := accentStyle.Render("   /                C Y B E R - P E T   1 9 9 6                 \\")
	shellDivider   := accentStyle.Render("  |    .----------------------------------------------------.    |")
	shellBorderBot := accentStyle.Render("   \\                   [A]       [B]       [C]                 /")
	shellFooter    := accentStyle.Render("    '----------------------------------------------------------'")

	var finalRows []string
	finalRows = append(finalRows, shellBorderTop)
	finalRows = append(finalRows, shellHeader)
	finalRows = append(finalRows, shellDivider)

	// Embed stats left and right side-by-side!
	statsLinesLeft := strings.Split(statsLeft, "\n")
	statsLinesRight := strings.Split(statusRight, "\n")
	for i := 0; i < 5; i++ {
		sl := ""
		sr := ""
		if i < len(statsLinesLeft) {
			sl = statsLinesLeft[i]
		}
		if i < len(statsLinesRight) {
			sr = statsLinesRight[i]
		}
		rowContent := fmt.Sprintf("  |   %-*s%-*s   |", 34, sl, 32, sr)
		finalRows = append(finalRows, rowContent)
	}
	
	finalRows = append(finalRows, shellDivider)

	// Embed the LCD screen
	lcdLines := strings.Split(lcdBlock, "\n")
	for _, l := range lcdLines {
		finalRows = append(finalRows, fmt.Sprintf("  |    %s    |", l))
	}

	finalRows = append(finalRows, shellDivider)

	// Embed the action button or sub-menus
	actLines := strings.Split(actionBlock, "\n")
	for _, l := range actLines {
		finalRows = append(finalRows, fmt.Sprintf("  |    %s    |", l))
	}

	finalRows = append(finalRows, shellDivider)
	finalRows = append(finalRows, shellBorderBot)
	finalRows = append(finalRows, shellFooter)

	// Return centered final casing
	return centerBlock(width, strings.Join(finalRows, "\n"))
}

func getEvolutionStageName(level int) string {
	if level >= 8 {
		return "Cyber-Ascended God 🐲"
	}
	if level >= 4 {
		return "Robo-Teenager 🐆"
	}
	return "Baby Droid 🐾"
}

func renderSmallBar(value int, width int, color string) string {
	filled := int(float64(value) / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled
	filledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray
	return fmt.Sprintf("[%s%s]", filledStyle.Render(strings.Repeat("█", filled)), emptyStyle.Render(strings.Repeat("░", empty)))
}

func renderSubMenu(title string, options []string, selected int, width int, color string) string {
	cStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	subtle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	var rows []string
	rows = append(rows, cStyle.Bold(true).Render("   "+title))
	rows = append(rows, subtle.Render("   "+strings.Repeat("─", width-6)))

	for i, opt := range options {
		if i == selected {
			rows = append(rows, accentStyle.Bold(true).Render(fmt.Sprintf("   ⚡ %s", opt)))
		} else {
			rows = append(rows, subtle.Render(fmt.Sprintf("     %s", opt)))
		}
	}
	// Pad menu to uniform height of 6 lines
	for len(rows) < 6 {
		rows = append(rows, "")
	}
	
	// Add return hint
	rows = append(rows, subtle.Render("   [↑/↓] Select  [Enter] Confirm  [Esc] Back"))
	return strings.Join(rows, "\n")
}

func renderMainButtonsGrid(options []string, selected int, width int, color string) string {
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	subtle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))

	var rows []string
	rows = append(rows, activeStyle.Bold(true).Render("   M A I N   O P E R A T I O N S   M E N U"))
	rows = append(rows, subtle.Render("   "+strings.Repeat("─", width-6)))

	// Draw 4 rows of 2 options side-by-side
	for r := 0; r < 4; r++ {
		idx1 := r * 2
		idx2 := r * 2 + 1

		opt1 := options[idx1]
		opt2 := options[idx2]

		var text1, text2 string
		if idx1 == selected {
			text1 = accentStyle.Bold(true).Render(fmt.Sprintf(" ▶ %s", opt1))
		} else {
			text1 = subtle.Render(fmt.Sprintf("   %s", opt1))
		}

		if idx2 == selected {
			text2 = accentStyle.Bold(true).Render(fmt.Sprintf(" ▶ %s", opt2))
		} else {
			text2 = subtle.Render(fmt.Sprintf("   %s", opt2))
		}

		rows = append(rows, fmt.Sprintf(" %-*s %-s", 34, text1, text2))
	}
	
	rows = append(rows, subtle.Render("   [↑/↓/←/→] Navigate Grid  [Enter] Press Button  [Esc] Exit"))
	return strings.Join(rows, "\n")
}

func centerBlock(width int, content string) string {
	if width <= 0 {
		return content
	}
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(content)
}

