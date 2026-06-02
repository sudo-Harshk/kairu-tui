package config

type ThemeStyle struct {
	Accent  string
	Primary string
	Notice  string
	Warning string
}

var ThemeStyles = map[string]ThemeStyle{
	"forest":     {Accent: "10", Primary: "2", Notice: "3", Warning: "1"},
	"ocean":      {Accent: "14", Primary: "6", Notice: "12", Warning: "9"},
	"ember":      {Accent: "208", Primary: "214", Notice: "220", Warning: "196"},
	"mono":       {Accent: "15", Primary: "7", Notice: "8", Warning: "9"},
	"matrix":     {Accent: "46", Primary: "22", Notice: "28", Warning: "1"},
	"cyberpunk":  {Accent: "201", Primary: "51", Notice: "226", Warning: "196"},
	"minimalist": {Accent: "244", Primary: "248", Notice: "240", Warning: "1"},
}

var ThemeOrder = []string{"forest", "ocean", "ember", "mono", "matrix", "cyberpunk", "minimalist"}
