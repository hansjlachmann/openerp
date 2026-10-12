package themes

import (
	"encoding/json"
	"errors"
	"regexp"
)

// Steps are the steps of the gray and accent ramps (Tailwind's scale).
var Steps = [11]int{50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950}

// Tokens are a theme's colors as the frontend applies them: every value becomes a CSS
// variable on <html> (gray[i] → --gray-<step>, accent[i] → --accent-<step>, vars["fg"] →
// --fg), see frontend/src/app.css. Values are always #rrggbb.
type Tokens struct {
	Version int        `json:"version"`
	Mode    string     `json:"mode"` // "light" or "dark": the mode the theme is made for
	Gray    [11]string `json:"gray"`
	Accent  [11]string `json:"accent"`
	Vars    Vars       `json:"vars"`
}

// Vars are the semantic colors (CSS variable name without "--").
type Vars struct {
	PageBg           string `json:"page-bg"`
	Surface          string `json:"surface"`
	Fg               string `json:"fg"`
	Heading          string `json:"heading"`
	Appbar           string `json:"appbar"`
	AppbarFg         string `json:"appbar-fg"`
	RowSelected      string `json:"row-selected"`
	RowSelectedHover string `json:"row-selected-hover"`
	TextSelBg        string `json:"text-sel-bg"`
	TextSelFg        string `json:"text-sel-fg"`
	ScrollTrack      string `json:"scroll-track"`
	ScrollThumb      string `json:"scroll-thumb"`
	ScrollThumbHover string `json:"scroll-thumb-hover"`
}

// TokensVersion is the version of the token format.
const TokensVersion = 1

// bcGray and bcTeal are the Business Central ramps — the defaults in app.css. Generated
// ramps follow their lightness profile, so a theme spreads its grays like BC does.
var (
	bcGray = [11]string{"#f7f7f7", "#f2f2f3", "#e5e7e9", "#d3d6da", "#a4b0c4", "#737d8a", "#505c6d", "#303032", "#1e1e1e", "#121212", "#0a0a0a"}
	bcTeal = [11]string{"#e6f3f4", "#cce8ea", "#99d1d4", "#66b9bf", "#37a1a5", "#00838f", "#008489", "#006e72", "#00585c", "#003a3e", "#002a2d"}
)

// bcLightTokens and bcDarkTokens are exactly the defaults in frontend/src/app.css (BC
// Light/Dark): applying them changes nothing.
func bcLightTokens() Tokens {
	return Tokens{
		Version: TokensVersion, Mode: "light", Gray: bcGray, Accent: bcTeal,
		Vars: Vars{
			PageBg: "#ffffff", Surface: "#ffffff", Fg: "#212121", Heading: "#282828",
			Appbar: "#282828", AppbarFg: "#ffffff",
			RowSelected: "#cce8ea", RowSelectedHover: "#cce8ea",
			TextSelBg: "#0078d4", TextSelFg: "#ffffff",
			ScrollTrack: "#fcfcfc", ScrollThumb: "#8b8b8b", ScrollThumbHover: "#6e6e6e",
		},
	}
}

func bcDarkTokens() Tokens {
	return Tokens{
		Version: TokensVersion, Mode: "dark", Gray: bcGray, Accent: bcTeal,
		Vars: Vars{
			PageBg: "#121212", Surface: "#ffffff", Fg: "#f7f7f7", Heading: "#282828",
			Appbar: "#282828", AppbarFg: "#ffffff",
			RowSelected: "#3c4047", RowSelectedHover: "#454a52",
			TextSelBg: "#505c6d", TextSelFg: "#f7f7f7",
			ScrollTrack: "#2c2c2c", ScrollThumb: "#9f9f9f", ScrollThumbHover: "#bdbdbd",
		},
	}
}

var tokenColor = regexp.MustCompile(`^#[0-9a-f]{6}$`)

// ErrTokens is returned for stored tokens that are not a valid theme.
var ErrTokens = errors.New("invalid theme tokens")

// ParseTokens reads stored tokens (Theme.tokens) and checks them strictly: known version
// and mode, and every color exactly #rrggbb — whatever wrote the record.
func ParseTokens(s string) (Tokens, error) {
	var t Tokens
	if err := json.Unmarshal([]byte(s), &t); err != nil {
		return Tokens{}, ErrTokens
	}
	if t.Version != TokensVersion || (t.Mode != "light" && t.Mode != "dark") {
		return Tokens{}, ErrTokens
	}
	v := t.Vars
	vals := append(append(t.Gray[:0:0], t.Gray[:]...), t.Accent[:]...)
	vals = append(vals, v.PageBg, v.Surface, v.Fg, v.Heading, v.Appbar, v.AppbarFg, v.RowSelected,
		v.RowSelectedHover, v.TextSelBg, v.TextSelFg, v.ScrollTrack, v.ScrollThumb, v.ScrollThumbHover)
	for _, c := range vals {
		if !tokenColor.MatchString(c) {
			return Tokens{}, ErrTokens
		}
	}
	return t, nil
}
