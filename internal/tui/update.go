package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"kairu-tui/internal/backup"
	"kairu-tui/internal/config"
	"kairu-tui/internal/entries"
	"kairu-tui/internal/kairutype"
	"kairu-tui/internal/notification"
	"kairu-tui/internal/pet"
	"kairu-tui/internal/soundscape"
	"kairu-tui/internal/streak"
	"kairu-tui/internal/tasks"
	"kairu-tui/internal/templates"
	"kairu-tui/internal/timer"
)

func (m model) activeSessionMode() string {
	switch m.mode {
	case "timer", "break":
		return m.mode
	case "edit":
		return m.editReturnMode
	case "settings":
		if m.settingsReturnMode == "stats" {
			return m.statsReturnMode
		}
		return m.settingsReturnMode
	case "stats":
		return m.statsReturnMode
	case "analytics":
		return m.statsReturnMode
	case "heatmap":
		return m.statsReturnMode
	case "report":
		return m.statsReturnMode
	case "templates":
		return m.templateReturnMode
	case "soundscapes":
		return m.soundscapeReturnMode
	case "logs":
		return m.statsReturnMode
	case "help":
		switch m.helpReturnMode {
		case "settings":
			if m.settingsReturnMode == "stats" {
				return m.statsReturnMode
			}
			return m.settingsReturnMode
		case "stats":
			return m.statsReturnMode
		case "heatmap":
			return m.statsReturnMode
		default:
			return m.helpReturnMode
		}
	default:
		return ""
	}
}

func (m model) returnModeForModal() string {
	if mode := m.activeSessionMode(); mode != "" {
		return mode
	}
	if m.mode == "stats" {
		return m.statsReturnMode
	}
	if m.mode == "analytics" {
		return m.statsReturnMode
	}
	if m.mode == "heatmap" {
		return m.statsReturnMode
	}
	if m.mode == "report" {
		return m.statsReturnMode
	}
	return m.mode
}

func (m *model) saveOnQuit() {
	m.stopSoundscape()
	sessionMode := m.activeSessionMode()
	if (sessionMode != "timer" && sessionMode != "break") || m.seconds <= 0 {
		return
	}
	previous := m.mode
	m.mode = sessionMode
	m.saveSession()
	m.mode = previous
}

type clearNotificationMsg struct {
	id int
}

func clearNotificationCmd(id int) tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return clearNotificationMsg{id: id}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if clearMsg, ok := msg.(clearNotificationMsg); ok {
		if clearMsg.id == m.notificationCounter {
			m.notificationStatus = ""
		}
		return m, nil
	}

	oldNotificationCounter := m.notificationCounter
	newModelInterface, cmd := m.updateInner(msg)

	newModel, ok := newModelInterface.(model)
	if !ok {
		return newModelInterface, cmd
	}

	if newModel.notificationCounter != oldNotificationCounter && newModel.notificationStatus != "" {
		clearCmd := clearNotificationCmd(newModel.notificationCounter)
		return newModel, tea.Batch(cmd, clearCmd)
	}

	return newModel, cmd
}

