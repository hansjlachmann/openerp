package themes

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// MaxThemeFileSize is the largest theme file (colors.toml, alacritty.toml) accepted.
const MaxThemeFileSize = 64 << 10

// ErrNoColors is returned when a theme file has no background or foreground color.
var ErrNoColors = errors.New("theme file has no background and foreground colors")

// ParsePalette reads the palette of an Omarchy theme file:
//   - colors.toml, newer form: mode, accent, selection, muted, background,
//     lighter_background, dark(er)_background, foreground, named colors (red, blue, …)
//   - colors.toml, older form: accent, foreground, background, selection_background,
//     selection_foreground, color0..color15
//   - alacritty.toml (themes without colors.toml): [colors.primary], [colors.selection],
//     [colors.normal], [colors.bright]
//
// name is the file name and decides the form. Only string color values are read, each
// through ParseColor; other keys (hyprland_*, cursor, …) are ignored. An invalid value of an
// optional color is skipped, a missing or invalid background/foreground is an error.
func ParsePalette(name string, data []byte) (Palette, error) {
	if len(data) > MaxThemeFileSize {
		return Palette{}, fmt.Errorf("%s is larger than %d KB", name, MaxThemeFileSize>>10)
	}
	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
		return Palette{}, fmt.Errorf("%s: %w", name, err)
	}

	type keys [][]string // alternatives, first found wins; each a path into the document
	var bg, fg, accent, sel, selFg, muted, lighter, darker keys
	if strings.EqualFold(name, "alacritty.toml") {
		bg = keys{{"colors", "primary", "background"}}
		fg = keys{{"colors", "primary", "foreground"}}
		accent = keys{{"colors", "normal", "blue"}, {"colors", "normal", "cyan"}}
		sel = keys{{"colors", "selection", "background"}}
		selFg = keys{{"colors", "selection", "text"}}
		muted = keys{{"colors", "bright", "black"}}
		darker = keys{{"colors", "normal", "black"}}
	} else {
		bg = keys{{"background"}}
		fg = keys{{"foreground"}}
		accent = keys{{"accent"}, {"blue"}, {"color4"}, {"cyan"}, {"color6"}}
		sel = keys{{"selection"}, {"selection_background"}}
		selFg = keys{{"selection_foreground"}}
		muted = keys{{"muted"}, {"color8"}}
		lighter = keys{{"lighter_background"}}
		darker = keys{{"darker_background"}, {"dark_background"}, {"color0"}}
	}

	find := func(alts keys) *RGB {
		for _, path := range alts {
			if s, ok := lookup(doc, path); ok {
				if c, err := ParseColor(s); err == nil {
					return &c
				}
			}
		}
		return nil
	}

	p := Palette{
		Accent: find(accent), Selection: find(sel), SelectionFg: find(selFg),
		Muted: find(muted), Lighter: find(lighter), Darker: find(darker),
	}
	b, f := find(bg), find(fg)
	if b == nil || f == nil {
		return Palette{}, fmt.Errorf("%s: %w", name, ErrNoColors)
	}
	p.Background, p.Foreground = *b, *f
	if m, ok := lookup(doc, []string{"mode"}); ok && (m == "light" || m == "dark") {
		p.Mode = m
	}
	return p, nil
}

// lookup returns the string at path in a decoded TOML document.
func lookup(doc map[string]any, path []string) (string, bool) {
	var cur any = doc
	for _, k := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return "", false
		}
		if cur, ok = m[k]; !ok {
			return "", false
		}
	}
	s, ok := cur.(string)
	return s, ok
}
