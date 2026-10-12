// Package themes holds the color themes of the user interface: built-in themes, the
// normalization of a palette (e.g. an Omarchy theme's colors.toml) into the tokens the
// frontend applies as CSS variables, and the color math behind it (OKLab, WCAG contrast).
//
// Themes carry colors only — never CSS, markup or URLs. Every color is parsed strictly and
// stored re-serialized as #rrggbb, so nothing from a theme file reaches the browser as typed.
package themes

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// RGB is an sRGB color.
type RGB struct{ R, G, B uint8 }

// Hex returns the color as #rrggbb.
func (c RGB) Hex() string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

var (
	hexColor  = regexp.MustCompile(`^(?:#|0x)?([0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	rgbaColor = regexp.MustCompile(`^rgba?\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*(?:,\s*(?:0|1|0?\.\d+)\s*)?\)$`)
	// Hyprland style: rgba(f0913fee) / rgb(f0913f)
	rgbaHex = regexp.MustCompile(`^rgba?\(([0-9a-fA-F]{6})(?:[0-9a-fA-F]{2})?\)$`)
)

// ParseColor parses #rgb, #rrggbb, #rrggbbaa (also without # or with 0x, as in alacritty
// files), rgb(r, g, b), rgba(r, g, b, a) and rgba(rrggbbaa). Alpha is dropped. Anything else
// is an error: theme values are never passed on as typed.
func ParseColor(s string) (RGB, error) {
	s = strings.TrimSpace(s)
	if len(s) > 40 {
		return RGB{}, fmt.Errorf("invalid color %.40q", s)
	}
	if m := hexColor.FindStringSubmatch(s); m != nil {
		h := m[1]
		if len(h) == 3 {
			h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
		}
		v, _ := strconv.ParseUint(h[:6], 16, 32)
		return RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)}, nil
	}
	if m := rgbaHex.FindStringSubmatch(s); m != nil {
		v, _ := strconv.ParseUint(m[1], 16, 32)
		return RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)}, nil
	}
	if m := rgbaColor.FindStringSubmatch(s); m != nil {
		var c [3]uint8
		for i := range 3 {
			v, _ := strconv.Atoi(m[i+1])
			if v > 255 {
				return RGB{}, fmt.Errorf("invalid color %q", s)
			}
			c[i] = uint8(v)
		}
		return RGB{c[0], c[1], c[2]}, nil
	}
	return RGB{}, fmt.Errorf("invalid color %q", s)
}

// MustColor parses a color known to be valid (built-in values).
func MustColor(s string) RGB {
	c, err := ParseColor(s)
	if err != nil {
		panic(err)
	}
	return c
}

// ---- sRGB <-> linear <-> OKLab ----

func toLinear(v uint8) float64 {
	c := float64(v) / 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func fromLinear(c float64) uint8 {
	if c <= 0.0031308 {
		c *= 12.92
	} else {
		c = 1.055*math.Pow(c, 1/2.4) - 0.055
	}
	return uint8(math.Round(math.Max(0, math.Min(1, c)) * 255))
}

// Lab is a color in OKLab (L 0..1).
type Lab struct{ L, A, B float64 }

// ToLab converts an sRGB color to OKLab.
func (c RGB) ToLab() Lab {
	r, g, b := toLinear(c.R), toLinear(c.G), toLinear(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return Lab{
		L: 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		A: 1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		B: 0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
	}
}

// linear returns the color's linear sRGB components (may be out of 0..1).
func (p Lab) linear() (float64, float64, float64) {
	l := p.L + 0.3963377774*p.A + 0.2158037573*p.B
	m := p.L - 0.1055613458*p.A - 0.0638541728*p.B
	s := p.L - 0.0894841775*p.A - 1.2914855480*p.B
	l, m, s = l*l*l, m*m*m, s*s*s
	return 4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		-0.0041960863*l - 0.7034186147*m + 1.7076147010*s
}

// inGamut reports whether the color fits sRGB.
func (p Lab) inGamut() bool {
	r, g, b := p.linear()
	const e = 1e-4
	return r >= -e && r <= 1+e && g >= -e && g <= 1+e && b >= -e && b <= 1+e
}

// ToRGB converts to sRGB; a color outside sRGB loses chroma (hue and lightness kept) until
// it fits.
func (p Lab) ToRGB() RGB {
	p.L = math.Max(0, math.Min(1, p.L))
	if !p.inGamut() {
		lo, hi := 0.0, 1.0
		for range 24 {
			mid := (lo + hi) / 2
			if (Lab{p.L, p.A * mid, p.B * mid}).inGamut() {
				lo = mid
			} else {
				hi = mid
			}
		}
		p.A, p.B = p.A*lo, p.B*lo
	}
	r, g, b := p.linear()
	return RGB{fromLinear(r), fromLinear(g), fromLinear(b)}
}

// Mix interpolates from a (t=0) to b (t=1) in OKLab; t outside 0..1 extrapolates.
func Mix(a, b RGB, t float64) RGB {
	la, lb := a.ToLab(), b.ToLab()
	return Lab{la.L + (lb.L-la.L)*t, la.A + (lb.A-la.A)*t, la.B + (lb.B-la.B)*t}.ToRGB()
}

// WithL returns the color with OKLab lightness L, hue and chroma kept as far as sRGB allows.
func WithL(c RGB, L float64) RGB {
	p := c.ToLab()
	p.L = L
	return p.ToRGB()
}

// ---- WCAG ----

// Luminance is the WCAG relative luminance (0 black .. 1 white).
func Luminance(c RGB) float64 {
	return 0.2126*toLinear(c.R) + 0.7152*toLinear(c.G) + 0.0722*toLinear(c.B)
}

// Contrast is the WCAG contrast ratio (1..21).
func Contrast(a, b RGB) float64 {
	la, lb := Luminance(a), Luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// EnsureContrast moves fg's lightness away from bg until the contrast is at least min
// (towards white on a dark bg, towards black on a light one). Returns fg unchanged when it
// already passes.
func EnsureContrast(fg, bg RGB, min float64) RGB {
	if Contrast(fg, bg) >= min {
		return fg
	}
	p := fg.ToLab()
	step := 0.01
	if Luminance(bg) > 0.18 {
		step = -0.01
	}
	for range 100 {
		p.L += step
		c := p.ToRGB()
		if Contrast(c, bg) >= min || p.L <= 0 || p.L >= 1 {
			return c
		}
	}
	return p.ToRGB()
}

// ReadableOn returns black or white, whichever contrasts more with bg.
func ReadableOn(bg RGB) RGB {
	white, black := RGB{255, 255, 255}, RGB{0, 0, 0}
	if Contrast(white, bg) >= Contrast(black, bg) {
		return white
	}
	return black
}