func (m model) updateInner(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tickTockMsg:
		sessionMode := m.activeSessionMode()
		if m.running && (sessionMode == "timer" || sessionMode == "break") {
			if m.seconds > 0 {
				m.seconds--
				m.sessionElapsed++
				if sessionMode == "timer" {
					m.totalWorkTime++
				} else {
					m.totalBreakTime++
				}
			}
			if m.seconds == 0 {
				if m.mode != sessionMode {
					m.mode = sessionMode
				}
				var notifyC tea.Cmd
				if sessionMode == "timer" {
					notifyC = m.notifyCmd("work_complete")
				} else {
					notifyC = m.notifyCmd("break_complete")
				}
				model, cmd := m.completeSession()
				return model, tea.Batch(notifyC, cmd)
			}
		}

		if m.mode == "kairu_type" {
			if m.kairuType.Active && !m.kairuType.Finished {
				if m.kairuType.Mode == "time" {
					if m.kairuType.TimeRemaining > 0 {
						m.kairuType.TimeRemaining--
						elapsedSecs := m.kairuType.TargetTime - m.kairuType.TimeRemaining
						m.kairuType.SampleWPM(elapsedSecs)
					}
					if m.kairuType.TimeRemaining <= 0 {
						m.kairuType.FinishTest()
						// Award coins during break!
						if sessionMode == "break" && m.running && m.kairuType.WPM >= 40 {
							coinsGained := (m.kairuType.WPM - 40) / 10
							if coinsGained < 2 {
								coinsGained = 2
							}
							if coinsGained > 15 {
								coinsGained = 15
							}
							m.petState.Coins += coinsGained
							m.logInternal("TYPING: Break typing test complete. Earned %d bonus Pomo-Coins! (WPM: %d)", coinsGained, m.kairuType.WPM)
							_ = pet.SavePetState(m.petFile, m.petState)
						}
					}
				} else {
					elapsedSecs := int(time.Since(m.kairuType.StartTime).Seconds())
					m.kairuType.SampleWPM(elapsedSecs)
				}
			}
		}

		if m.mode == "tamagotchi" {
			m.petState.TickStateDecay(time.Now())
			return m, tickCmd()
		}

		if m.mode == "timer" || m.mode == "break" || m.mode == "settings" || m.mode == "stats" || m.mode == "analytics" || m.mode == "heatmap" || m.mode == "history" || m.mode == "report" || m.mode == "help" || m.mode == "logs" || m.mode == "templates" || m.mode == "soundscapes" || m.mode == "tamagotchi" || m.mode == "kairu_type" {
			return m, tickCmd()
		}
		return m, nil

	case petAnimTickMsg:
		if m.petEnabled && (m.mode == "tamagotchi" || (m.showPetSidebar && m.width >= 90)) {
			m.petState.TickStateDecay(time.Now())
			return m, petAnimTick()
		}
		return m, nil

	case notifResultMsg:
		if msg.err != nil {
			m.setAppError(msg.err, "Notification failed")
		} else {
			if msg.status != "" {
				m.setNotificationStatus(msg.status)
			}
			if msg.id != "" {
				if m.deliveredNotifyIDs == nil {
					m.deliveredNotifyIDs = make(map[string]time.Time)
				}
				m.deliveredNotifyIDs[msg.id] = time.Now()
				m.logInternal("NOTIF: Delivered %s", msg.id)
			}
		}
		return m, nil

	case outboxFlushedMsg:
		m.notificationOutbox = msg.remaining
		if len(msg.deliveredIDs) > 0 {
			if m.deliveredNotifyIDs == nil {
				m.deliveredNotifyIDs = make(map[string]time.Time)
			}
			now := time.Now()
			for _, id := range msg.deliveredIDs {
				m.deliveredNotifyIDs[id] = now
				m.logInternal("NOTIF: Flushed %s", id)
			}
		}
		if msg.err != nil {
			m.setAppError(msg.err, "Failed to save notification queue")
		} else if msg.status != "" {
			m.setNotificationStatus(msg.status)
		}
		if len(msg.remaining) > 0 {
			m.setAppError(fmt.Errorf("%s", msg.remaining[0].LastError), "Notification queued for retry")
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		if m.petEnabled && m.showPetSidebar && m.width >= 90 {
			return m, petAnimTick()
		}
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		if m.showPetLevelUpOverlay {
			m.showPetLevelUpOverlay = false
			return m, nil
		}

		if key == "ctrl+g" && m.mode != "fatal" {
			if m.petEnabled {
				m.showPetSidebar = !m.showPetSidebar
				if m.showPetSidebar && m.width >= 90 {
					return m, petAnimTick()
				}
				return m, nil
			}
		}

		if key == "ctrl+t" && m.mode != "fatal" {
			if m.petEnabled {
				if m.mode == "tamagotchi" {
					m.mode = m.tamagotchiReturnMode
					if m.mode == "" {
						m.mode = "input"
					}
				} else {
					m.tamagotchiReturnMode = m.mode
					m.mode = "tamagotchi"
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					m.tamagotchiFeedback = ""
					m.petState.TickStateDecay(time.Now())
					_ = pet.SavePetState(m.petFile, m.petState)
				}
				return m, tea.Batch(tickCmd(), petAnimTick())
			}
		}

		if key == "ctrl+y" && m.mode != "fatal" {
			if m.mode == "kairu_type" {
				m.mode = m.statsReturnMode
				if m.mode == "" || m.mode == "kairu_type" {
					m.mode = "input"
				}
			} else {
				m.statsReturnMode = m.mode
				m.mode = "kairu_type"
				m.kairuType.ResetTest()
			}
			return m, tickCmd()
		}

		if m.mode == "fatal" {
			switch key {
			case "ctrl+c", "q", "enter", "esc":
				return m, tea.Quit
			}
			return m, nil
		}

		if m.mode == "history" {
			if m.historyFilter.searchFocused {
				if key == "ctrl+c" {
					m.saveOnQuit()
					return m, tea.Quit
				}
				if key == "esc" {
					m.historyFilter.searchInput.SetValue("")
					m.historyFilter.searchInput.Blur()
					m.historyFilter.searchFocused = false
					m.historyFilter.typeFilter = "all"
					m.historyFilter.dateRange = "all"
					return m, nil
				}
				if key == "enter" {
					m.historyFilter.searchInput.Blur()
					m.historyFilter.searchFocused = false
					return m, nil
				}
				if key == "/" {
					m.historyFilter.searchInput.Blur()
					m.historyFilter.searchFocused = false
					return m, nil
				}
				var textCmd tea.Cmd
				m.historyFilter.searchInput, textCmd = m.historyFilter.searchInput.Update(msg)
				return m, textCmd
			}

			switch key {
			case "/":
				m.historyFilter.searchInput.Focus()
				m.historyFilter.searchFocused = true
				return m, nil
			case "t":
				switch m.historyFilter.typeFilter {
				case "all":
					m.historyFilter.typeFilter = "work"
				case "work":
					m.historyFilter.typeFilter = "break"
				default:
					m.historyFilter.typeFilter = "all"
				}
				return m, nil
			case "d":
				switch m.historyFilter.dateRange {
				case "all":
					m.historyFilter.dateRange = "today"
				case "today":
					m.historyFilter.dateRange = "week"
				case "week":
					m.historyFilter.dateRange = "month"
				default:
					m.historyFilter.dateRange = "all"
				}
				return m, nil
			case "esc":
				if m.historyFilter.searchInput.Value() != "" || m.historyFilter.typeFilter != "all" || m.historyFilter.dateRange != "all" {
					m.historyFilter.searchInput.SetValue("")
					m.historyFilter.typeFilter = "all"
					m.historyFilter.dateRange = "all"
					return m, nil
				}
				// Otherwise fall through to general esc handling to exit history view.
			}
		}

		if m.mode == "kairu_type" {
			sessionMode := m.activeSessionMode()
			if m.kairuType.Finished {
				switch key {
				case "enter":
					m.kairuType.ResetTest()
					return m, nil
				case "esc":
					m.mode = m.statsReturnMode
					if m.mode == "" || m.mode == "kairu_type" {
						m.mode = "input"
					}
					return m, nil
				case "tab":
					if m.kairuType.Mode == "time" {
						m.kairuType = kairutype.InitKairuType("words", 25)
					} else {
						m.kairuType = kairutype.InitKairuType("time", 30)
					}
					return m, nil
				case "ctrl+r":
					m.kairuType.ResetTest()
					return m, nil
				}
				return m, nil
			}

			switch key {
			case "esc":
				m.mode = m.statsReturnMode
				if m.mode == "" || m.mode == "kairu_type" {
					m.mode = "input"
				}
				return m, nil
			case "ctrl+r":
				m.kairuType.ResetTest()
				return m, nil
			case "tab":
				if !m.kairuType.Active {
					if m.kairuType.Mode == "time" {
						if m.kairuType.TargetTime == 15 {
							m.kairuType = kairutype.InitKairuType("time", 30)
						} else if m.kairuType.TargetTime == 30 {
							m.kairuType = kairutype.InitKairuType("time", 60)
						} else {
							m.kairuType = kairutype.InitKairuType("words", 10)
						}
					} else {
						if m.kairuType.TargetWords == 10 {
							m.kairuType = kairutype.InitKairuType("words", 25)
						} else if m.kairuType.TargetWords == 25 {
							m.kairuType = kairutype.InitKairuType("words", 50)
						} else {
							m.kairuType = kairutype.InitKairuType("time", 15)
						}
					}
				}
				return m, nil
			case "backspace":
				m.kairuType.HandleBackspace()
				return m, nil
			case "space":
				m.kairuType.AddChar(" ")
				return m, nil
			case "enter", "ctrl+m":
				return m, nil
			default:
				if len(key) == 1 {
					m.kairuType.AddChar(key)
					
					if m.kairuType.Finished && sessionMode == "break" && m.running && m.kairuType.WPM >= 40 {
						coinsGained := (m.kairuType.WPM - 40) / 10
						if coinsGained < 2 {
							coinsGained = 2
						}
						if coinsGained > 15 {
							coinsGained = 15
						}
						m.petState.Coins += coinsGained
						m.logInternal("TYPING: Break typing test complete. Earned %d bonus Pomo-Coins! (WPM: %d)", coinsGained, m.kairuType.WPM)
						_ = pet.SavePetState(m.petFile, m.petState)
					}
				}
				return m, nil
			}
		}

		if key == "?" {
			if m.mode == "help" {
				return m.closeHelp(true)
			}
			m = m.openHelp()
			return m, nil
		}

		if m.mode == "tamagotchi" {
			if !m.tamagotchiFeedbackTime.IsZero() && time.Since(m.tamagotchiFeedbackTime) > 3*time.Second {
				m.tamagotchiFeedback = ""
			}

			if m.tamagotchiActiveMenu == "rebirth" {
				switch key {
				case "enter":
					name := strings.TrimSpace(m.textInput.Value())
					if name == "" {
						name = "Neko"
					}
					m.tamagotchiFeedback = m.petState.RebirthPet(name)
					m.tamagotchiFeedbackTime = time.Now()
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					_ = pet.SavePetState(m.petFile, m.petState)
					return m, nil
				case "esc":
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					return m, nil
				default:
					m.textInput, cmd = m.textInput.Update(msg)
					return m, cmd
				}
			}

			if m.tamagotchiActiveMenu == "typing" {
				if m.typingGame.Finished {
					if key == "enter" {
						m.tamagotchiActiveMenu = ""
						m.tamagotchiMenuSelect = 0
						m.tamagotchiFeedback = ""
					}
					return m, nil
				}
				
				switch key {
				case "esc":
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					m.tamagotchiFeedback = ""
					return m, nil
				case "backspace":
					if len(m.typingGame.TypedText) > 0 {
						m.typingGame.TypedText = m.typingGame.TypedText[:len(m.typingGame.TypedText)-1]
					}
					return m, nil
				case "enter", "ctrl+m", "space", "tab":
					char := " "
					if key == "enter" || key == "ctrl+m" {
						char = "\n"
					} else if key == "tab" {
						char = "\t"
					}
					if len(m.typingGame.TypedText) < len(m.typingGame.TargetText) {
						m.typingGame.TypedText += char
					}
				default:
					if len(key) == 1 && len(m.typingGame.TypedText) < len(m.typingGame.TargetText) {
						m.typingGame.TypedText += key
					}
				}

				if len(m.typingGame.TypedText) >= len(m.typingGame.TargetText) {
					m.typingGame.Finished = true
					correctCount := 0
					for i := 0; i < len(m.typingGame.TargetText); i++ {
						if i < len(m.typingGame.TypedText) && m.typingGame.TargetText[i] == m.typingGame.TypedText[i] {
							correctCount++
						}
					}
					m.typingGame.Accuracy = (float64(correctCount) / float64(len(m.typingGame.TargetText))) * 100.0
					duration := time.Since(m.typingGame.StartTime).Minutes()
					if duration <= 0 {
						duration = 0.01
					}
					words := float64(len(m.typingGame.TargetText)) / 5.0
					m.typingGame.WPM = int(words / duration)
					if m.typingGame.WPM > 200 {
						m.typingGame.WPM = 200
					}

					coinsReward := 5
					if m.typingGame.Accuracy >= 90.0 {
						coinsReward = 10
						if m.typingGame.WPM > 60 {
							coinsReward = 15
						}
					}
					m.typingGame.CoinsWon = coinsReward
					m.petState.Coins += coinsReward
					m.petState.Happiness += 25
					if m.petState.Happiness > 100 {
						m.petState.Happiness = 100
					}
					_ = pet.SavePetState(m.petFile, m.petState)
				}
				return m, nil
			}

			if m.tamagotchiActiveMenu == "guessing" {
				if m.binaryGame.Finished {
					if key == "enter" {
						m.tamagotchiActiveMenu = ""
						m.tamagotchiMenuSelect = 0
						m.tamagotchiFeedback = ""
					}
					return m, nil
				}

				switch key {
				case "esc":
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					m.tamagotchiFeedback = ""
					return m, nil
				case "backspace":
					if len(m.binaryGame.InputStr) > 0 {
						m.binaryGame.InputStr = m.binaryGame.InputStr[:len(m.binaryGame.InputStr)-1]
					}
					return m, nil
				case "enter":
					var guess int
					_, err := fmt.Sscanf(m.binaryGame.InputStr, "%d", &guess)
					if err != nil {
						m.binaryGame.LastHint = "Invalid decimal number! Enter digits only."
						m.binaryGame.InputStr = ""
						return m, nil
					}

					m.binaryGame.Attempts++
					if guess == m.binaryGame.TargetNum {
						m.binaryGame.Won = true
						m.binaryGame.Finished = true
						m.petState.Coins += 5
						m.petState.Happiness += 15
						if m.petState.Happiness > 100 {
							m.petState.Happiness = 100
						}
						_ = pet.SavePetState(m.petFile, m.petState)
					} else if m.binaryGame.Attempts >= 4 {
						m.binaryGame.Finished = true
					} else if guess < m.binaryGame.TargetNum {
						m.binaryGame.LastHint = "Higher!"
						m.binaryGame.InputStr = ""
					} else {
						m.binaryGame.LastHint = "Lower!"
						m.binaryGame.InputStr = ""
					}
					return m, nil
				default:
					if len(key) == 1 && key >= "0" && key <= "9" && len(m.binaryGame.InputStr) < 3 {
						m.binaryGame.InputStr += key
					}
				}
				return m, nil
			}

			switch m.tamagotchiActiveMenu {
			case "feed":
				switch key {
				case "up", "k":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect - 1 + 3) % 3
					return m, nil
				case "down", "j":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect + 1) % 3
					return m, nil
				case "enter":
					items := []string{"fish", "treat", "drink"}
					item := items[m.tamagotchiMenuSelect]
					var leveledUp bool
					m.tamagotchiFeedback, leveledUp = m.petState.FeedItem(item)
					m.tamagotchiFeedbackTime = time.Now()
					if leveledUp {
						m.showPetLevelUpOverlay = true
					}
					_ = pet.SavePetState(m.petFile, m.petState)
					return m, nil
				case "esc":
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					m.tamagotchiFeedback = ""
					return m, nil
				}

			case "shop":
				switch key {
				case "up", "k":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect - 1 + 4) % 4
					return m, nil
				case "down", "j":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect + 1) % 4
					return m, nil
				case "enter":
					items := []string{"fish", "treat", "drink", "medicine"}
					costs := []int{5, 10, 8, 15}
					item := items[m.tamagotchiMenuSelect]
					cost := costs[m.tamagotchiMenuSelect]
					m.tamagotchiFeedback = m.petState.BuyItem(item, cost)
					m.tamagotchiFeedbackTime = time.Now()
					_ = pet.SavePetState(m.petFile, m.petState)
					return m, nil
				case "esc":
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					m.tamagotchiFeedback = ""
					return m, nil
				}

			case "play":
				switch key {
				case "up", "k":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect - 1 + 2) % 2
					return m, nil
				case "down", "j":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect + 1) % 2
					return m, nil
				case "enter":
					if m.tamagotchiMenuSelect == 0 {
						m.tamagotchiActiveMenu = "typing"
						m.typingGame = pet.InitTypingGame()
					} else {
						m.tamagotchiActiveMenu = "guessing"
						m.binaryGame = pet.InitBinaryGame()
					}
					return m, nil
				case "esc":
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					m.tamagotchiFeedback = ""
					return m, nil
				}

			default:
				switch key {
				case "up", "k":
					m.tamagotchiMenuSelect -= 2
					if m.tamagotchiMenuSelect < 0 {
						m.tamagotchiMenuSelect += 8
					}
					return m, nil
				case "down", "j":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect + 2) % 8
					return m, nil
				case "left", "h":
					if m.tamagotchiMenuSelect%2 == 1 {
						m.tamagotchiMenuSelect--
					} else {
						m.tamagotchiMenuSelect++
					}
					return m, nil
				case "right", "l":
					if m.tamagotchiMenuSelect%2 == 0 {
						m.tamagotchiMenuSelect++
					} else {
						m.tamagotchiMenuSelect--
					}
					return m, nil
				case "esc", "q":
					m.mode = m.tamagotchiReturnMode
					if m.mode == "" {
						m.mode = "input"
					}
					return m, nil
				case "enter":
					switch m.tamagotchiMenuSelect {
					case 0:
						m.tamagotchiActiveMenu = "feed"
						m.tamagotchiMenuSelect = 0
					case 1:
						m.tamagotchiActiveMenu = "play"
						m.tamagotchiMenuSelect = 0
					case 2:
						m.tamagotchiFeedback = m.petState.CleanPoop()
						m.tamagotchiFeedbackTime = time.Now()
						_ = pet.SavePetState(m.petFile, m.petState)
					case 3:
						m.tamagotchiFeedback = m.petState.HealSick()
						m.tamagotchiFeedbackTime = time.Now()
						_ = pet.SavePetState(m.petFile, m.petState)
					case 4:
						m.tamagotchiFeedback = m.petState.ToggleSleep()
						m.tamagotchiFeedbackTime = time.Now()
						_ = pet.SavePetState(m.petFile, m.petState)
					case 5:
						m.tamagotchiActiveMenu = "shop"
						m.tamagotchiMenuSelect = 0
					case 6:
						m.tamagotchiActiveMenu = "rebirth"
						m.textInput.SetValue("")
						m.textInput.Focus()
					case 7:
						m.mode = m.tamagotchiReturnMode
						if m.mode == "" {
							m.mode = "input"
						}
					}
					return m, nil
				}
			}
			return m, nil
		}

		if m.mode == "help" {
			if key == "esc" {
				return m.closeHelp(true)
			}
			if key == "ctrl+c" || key == "q" {
				m.saveOnQuit()
				return m, tea.Quit
			}
			return m, nil
		}

		if m.mode == "settings" {
			switch key {
			case "tab":
				if m.settingsCursor == settingsCount-1 && m.settingsReturnMode != "" {
					m.statsReturnMode = m.settingsReturnMode
					m.mode = "stats"
					return m, nil
				}
				m.settingsCursor = (m.settingsCursor + 1) % settingsCount
				return m, nil
			case "down", "j":
				m.settingsCursor = (m.settingsCursor + 1) % settingsCount
				return m, nil
			case "shift+tab", "up", "k":
				m.settingsCursor--
				if m.settingsCursor < 0 {
					m.settingsCursor = settingsCount - 1
				}
				return m, nil
			case "enter", " ", "space":
				if m.settingsCursor == settingsBackup {
					if err := backup.BackupProject(m.dataFile, m.templateFile, m.configFile, m.outboxFile, config.DefaultBackupFile()); err != nil {
						m.setAppError(err, "Failed to create backup")
					} else {
						m.setNotificationStatus("Backup saved to backup.json")
					}
					return m, nil
				}
				if m.settingsCursor == settingsRestore {
					if err := backup.RestoreProject(m.dataFile, m.templateFile, m.configFile, m.outboxFile, config.DefaultBackupFile()); err != nil {
						m.setAppError(err, "Failed to restore backup")
					} else {
						if err := reloadProjectState(&m); err != nil {
							m.setAppError(err, "Backup restored but reload failed")
						} else {
							m.setNotificationStatus("Backup restored from backup.json")
						}
					}
					return m, nil
				}
				if m.settingsCursor == settingsClearOutbox {
					m.notificationOutbox = nil
					if err := notification.SaveNotificationOutbox(m.outboxFile, m.notificationOutbox); err != nil {
						m.setAppError(err, "Failed to clear outbox")
					} else {
						m.setNotificationStatus("Notification queue cleared")
					}
					return m, nil
				}
				m.toggleSetting()
				return m, nil
			case "left", "h":
				m.adjustSetting(-1)
				return m, nil
			case "right", "l":
				m.adjustSetting(1)
				return m, nil
			case "esc":
				target := m.settingsReturnMode
				if target == "" {
					target = "timer"
				}
				m.mode = target
				return m, nil
			case "q", "ctrl+c":
				m.saveOnQuit()
				return m, tea.Quit
			}
			return m, nil
		}

		return m.handleKeyMsg(msg)
	}

	return m, nil
}

