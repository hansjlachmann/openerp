package i18n

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// Every translation file must be valid YAML: the loader only logs a warning for a file it
// cannot parse, and then every key of that file shows as its raw name in the UI.
func TestTranslationFilesAreValidYAML(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "..", "translations", "*", "*.yaml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no translation files found (%v)", err)
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var v map[string]interface{}
		if err := yaml.Unmarshal(raw, &v); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
}
