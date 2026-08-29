package cfgutil

import "testing"

func TestParseFields_PreservesOptionsAndNested(t *testing.T) {
	raw := []byte(`
fields:
  mode:
    type: select
    label: Mode
    default: a
    options:
      - { value: a, label: A }
      - { value: b, label: B }
  size:
    type: number
    default: 3
    min: 1
    max: 10
  fonts:
    type: object
    fields:
      regular: { type: text, default: "" }
`)
	fields, err := ParseFields(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 3 || fields[0].Name != "fonts" || fields[1].Name != "mode" || fields[2].Name != "size" {
		t.Fatalf("unexpected order/count: %+v", fields)
	}
	if len(fields[1].Options) != 2 || fields[1].Options[1].Label != "B" {
		t.Fatalf("options not preserved: %+v", fields[1])
	}
	if fields[2].Min == nil || *fields[2].Min != 1 || fields[2].Max == nil || *fields[2].Max != 10 {
		t.Fatalf("min/max not preserved: %+v", fields[2])
	}
	if len(fields[0].Fields) != 1 || fields[0].Fields[0].Name != "regular" {
		t.Fatalf("nested fields not preserved: %+v", fields[0])
	}
	empty, err := ParseFields([]byte("fields: {}"))
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty doc should give empty slice, got %v %v", empty, err)
	}
}