func (m model) setInputFocus(field int) model {
	m.showRecentOverlay = false
	m.focusedField = field
	m.textInput.Blur()
	m.durationInput.Blur()
	m.noteInput.Blur()
	m.tagInput.Blur()

	switch field {
	case focusTemplate:
		// templates don't grab typing focus
	case focusTask:
		m.textInput.Focus()
	case focusDuration:
		m.durationInput.Focus()
	case focusNote:
		m.noteInput.Focus()
	case focusTags:
		m.tagInput.Focus()
	}
	return m
}

func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	key := msg.String()

	switch key {
	case "ctrl+c", "q":
		if m.guardianLocked {
			m.abortConfirmation = false
			m.appError = "Guardian Mode Active: Press Esc to Abort"
			m.logInternal("GUARDIAN: Blocked exit attempt")
			return m, nil
		}
		m.saveOnQuit()
		return m, tea.Quit

	case "tab":
		if m.mode == "input" {
			if m.focusedField == focusTemplate {
				m = m.setInputFocus(focusTask)
			} else if m.focusedField == focusTask {
				m = m.setInputFocus(focusDuration)
			} else if m.focusedField == focusDuration {
				m = m.setInputFocus(focusNote)
			} else if m.focusedField == focusNote {
				m = m.setInputFocus(focusTags)
			} else {
				m = m.setInputFocus(focusTemplate)
			}
			return m, nil
		}
		if m.mode == "templates" {
			if len(m.templates) == 0 {
				return m, nil
			}
			m = m.cycleTemplate(1)
			return m, nil
		}
		if m.mode == "timer" || m.mode == "break" {
			m.statsReturnMode = m.mode
			m.mode = "stats"
			return m, nil
		}
		if m.mode == "stats" {
			m.mode = "analytics"
			return m, nil
		}
		if m.mode == "analytics" {
			m.mode = "heatmap"
			return m, nil
		}
		if m.mode == "heatmap" {
			m.mode = "history"
			return m, nil
		}
		if m.mode == "history" {
			m.mode = "report"
			return m, nil
		}
		if m.mode == "report" {
			m.mode = "stats"
			return m, nil
		}

	case "esc":
		if m.guardianLocked {
			if !m.abortConfirmation {
				m.abortConfirmation = true
				m.appError = "Confirm Abort? Press Esc again to force exit (Streak Penalty)"
				return m, nil
			}
			m.logInternal("GUARDIAN: Session Forcefully Aborted")
			m.stopSoundscape()
			m.runGuardianCommand(m.config.UnlockCommand)
			m.guardianLocked = false
			m.abortConfirmation = false
			m.mode = "input"
			m.taskName = ""
			m.seconds = 0
			m.sessionTarget = 0
			m.sessionElapsed = 0
			m.running = false
			m.inputError = ""
			m.textInput.SetValue("")
			m.noteInput.SetValue("")
			m.tagInput.SetValue("")
			m = m.setInputFocus(focusTask)
			return m, nil
		}
		m.abortConfirmation = false
		if m.mode == "stats" || m.mode == "analytics" || m.mode == "heatmap" || m.mode == "history" || m.mode == "report" || m.mode == "logs" {
			if m.statsReturnMode != "" {
				m.mode = m.statsReturnMode
			} else {
				m.mode = "timer"
			}
			return m, nil
		}
		if m.mode == "edit" {
			m.mode = m.editReturnMode
			m.inputError = ""
			if m.editWasRunning && m.seconds > 0 {
				m.running = true
				return m, tickCmd()
			}
			return m, nil
		}
		return m, nil

	case "up":
		if m.mode == "input" && m.focusedField == focusTask && len(m.taskSuggestions) > 0 {
			m.showRecentOverlay = true
			m = m.applyTaskSuggestion(-1)
			return m, nil
		}
		if m.mode == "templates" {
			m = m.cycleTemplate(-1)
			return m, nil
		}

	case "down":
		if m.mode == "input" && m.focusedField == focusTask && len(m.taskSuggestions) > 0 {
			m.showRecentOverlay = true
			m = m.applyTaskSuggestion(1)
			return m, nil
		}
		if m.mode == "templates" {
			m = m.cycleTemplate(1)
			return m, nil
		}

	case "left", "h":
		if m.mode == "input" && m.focusedField == focusTemplate && len(m.templates) > 0 {
			m.templateIndex = (m.templateIndex - 1 + len(m.templates)) % len(m.templates)
			m = m.applySelectedTemplate()
			return m, nil
		}
		if m.mode == "templates" {
			m = m.cycleTemplate(-1)
			return m, nil
		}

	case "right", "l":
		if m.mode == "input" && m.focusedField == focusTemplate && len(m.templates) > 0 {
			m.templateIndex = (m.templateIndex + 1) % len(m.templates)
			m = m.applySelectedTemplate()
			return m, nil
		}
		if m.mode == "templates" {
			m = m.cycleTemplate(1)
			return m, nil
		}

	case "ctrl+t":
		if m.mode == "input" {
			if err := m.saveCurrentTemplate(); err != nil {
				m.setAppError(err, "Failed to save template")
			}
			return m, nil
		}
		if m.mode == "templates" {
			if err := m.saveCurrentTemplate(); err != nil {
				m.setAppError(err, "Failed to save template")
			}
			return m, nil
		}

	case "ctrl+r":
		if m.mode == "input" && m.focusedField == focusTemplate {
			if err := m.renameSelectedTemplate(); err != nil {
				m.setAppError(err, "Failed to rename template")
			}
			return m, nil
		}
		if m.mode == "templates" {
			if err := m.renameSelectedTemplate(); err != nil {
				m.setAppError(err, "Failed to rename template")
			}
			return m, nil
		}

	case "ctrl+d":
		if m.mode == "input" && m.focusedField == focusTemplate {
			if err := m.deleteSelectedTemplate(); err != nil {
				m.setAppError(err, "Failed to delete template")
			}
			return m, nil
		}
		if m.mode == "templates" {
			if err := m.deleteSelectedTemplate(); err != nil {
				m.setAppError(err, "Failed to delete template")
			}
			return m, nil
		}

	case "ctrl+z":
		if m.mode == "templates" {
			if err := m.undoLastTemplateDelete(); err != nil {
				m.setAppError(err, "Failed to undo template delete")
			}
			return m, nil
		}

	case "ctrl+y":
		if m.mode == "templates" {
			if err := m.duplicateSelectedTemplate(); err != nil {
				m.setAppError(err, "Failed to duplicate template")
			}
			return m, nil
		}

	case "s":
		if m.mode == "timer" || m.mode == "break" || m.mode == "stats" || m.mode == "analytics" || m.mode == "heatmap" || m.mode == "history" || m.mode == "report" {
			m.settingsReturnMode = m.mode
			m.settingsCursor = settingsNotifications
			m.mode = "settings"
			return m, nil
		}

	case "L":
		if m.mode != "logs" {
			m.statsReturnMode = m.mode
			m.mode = "logs"
		} else {
			m.mode = m.statsReturnMode
			if m.mode == "" {
				m.mode = "stats"
			}
		}
		return m, nil

	case "r":
		if m.mode == "stats" || m.mode == "analytics" || m.mode == "heatmap" || m.mode == "history" {
			m.mode = "report"
			return m, nil
		}

	case "e":
		if m.mode == "report" {
			if path, err := m.exportDailyReport(); err != nil {
				m.setAppError(err, "Failed to export daily report")
			} else {
				m.setNotificationStatus(fmt.Sprintf("Daily report exported to %s", path))
			}
			return m, nil
		}
		if m.mode == "timer" || m.mode == "break" {
			m.editReturnMode = m.mode
			m.editWasRunning = m.running
			m.running = false
			m.mode = "edit"
			m.durationInput.SetValue(timer.FormatDurationInput(m.sessionTarget))
			m.durationInput.Focus()
			m.textInput.Blur()
			m.inputError = ""
			return m, nil
		}

	case "enter":
		if m.mode == "input" {
			if m.focusedField == focusTemplate {
				m = m.applySelectedTemplate()
				m = m.setInputFocus(focusTask)
				return m, nil
			}
			if m.focusedField == focusTask {
				m = m.setInputFocus(focusDuration)
				return m, nil
			}
			if m.focusedField == focusDuration {
				m = m.setInputFocus(focusNote)
				return m, nil
			}
			if m.focusedField == focusNote {
				m = m.setInputFocus(focusTags)
				return m, nil
			}
			if m.focusedField == focusTags {
				if strings.TrimSpace(m.textInput.Value()) == "" {
					m.inputError = "Task name is required."
					return m, nil
				}
				durationSeconds, err := timer.ParseDurationInput(m.durationInput.Value())
				if err != nil {
					m.inputError = err.Error()
					return m, nil
				}
				m.mode = "timer"
				m.startSoundscape()
				m.taskName = strings.TrimSpace(m.textInput.Value())
				m.textInput.Blur()
				m.durationInput.Blur()
				m.noteInput.Blur()
				m.tagInput.Blur()
				m.sessionStart = time.Now()
				m.running = true
				m.sessionTarget = durationSeconds
				m.seconds = durationSeconds
				m.sessionElapsed = 0
				m.inputError = ""
				return m, tea.Batch(tickCmd(), m.notifyCmd("session_start"))
			}
			if strings.TrimSpace(m.textInput.Value()) == "" {
				m.inputError = "Task name is required."
				return m, nil
			}
			durationSeconds, err := timer.ParseDurationInput(m.durationInput.Value())
			if err != nil {
				m.inputError = err.Error()
				return m, nil
			}
			m.mode = "timer"
			m.startSoundscape()
			m.taskName = strings.TrimSpace(m.textInput.Value())
			m.textInput.Blur()
			m.durationInput.Blur()
			m.noteInput.Blur()
			m.tagInput.Blur()
			m.sessionStart = time.Now()
			m.running = true
			m.sessionTarget = durationSeconds
			m.seconds = durationSeconds
			m.sessionElapsed = 0
			m.inputError = ""
			if m.config.GuardianMode {
				m.guardianLocked = true
				m.runGuardianCommand(m.config.LockdownCommand)
			}
			return m, tea.Batch(tickCmd(), m.notifyCmd("session_start"))
		}
		if m.mode == "templates" {
			m = m.applySelectedTemplate()
			m.mode = "input"
			m = m.setInputFocus(focusTask)
			return m, nil
		}

		if m.mode == "input" && key == "ctrl+t" {
			// handled above
		}

		if m.mode == "edit" {
			durationSeconds, err := timer.ParseDurationInput(m.durationInput.Value())
			if err != nil {
				m.inputError = err.Error()
				return m, nil
			}
			if durationSeconds <= m.sessionElapsed {
				m.inputError = "Duration must be greater than elapsed time."
				return m, nil
			}
			m.sessionTarget = durationSeconds
			m.seconds = durationSeconds - m.sessionElapsed
			m.mode = m.editReturnMode
			m.inputError = ""
			if m.editWasRunning && m.seconds > 0 {
				m.running = true
				return m, tickCmd()
			}
			return m, nil
		}

		if m.mode == "timer" || m.mode == "break" {
			notifyC := m.notifyCmd("session_end")
			model, cmd := m.completeSession()
			return model, tea.Batch(notifyC, cmd)
		}
	case " ", "space":
		if m.mode == "timer" || m.mode == "break" {
			m.running = !m.running

			if m.soundscapeIndex >= 0 && m.soundscapeIndex < len(m.soundscapes) {
				track := m.soundscapes[m.soundscapeIndex]
				if soundscape.IsSyntheticTrack(track) {
					soundscape.PauseNativeSynth(!m.running)
					if m.running {
						soundscape.FadeNativeVolume(m.config.SynthVolume, time.Duration(m.config.FadeInDuration)*time.Millisecond)
					} else {
						soundscape.FadeNativeVolume(0.15 * m.config.SynthVolume, time.Duration(m.config.FadeOutDuration)*time.Millisecond)
					}
				}
			}

			notifyC := m.notifyCmd("pause_resume")
			if m.running && m.seconds > 0 {
				return m, tea.Batch(tickCmd(), notifyC)
			}
			return m, notifyC
		}
	case "ctrl+p":
		if m.mode == "input" {
			m.templateReturnMode = m.mode
			m.templateWasRunning = m.running
			m.mode = "templates"
			return m, nil
		}
		if m.mode == "templates" {
			m.mode = "input"
			return m, nil
		}
	case "ctrl+m", "ctrl+b":
		if m.mode == "soundscapes" {
			m.mode = m.soundscapeReturnMode
			return m, nil
		}
		m.soundscapeReturnMode = m.mode
		m.mode = "soundscapes"
		return m, nil
	}

	if m.mode == "soundscapes" {
		switch key {
		case "up", "k":
			if len(m.soundscapes) > 0 {
				m.soundscapeIndex--
				if m.soundscapeIndex < -1 {
					m.soundscapeIndex = len(m.soundscapes) - 1
				}
			}
			return m, nil
		case "down", "j":
			if len(m.soundscapes) > 0 {
				m.soundscapeIndex++
				if m.soundscapeIndex >= len(m.soundscapes) {
					m.soundscapeIndex = -1
				}
			}
			return m, nil
		case "enter":
			if m.activeSessionMode() == "timer" {
				if m.soundscapeIndex == -1 {
					m.stopSoundscape()
				} else {
					m.startSoundscape()
				}
			}
			m.mode = m.soundscapeReturnMode
			return m, nil
		case "esc":
			m.mode = m.soundscapeReturnMode
			return m, nil
		}
		return m, nil
	}

	if m.mode == "input" {
		if m.focusedField == focusTask {
			m.textInput, cmd = m.textInput.Update(msg)
			m.suggestionIndex = -1
			m.showRecentOverlay = false
		} else if m.focusedField == focusDuration {
			m.durationInput, cmd = m.durationInput.Update(msg)
		} else if m.focusedField == focusNote {
			m.noteInput, cmd = m.noteInput.Update(msg)
		} else {
			m.tagInput, cmd = m.tagInput.Update(msg)
		}
		if m.inputError != "" {
			m.inputError = ""
		}
		return m, cmd
	}

	if m.mode == "edit" {
		m.durationInput, cmd = m.durationInput.Update(msg)
		if m.inputError != "" {
			m.inputError = ""
		}
		return m, cmd
	}

	return m, nil
}

