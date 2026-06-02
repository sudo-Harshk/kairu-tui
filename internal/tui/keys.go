package tui

const (
	focusTemplate = iota
	focusTask
	focusDuration
	focusNote
	focusTags
)

const (
	templateActionApply = iota
	templateActionSave
	templateActionRename
	templateActionDelete
	templateActionDuplicate
	templateActionCount
)

const (
	settingsNotifications = iota
	settingsDesktop
	settingsWorkComplete
	settingsBreakComplete
	settingsSessionStart
	settingsSessionEnd
	settingsPauseResume
	settingsEndingSoon
	settingsTheme
	settingsFont
	settingsLayout
	settingsQuietStart
	settingsQuietEnd
	settingsSynthVolume
	settingsBinauralPreset
	settingsBinauralCarrier
	settingsBinauralBeat
	settingsFadeIn
	settingsFadeOut
	settingsBackup
	settingsRestore
	settingsClearOutbox
	settingsCount
)
