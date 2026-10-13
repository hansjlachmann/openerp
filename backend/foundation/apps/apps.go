// Package apps reads application.yaml: an app's identity (BC app.json) and the number series
// its objects and fields may use. The base application has one
// (backend/definitions/application.yaml); every extension app has its own.
package apps

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Object types that have numbers. A number is unique per type (table 18 and page 18 can both
// exist, as in BC).
const (
	Table    = "table"
	Page     = "page"
	Codeunit = "codeunit"
	Report   = "report"
)

// Range is an inclusive number range
type Range struct {
	From int `yaml:"from"`
	To   int `yaml:"to"`
}

// Contains reports whether n is in the range
func (r Range) Contains(n int) bool { return n >= r.From && n <= r.To }

func (r Range) String() string { return fmt.Sprintf("%d–%d", r.From, r.To) }

// Dependency is another app this app needs (an extension depends on the base application)
type Dependency struct {
	ID         string `yaml:"id"`
	Name       string `yaml:"name"`
	MinVersion string `yaml:"min_version"`
}

// App is an application.yaml
type App struct {
	ID           string `yaml:"id"`
	Name         string `yaml:"name"`
	Publisher    string `yaml:"publisher"`
	Contact      string `yaml:"contact"`
	Version      string `yaml:"version"`
	NumberSeries struct {
		Objects []Range `yaml:"objects"`
		Fields  []Range `yaml:"fields"`
	} `yaml:"number_series"`
	// SystemTables are the system table numbers (NAV/BC 2000000000 range), a section of its own
	SystemTables []Range      `yaml:"system_tables"`
	Dependencies []Dependency `yaml:"dependencies"`
}

var guidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Load reads and checks an application.yaml
func Load(path string) (*App, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse reads and checks the content of an application.yaml
func Parse(data []byte) (*App, error) {
	var file struct {
		Application App `yaml:"application"`
	}
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(&file); err != nil {
		return nil, fmt.Errorf("application.yaml: %w", err)
	}
	app := &file.Application
	if err := app.check(); err != nil {
		return nil, fmt.Errorf("application.yaml: %w", err)
	}
	return app, nil
}

func (a *App) check() error {
	if !guidPattern.MatchString(a.ID) {
		return fmt.Errorf("id %q is not a GUID (lower case)", a.ID)
	}
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("name is missing")
	}
	if len(a.NumberSeries.Objects) == 0 || len(a.NumberSeries.Fields) == 0 {
		return fmt.Errorf("number_series needs objects and fields")
	}
	all := map[string][]Range{"objects": a.NumberSeries.Objects, "fields": a.NumberSeries.Fields, "system_tables": a.SystemTables}
	for name, ranges := range all {
		for i, r := range ranges {
			if r.From < 1 || r.To < r.From {
				return fmt.Errorf("%s: range %s is invalid", name, r)
			}
			for _, o := range ranges[:i] {
				if r.From <= o.To && o.From <= r.To {
					return fmt.Errorf("%s: ranges %s and %s overlap", name, o, r)
				}
			}
		}
	}
	return nil
}

func inRanges(ranges []Range, n int) bool {
	for _, r := range ranges {
		if r.Contains(n) {
			return true
		}
	}
	return false
}

// ObjectAllowed reports whether the app may use number id for an object of the type: its
// object series, and for tables also its system tables
func (a *App) ObjectAllowed(objectType string, id int) bool {
	if objectType == Table && inRanges(a.SystemTables, id) {
		return true
	}
	return inRanges(a.NumberSeries.Objects, id)
}

// FieldAllowed reports whether the app may use field number no
func (a *App) FieldAllowed(no int) bool {
	return inRanges(a.NumberSeries.Fields, no)
}

// CheckObject returns an error naming the object when its number is outside the app's series
func (a *App) CheckObject(objectType string, id int, name string) error {
	if a.ObjectAllowed(objectType, id) {
		return nil
	}
	series := a.NumberSeries.Objects
	if objectType == Table {
		series = append(append([]Range{}, series...), a.SystemTables...)
	}
	return fmt.Errorf("%s %d %s is outside the number series of %s (%s)", objectType, id, name, a.Name, rangesText(series))
}

// CheckOverlap returns an error when two apps share numbers: their object, field or system
// table series overlap (every installed app has numbers of its own)
func CheckOverlap(list []*App) error {
	ids := map[string]string{}
	for i, a := range list {
		if other, dup := ids[a.ID]; dup {
			return fmt.Errorf("apps %s and %s have the same id %s", other, a.Name, a.ID)
		}
		ids[a.ID] = a.Name
		for _, b := range list[:i] {
			pairs := []struct {
				what string
				x, y []Range
			}{
				{"object", a.NumberSeries.Objects, b.NumberSeries.Objects},
				{"field", a.NumberSeries.Fields, b.NumberSeries.Fields},
				{"system table", a.SystemTables, b.SystemTables},
			}
			for _, p := range pairs {
				for _, r := range p.x {
					for _, o := range p.y {
						if r.From <= o.To && o.From <= r.To {
							return fmt.Errorf("the %s series %s of %s overlaps %s of %s", p.what, r, a.Name, o, b.Name)
						}
					}
				}
			}
		}
	}
	return nil
}

func rangesText(ranges []Range) string {
	parts := make([]string, len(ranges))
	for i, r := range ranges {
		parts[i] = r.String()
	}
	return strings.Join(parts, ", ")
}

// BasePath is the base application's application.yaml, relative to the project root
const BasePath = "backend/definitions/application.yaml"

// LoadBase reads the base application's application.yaml (the project root is the directory
// with go.mod, searched upwards from the working directory)
func LoadBase() (*App, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return Load(filepath.Join(dir, BasePath))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("project root not found (no go.mod)")
		}
		dir = parent
	}
}