func (m model) completeSession() (tea.Model, tea.Cmd) {
	m.stopSoundscape()
	if m.guardianLocked {
		m.runGuardianCommand(m.config.UnlockCommand)
		m.guardianLocked = false
	}
	flushCmd := m.saveSession()
	if m.mode == "timer" {
		m.sessionCount++
		if m.config.AutoBreak && m.sessionCount%m.config.SessionsBeforeBreak == 0 {
			m.mode = "break"
			m.sessionStart = time.Now()
			m.sessionTarget = m.config.BreakDuration * 60
			m.seconds = m.sessionTarget
			m.sessionElapsed = 0
			m.running = true
			return m, tea.Batch(tickCmd(), flushCmd)
		}
	}

	m.mode = "input"
	m.taskName = ""
	m.seconds = 0
	m.sessionTarget = 0
	m.sessionElapsed = 0
	m.running = false
	m.inputError = ""
	m.textInput.SetValue("")
	m.noteInput.SetValue("")
	m.tagInput.SetValue("")
	m = m.setInputFocus(focusTask)
	return m, flushCmd
}

func (m *model) saveSession() tea.Cmd {
	duration := m.sessionElapsed
	sessionType := "work"
	if m.mode == "break" {
		sessionType = "break"
	}

	entry := entries.Entry{
		Task:     m.taskName,
		Note:     strings.TrimSpace(m.noteInput.Value()),
		Tags:     parseTags(m.tagInput.Value()),
		Start:    m.sessionStart,
		End:      time.Now(),
		Duration: duration,
		Type:     sessionType,
	}

	var entriesList []entries.Entry
	if data, err := os.ReadFile(m.dataFile); err == nil {
		if err := json.Unmarshal(data, &entriesList); err != nil {
			m.setAppError(err, "Failed to parse entries")
			return nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		m.setAppError(err, "Failed to read entries")
		return nil
	}
	entriesList = append(entriesList, entry)
	fileData, err := json.MarshalIndent(entriesList, "", "  ")
	if err != nil {
		m.setAppError(err, "Failed to encode entries")
		return nil
	}
	if err := os.WriteFile(m.dataFile, fileData, 0644); err != nil {
		m.setAppError(err, "Failed to write entries")
		return nil
	}

	if m.petEnabled {
		leveledUp := false
		coinsGained := 0
		if sessionType == "work" {
			leveledUp = m.petState.Feed(duration / 60)
			coinsGained = duration / 60
			if coinsGained < 5 {
				coinsGained = 5
			}
			m.petState.Coins += coinsGained
			m.logInternal("PET: Work block complete. Gained XP and earned %d Pomo-Coins! (Total: %d)", coinsGained, m.petState.Coins)
		} else {
			leveledUp = m.petState.AddXP(20)
			coinsGained = 5
			m.petState.Coins += coinsGained
			m.logInternal("PET: Break complete. Gained 20 XP and earned 5 Pomo-Coins! (Total: %d)", m.petState.Coins)
		}

		if sessionType == "work" && rand.Float64() < 0.15 && m.petState.ActiveItem == "" {
			items := []string{"Wizard Hat", "Cyber Visor", "Golden Crown", "Laser Goggles", "Mini Cape"}
			discovered := items[rand.Intn(len(items))]
			m.petState.ActiveItem = discovered
			m.petState.AddXP(50)
			m.logInternal("PET: %s found a rare item: %s!", m.petState.Name, discovered)
		}

		if err := pet.SavePetState(m.petFile, m.petState); err != nil {
			m.setAppError(err, "Failed to save pet state")
		}

		if leveledUp {
			m.logInternal("PET: Level Up! %s reached Level %d", m.petState.Name, m.petState.Level)
			m.showPetLevelUpOverlay = true
		}
	}

	m.logInternal("SESSION: Saved %s (%s)", m.taskName, streak.FormatDuration(duration))
	m.entries = entriesList
	fileTasks := tasks.LoadTasksFromFile(m.config.TasksFile)
	m.taskSuggestions = tasks.BuildTaskSuggestions(entriesList, m.config.PinnedTasks, fileTasks)
	m.suggestionIndex = -1
	return nil
}

func (m *model) runGuardianCommand(cmdStr string) {
	if strings.TrimSpace(cmdStr) == "" {
		return
	}
	m.logInternal("GUARDIAN: Running command: %s", cmdStr)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}
	if err := cmd.Run(); err != nil {
		m.logInternal("GUARDIAN: Command failed: %v", err)
	} else {
		m.logInternal("GUARDIAN: Command completed successfully")
	}
}

