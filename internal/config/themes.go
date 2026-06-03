package config

type ThemeStyle struct {
	WorkAccent  string
	BreakAccent string
	Primary     string
	Notice      string
	Warning     string
	Success     string
	Muted       string
	Background  string
	Border      string
	Highlight   string
	Base        string
}

// StateAccent returns WorkAccent or BreakAccent depending on the mode.
func StateAccent(theme ThemeStyle, mode string) string {
	if mode == "break" || mode == "longbreak" {
		return theme.BreakAccent
	}
	return theme.WorkAccent
}

var ThemeStyles = map[string]ThemeStyle{
	"forest": {
		WorkAccent:  "#ff79c6",
		BreakAccent: "#a6e3a1",
		Primary:     "#cdd6f4",
		Notice:      "#f9e2af",
		Warning:     "#f38ba8",
		Success:     "#a6e3a1",
		Muted:       "#6c7086",
		Background:  "#1e1e2e",
		Border:      "#313244",
		Highlight:   "#ff79c6",
		Base:        "#1e1e2e",
	},
	"ocean": {
		WorkAccent:  "#89b4fa",
		BreakAccent: "#a6e3a1",
		Primary:     "#89dceb",
		Notice:      "#f9e2af",
		Warning:     "#f38ba8",
		Success:     "#a6e3a1",
		Muted:       "#6c7086",
		Background:  "#1e1e2e",
		Border:      "#313244",
		Highlight:   "#89b4fa",
		Base:        "#1e1e2e",
	},
	"ember": {
		WorkAccent:  "#fab387",
		BreakAccent: "#a6e3a1",
		Primary:     "#f9e2af",
		Notice:      "#f9e2af",
		Warning:     "#f38ba8",
		Success:     "#a6e3a1",
		Muted:       "#6c7086",
		Background:  "#1e1e2e",
		Border:      "#313244",
		Highlight:   "#fab387",
		Base:        "#1e1e2e",
	},
	"mono": {
		WorkAccent:  "#cdd6f4",
		BreakAccent: "#bac2de",
		Primary:     "#bac2de",
		Notice:      "#a6adc8",
		Warning:     "#f38ba8",
		Success:     "#a6e3a1",
		Muted:       "#585b70",
		Background:  "#1e1e2e",
		Border:      "#313244",
		Highlight:   "#cdd6f4",
		Base:        "#1e1e2e",
	},
	"matrix": {
		WorkAccent:  "#a6e3a1",
		BreakAccent: "#94e2d5",
		Primary:     "#94e2d5",
		Notice:      "#f9e2af",
		Warning:     "#f38ba8",
		Success:     "#a6e3a1",
		Muted:       "#6c7086",
		Background:  "#1e1e2e",
		Border:      "#313244",
		Highlight:   "#a6e3a1",
		Base:        "#1e1e2e",
	},
	"cyberpunk": {
		WorkAccent:  "#f5c2e7",
		BreakAccent: "#cba6f7",
		Primary:     "#f5c2e7",
		Notice:      "#f9e2af",
		Warning:     "#f38ba8",
		Success:     "#a6e3a1",
		Muted:       "#6c7086",
		Background:  "#1e1e2e",
		Border:      "#313244",
		Highlight:   "#f5c2e7",
		Base:        "#1e1e2e",
	},
	"minimalist": {
		WorkAccent:  "#f2cdcd",
		BreakAccent: "#b4befe",
		Primary:     "#cdd6f4",
		Notice:      "#f9e2af",
		Warning:     "#f38ba8",
		Success:     "#a6e3a1",
		Muted:       "#6c7086",
		Background:  "#1e1e2e",
		Border:      "#313244",
		Highlight:   "#f2cdcd",
		Base:        "#1e1e2e",
	},
}

var ThemeOrder = []string{"forest", "ocean", "ember", "mono", "matrix", "cyberpunk", "minimalist"}
