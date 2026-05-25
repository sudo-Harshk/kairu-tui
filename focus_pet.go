package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// PetState represents the persistent status of the companion
type PetState struct {
	Name             string    `json:"name"`
	Type             string    `json:"type"`                // Always "kitty" (Neko)
	Level            int       `json:"level"`               // 1 to 10
	Experience       int       `json:"experience"`          // Current XP in this level
	TotalSessionsFed int       `json:"total_sessions_fed"`  // Completed Pomodoros
	Mood             string    `json:"mood"`                // "idle", "working", "happy", "sleeping", "grumpy"
	LastFedTime      time.Time `json:"last_fed_time"`       // Track to detect inactivity/neglect
	ActiveItem       string    `json:"active_item"`         // Discovered cosmetic items (e.g., "Wizard Hat", "Cyber Visor")
}

// DefaultPet creates a basic starting pet (Neko the Robo-Kitty)
func DefaultPet(name string, petType string) PetState {
	if name == "" {
		name = "Neko"
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
	var quotes []string

	switch p.Mood {
	case "working":
		quotes = []string{
			"Intense focus meow! Let's code!",
			"I'm keeping an eye on your syntax, purr.",
			"No distractions, human! Purr-fect work ahead.",
			"My sensors detect heavy productivity...",
		}
	case "happy":
		quotes = []string{
			"Purrr... That session was delicious!",
			"You did so well! Meow! Hugs!",
			"Mrow! Break time! Time to stretch!",
			"I am so proud of you! *purrs*",
		}
	case "sleeping":
		quotes = []string{
			"Zzz... soft paws coding... zzz...",
			"Mew... just 5 more minutes of sleep...",
			"Zzz... dreaming of clean pull requests... zzz...",
		}
	case "grumpy":
		quotes = []string{
			"Hiss! Why did we stop coding?",
			"My binary bowl is empty. Focus please!",
			"Mew... Don't ignore me for too long...",
			"Hey! Tap those keys, I'm waiting!",
		}
	default: // "idle"
		quotes = []string{
			"Mrow? Ready to start a Pomodoro?",
			"Let's build a streak, human!",
			"Purring softly... standard focus routines loaded.",
			"Which file are we refactoring today?",
		}
	}

	// Pick a random quote
	if len(quotes) == 0 {
		return "Let's focus!"
	}
	// Use standard deterministic mapping to prevent dialogue bubble flickering every animation frame
	hourHash := time.Now().Hour() + time.Now().Minute()
	return quotes[hourHash%len(quotes)]
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
		strings.Title(pet.Mood),
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