func (m *model) setAppError(err error, context string) {
	if err == nil {
		return
	}
	msg := err.Error()
	if context != "" {
		msg = fmt.Sprintf("%s: %v", context, err)
	}
	m.appError = msg
	m.logInternal("ERROR: %s", msg)
}

func (m *model) setNotificationStatus(status string) {
	m.notificationStatus = status
	if status != "" {
		m.notificationCounter++
		m.logInternal("STATUS: %s", status)
	}
}

func (m *model) logInternal(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	entry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	m.internalLogs = append(m.internalLogs, entry)
	if len(m.internalLogs) > 100 {
		m.internalLogs = m.internalLogs[len(m.internalLogs)-100:]
	}
}

func (m model) eventEnabled(event string) bool {
	switch event {
	case "work_complete":
		return m.config.NotifyWorkComplete
	case "break_complete":
		return m.config.NotifyBreakComplete
	case "session_start":
		return m.config.NotifySessionStart
	case "session_end":
		return m.config.NotifySessionEnd
	case "pause_resume":
		return m.config.NotifyPauseResume
	case "ending_soon":
		return m.config.NotifyEndingSoon
	default:
		return false
	}
}

func (m model) quietHoursActive(now time.Time) bool {
	start := m.config.QuietHoursStart
	end := m.config.QuietHoursEnd
	if start < 0 || start > 23 || end < 0 || end > 23 || start == end {
		return false
	}
	hour := now.Hour()
	if start < end {
		return hour >= start && hour < end
	}
	return hour >= start || hour < end
}

