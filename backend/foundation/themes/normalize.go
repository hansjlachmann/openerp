package themes

// Palette is a theme's colors as read from a theme file (any dialect, see parse.go).
// Background and Foreground are required; nil fields are derived.
type Palette struct {
	Mode        string // "light", "dark" or "" (decided by the background)
	Background  RGB
	Foreground  RGB
	Accent      *RGB
	Selection   *RGB // selection background
	SelectionFg *RGB
	Muted       *RGB // de-emphasized text
	Lighter     *RGB // raised surface (dark themes)
	Darker      *RGB // deepest surface, used for the app bar
}

// Contrast minimums (WCAG): body text AAA, secondary text and links AA, muted text 3:1.
const (
	minText      = 7.0
	minSecondary = 4.5
	minMuted     = 3.0
)

// ResolveMode returns the palette's mode. The background decides — a declared mode that
// contradicts it would turn the ramps upside down; the declaration only settles a
// mid-tone background.
func (p Palette) ResolveMode() string {
	lum := Luminance(p.Background)
	switch {
	case lum < 0.1:
		return "dark"
	case lum > 0.3:
		return "light"
	case p.Mode == "light" || p.Mode == "dark":
		return p.Mode
	case lum < 0.18:
		return "dark"
	}
	return "light"
}

// Normalize turns a palette into tokens. The gray ramp runs from the background to the
// foreground in OKLab with the classic lightness profile, the accent ramp keeps the accent's hue
// with teal's profile, and text colors are adjusted until they reach the contrast
// minimums against the page — a theme can recolor the app but never make it unreadable.
func Normalize(p Palette) Tokens {
	mode := p.ResolveMode()
	bg, fg := p.Background, p.Foreground
	// A mid-tone page cannot carry readable text: move it towards black (dark) or white
	// (light) until the text color can reach the minimum
	if mode == "dark" {
		bg = EnsureContrast(bg, RGB{255, 255, 255}, minText+1)
	} else {
		bg = EnsureContrast(bg, RGB{0, 0, 0}, minText+1)
	}
	t := Tokens{Version: TokensVersion, Mode: mode}

	var gray [11]RGB
	if mode == "dark" {
		fg = EnsureContrast(fg, bg, minText)
		// dark: 900 is the page, 50 the text
		l900, l50 := MustColor(classicGray[9]).ToLab().L, MustColor(classicGray[0]).ToLab().L
		for i, h := range classicGray {
			gray[i] = Mix(bg, fg, (MustColor(h).ToLab().L-l900)/(l50-l900))
		}
		gray[9], gray[0] = bg, fg
		if p.Lighter != nil {
			gray[8] = *p.Lighter
		}
		if p.Muted != nil {
			gray[5] = *p.Muted
		}
		gray[4] = EnsureContrast(gray[4], bg, minSecondary) // secondary text
		gray[5] = EnsureContrast(gray[5], bg, minMuted)
	} else {
		fg = EnsureContrast(fg, bg, minText)
		// light: white page, #212121 text; 50 sits just below the page, 900 beyond the text
		lText := MustColor("#212121").ToLab().L
		for i, h := range classicGray {
			gray[i] = Mix(bg, fg, (1-MustColor(h).ToLab().L)/(1-lText))
		}
		if p.Muted != nil {
			gray[5] = *p.Muted
		}
		gray[6] = EnsureContrast(gray[6], bg, minSecondary) // secondary text
		gray[5] = EnsureContrast(gray[5], bg, minMuted)
		gray[9] = EnsureContrast(gray[9], bg, minText)
	}

	accent := Mix(bg, fg, 0.5)
	if p.Accent != nil {
		accent = *p.Accent
	}
	// The accent is the dark-mode link step (400) or the light-mode one (600)
	anchor := 6
	if mode == "dark" {
		anchor = 4
	}
	acc := accentRamp(accent, anchor)
	acc[anchor] = EnsureContrast(acc[anchor], bg, minSecondary) // links
	if mode == "dark" {
		acc[5] = EnsureContrast(acc[5], RGB{255, 255, 255}, minMuted) // buttons with white text
	} else {
		acc[6] = EnsureContrast(acc[6], RGB{255, 255, 255}, minMuted)
	}

	for i := range gray {
		t.Gray[i] = gray[i].Hex()
		t.Accent[i] = acc[i].Hex()
	}

	v := &t.Vars
	v.PageBg, v.Fg, v.Heading = bg.Hex(), fg.Hex(), fg.Hex()
	if mode == "dark" {
		v.Surface = gray[8].Hex()
		appbar := gray[8]
		if p.Darker != nil {
			appbar = *p.Darker
		}
		v.Appbar, v.AppbarFg = appbar.Hex(), EnsureContrast(fg, appbar, minSecondary).Hex()

		sel := Mix(bg, accent, 0.3)
		if p.Selection != nil {
			sel = *p.Selection
		}
		sel = EnsureContrast(sel, fg, minSecondary)
		v.RowSelected, v.RowSelectedHover = sel.Hex(), Mix(sel, fg, 0.08).Hex()

		textSel := gray[6]
		if p.Selection != nil {
			textSel = *p.Selection
		}
		textSelFg := fg
		if p.SelectionFg != nil {
			textSelFg = *p.SelectionFg
		}
		v.TextSelBg, v.TextSelFg = textSel.Hex(), EnsureContrast(textSelFg, textSel, minSecondary).Hex()

		v.ScrollTrack, v.ScrollThumb, v.ScrollThumbHover = Mix(bg, fg, 0.1).Hex(), Mix(bg, fg, 0.55).Hex(), Mix(bg, fg, 0.7).Hex()
	} else {
		v.Surface = bg.Hex()
		// A dark app bar, unless the theme has its own deep surface
		appbar, appbarFg := Mix(fg, bg, 0.08), bg
		if p.Darker != nil {
			appbar, appbarFg = *p.Darker, fg
		}
		v.Appbar, v.AppbarFg = appbar.Hex(), EnsureContrast(appbarFg, appbar, minSecondary).Hex()

		sel := EnsureContrast(acc[1], fg, minSecondary)
		v.RowSelected, v.RowSelectedHover = sel.Hex(), sel.Hex()

		textSel := acc[6]
		if p.Selection != nil {
			textSel = *p.Selection
		}
		textSelFg := ReadableOn(textSel)
		if p.SelectionFg != nil {
			textSelFg = *p.SelectionFg
		}
		v.TextSelBg, v.TextSelFg = textSel.Hex(), EnsureContrast(textSelFg, textSel, minSecondary).Hex()

		v.ScrollTrack, v.ScrollThumb, v.ScrollThumbHover = Mix(bg, fg, 0.02).Hex(), Mix(bg, fg, 0.45).Hex(), Mix(bg, fg, 0.6).Hex()
	}
	return t
}

// accentRamp builds the 11 accent steps around accent: hue and chroma kept, lightness
// shifted along teal's profile so that step anchor is the accent itself.
func accentRamp(accent RGB, anchor int) [11]RGB {
	var out [11]RGB
	delta := accent.ToLab().L - MustColor(classicTeal[anchor]).ToLab().L
	for i, h := range classicTeal {
		L := MustColor(h).ToLab().L + delta
		if L > 0.98 {
			L = 0.98
		}
		if L < 0.08 {
			L = 0.08
		}
		out[i] = WithL(accent, L)
	}
	out[anchor] = accent
	return out
}
