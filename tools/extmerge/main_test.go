package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const coreCustomer = `table:
  id: 18
  name: "Customer"
  fields:
    - name: no
      no: 1
      type: types.Code
      db_name: no
      primary_key: true
      length: 20
    - name: name
      no: 2
      type: types.Text
      length: 100
`

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func mergeCustomer(t *testing.T, extension string) (map[string]interface{}, error) {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "core", "tables", "customer.yaml"), coreCustomer)
	writeFile(t, filepath.Join(dir, "ext", "tables", "customer.extend.yaml"), extension)
	out := filepath.Join(dir, "out")
	if err := run(filepath.Join(dir, "core"), filepath.Join(dir, "ext"), out, false); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(out, "tables", "customer.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var def map[string]interface{}
	if err := yaml.Unmarshal(data, &def); err != nil {
		t.Fatal(err)
	}
	return def, nil
}

// An extension field is added with every key it has (not only name/type/length/caption).
func TestTableExtensionKeepsFieldKeys(t *testing.T) {
	def, err := mergeCustomer(t, `extends_table: 18
fields:
  - name: loyalty_points
    no: 50000
    type: int
    db_name: loyalty_points
    default: 100
    validation: {min: 0, max: 10000}
`)
	if err != nil {
		t.Fatal(err)
	}
	fields := def["table"].(map[string]interface{})["fields"].([]interface{})
	if len(fields) != 3 {
		t.Fatalf("%d fields, want 3", len(fields))
	}
	added := fields[2].(map[string]interface{})
	if added["no"] != 50000 || added["default"] != 100 || added["validation"] == nil || added["db_name"] != "loyalty_points" {
		t.Errorf("added field %v", added)
	}
}

func TestTableExtensionFieldRules(t *testing.T) {
	cases := map[string]struct{ ext, want string }{
		"no number": {`extends_table: 18
fields:
  - name: loyalty_points
    type: int
`, "needs a field number"},
		"base range": {`extends_table: 18
fields:
  - name: loyalty_points
    no: 999
    type: int
`, "needs a field number"},
		"number used by the table": {`extends_table: 18
fields:
  - name: loyalty_points
    no: 50000
    type: int
  - name: tier
    no: 50000
    type: types.Code
`, "number 50000 is already used by field loyalty_points"},
		"name used by the table": {`extends_table: 18
fields:
  - name: Name
    no: 50001
    type: types.Text
`, "already has a field named Name"},
	}
	for name, c := range cases {
		if _, err := mergeCustomer(t, c.ext); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want %q", name, err, c.want)
		}
	}
}