func (m model) notificationTitle(event string) string {
	switch event {
	case "work_complete":
		return "Work session complete"
	case "break_complete":
		return "Break complete"
	case "session_start":
		return "Session started"
	case "session_end":
		return "Session ended"
	case "pause_resume":
		if m.running {
			return "Session resumed"
		}
		return "Session paused"
	case "ending_soon":
		return "Session ending soon"
	default:
		return "Kairu"
	}
}

func (m model) notificationBody(event string) string {
	switch event {
	case "work_complete":
		return fmt.Sprintf("%s completed in %s", m.taskName, streak.FormatDuration(m.sessionElapsed))
	case "break_complete":
		return "Break is over. Ready to focus again?"
	case "session_start":
		return fmt.Sprintf("Focus session started: %s", m.taskName)
	case "session_end":
		return fmt.Sprintf("Session ended: %s", m.taskName)
	case "pause_resume":
		if m.running {
			return "Focus timer resumed."
		}
		return "Focus timer paused."
	case "ending_soon":
		return fmt.Sprintf("Only %s left in this session.", streak.FormatDuration(m.seconds))
	default:
		return ""
	}
}

func (m model) notificationID(event string) string {
	base := fmt.Sprintf("%s-%d", event, m.sessionStart.UnixNano())
	switch event {
	case "pause_resume":
		return fmt.Sprintf("%s-%t-%d", base, m.running, m.sessionElapsed)
	case "session_start":
		return fmt.Sprintf("%s-%s", base, m.taskName)
	case "session_end", "work_complete", "break_complete", "ending_soon":
		return fmt.Sprintf("%s-%d", base, m.sessionElapsed)
	default:
		return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
	}
}

