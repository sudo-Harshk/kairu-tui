package config

type ThemeStyle struct {
	Accent    string
	Primary   string
	Notice    string
	Warning   string
	Success   string
	Dim       string
	Surface   string
	Border    string
	Highlight string
}

var ThemeStyles = map[string]ThemeStyle{
	"forest": {
		Accent:    "10",
		Primary:   "2",
		Notice:    "3",
		Warning:   "1",
		Success:   "10",
		Dim:       "8",
		Surface:   "0",
		Border:    "2",
		Highlight: "10",
	},
	"ocean": {
		Accent:    "14",
		Primary:   "6",
		Notice:    "12",
		Warning:   "9",
		Success:   "14",
		Dim:       "8",
		Surface:   "0",
		Border:    "6",
		Highlight: "14",
	},
	"ember": {
		Accent:    "208",
		Primary:   "214",
		Notice:    "220",
		Warning:   "196",
		Success:   "120",
		Dim:       "244",
		Surface:   "0",
		Border:    "214",
		Highlight: "208",
	},
	"mono": {
		Accent:    "15",
		Primary:   "7",
		Notice:    "8",
		Warning:   "9",
		Success:   "15",
		Dim:       "8",
		Surface:   "0",
		Border:    "7",
		Highlight: "15",
	},
	"matrix": {
		Accent:    "46",
		Primary:   "22",
		Notice:    "28",
		Warning:   "1",
		Success:   "46",
		Dim:       "28",
		Surface:   "0",
		Border:    "22",
		Highlight: "46",
	},
	"cyberpunk": {
		Accent:    "201",
		Primary:   "51",
		Notice:    "226",
		Warning:   "196",
		Success:   "121",
		Dim:       "244",
		Surface:   "0",
		Border:    "51",
		Highlight: "201",
	},
	"minimalist": {
		Accent:    "244",
		Primary:   "248",
		Notice:    "240",
		Warning:   "1",
		Success:   "250",
		Dim:       "242",
		Surface:   "0",
		Border:    "248",
		Highlight: "244",
	},
}

var ThemeOrder = []string{"forest", "ocean", "ember", "mono", "matrix", "cyberpunk", "minimalist"}