func (m model) hasNotification(id string) bool {
	if id == "" {
		return false
	}
	if m.deliveredNotifyIDs != nil {
		if _, ok := m.deliveredNotifyIDs[id]; ok {
			return true
		}
	}
	for _, job := range m.notificationOutbox {
		if job.ID == id {
			return true
		}
	}
	return false
}

func (m model) notifyCmd(event string) tea.Cmd {
	if !m.config.Notifications || !m.eventEnabled(event) {
		return nil
	}
	if m.quietHoursActive(time.Now()) {
		return func() tea.Msg {
			return notifResultMsg{status: "Notification suppressed by quiet hours"}
		}
	}
	title := m.notificationTitle(event)
	body := m.notificationBody(event)
	if body == "" {
		return nil
	}
	id := m.notificationID(event)
	if m.hasNotification(id) {
		return func() tea.Msg {
			return notifResultMsg{id: id, status: "Duplicate notification suppressed"}
		}
	}
	job := notification.NewNotificationJob(id, event, title, body)
	cfg := m.config
	outboxFile := m.outboxFile
	existing := append([]notification.NotificationJob(nil), m.notificationOutbox...)
	return func() tea.Msg {
		status, err := notification.SendNotification(cfg, job)
		if err == nil {
			return notifResultMsg{id: id, status: status}
		}
		job.Attempts = 1
		job.LastError = err.Error()
		notification.ScheduleNextAttempt(&job)
		updated := append(existing, job)
		if saveErr := notification.SaveNotificationOutbox(outboxFile, updated); saveErr != nil {
			return outboxFlushedMsg{remaining: updated, err: saveErr}
		}
		return outboxFlushedMsg{
			remaining: updated,
			status:    "Notification queued for retry",
		}
	}
}

func (m model) flushOutboxCmd() tea.Cmd {
	if !m.config.Notifications || len(m.notificationOutbox) == 0 {
		return nil
	}
	jobs := make([]notification.NotificationJob, len(m.notificationOutbox))
	copy(jobs, m.notificationOutbox)
	cfg := m.config
	outboxFile := m.outboxFile
	return func() tea.Msg {
		remaining := make([]notification.NotificationJob, 0)
		var delivered []string
		var lastStatus string
		now := time.Now()
		for _, job := range jobs {
			if !job.NextAttemptAt.IsZero() && job.NextAttemptAt.After(now) {
				remaining = append(remaining, job)
				continue
			}
			status, err := notification.SendNotification(cfg, job)
			if err != nil {
				job.Attempts++
				job.LastError = err.Error()
				notification.ScheduleNextAttempt(&job)
				remaining = append(remaining, job)
				continue
			}
			lastStatus = status
			delivered = append(delivered, job.ID)
		}
		var saveErr error
		if err := notification.SaveNotificationOutbox(outboxFile, remaining); err != nil {
			saveErr = err
		}
		return outboxFlushedMsg{remaining: remaining, deliveredIDs: delivered, status: lastStatus, err: saveErr}
	}
}

func reloadProjectState(m *model) error {
	cfg, err := config.LoadConfig(m.configFile)
	if err != nil {
		return err
	}
	templatesList, err := templates.LoadSessionTemplates(m.templateFile)
	if err != nil {
		return err
	}
	entryList, err := entries.LoadEntries(m.dataFile)
	if err != nil {
		return err
	}
	pState, petErr := pet.LoadPetState(m.petFile)
	if petErr != nil {
		pState = pet.DefaultPet("Neko", "kitty")
		_ = pet.SavePetState(m.petFile, pState)
	}

	m.config = cfg
	m.templates = templatesList
	m.entries = entryList
	m.petState = pState
	m.streakState = streak.ComputeStreakState(entryList)

	fileTasks := tasks.LoadTasksFromFile(cfg.TasksFile)
	m.taskSuggestions = tasks.BuildTaskSuggestions(entryList, cfg.PinnedTasks, fileTasks)
	m.suggestionIndex = -1

	soundscapesList, _ := soundscape.LoadSoundscapes(cfg.SoundscapesDir)
	m.soundscapes = soundscapesList

	return nil
}
