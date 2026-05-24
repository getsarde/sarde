package codeblock

import (
	"testing"
)

func TestParseInfoString(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantLang        string
		wantTitle       string
		wantHighlight   map[int]bool
		wantInserted    map[int]bool
		wantDeleted     map[int]bool
		wantTerminal    bool
		wantLineNumbers bool
	}{
		{
			name:     "language only",
			input:    "js",
			wantLang: "js",
		},
		{
			name:      "language with title",
			input:     `js title="app.js"`,
			wantLang:  "js",
			wantTitle: "app.js",
		},
		{
			name:          "language with highlight lines",
			input:         `js {2-5,8}`,
			wantLang:      "js",
			wantHighlight: map[int]bool{2: true, 3: true, 4: true, 5: true, 8: true},
		},
		{
			name:         "language with ins and del",
			input:        `js ins={2} del={3}`,
			wantLang:     "js",
			wantInserted: map[int]bool{2: true},
			wantDeleted:  map[int]bool{3: true},
		},
		{
			name:         "terminal language",
			input:        "bash",
			wantLang:     "bash",
			wantTerminal: true,
		},
		{
			name:            "showLineNumbers flag",
			input:           `js showLineNumbers`,
			wantLang:        "js",
			wantLineNumbers: true,
		},
		{
			name:     "empty string",
			input:    "",
			wantLang: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseInfoString(tt.input)

			if got.Language != tt.wantLang {
				t.Errorf("Language = %q, want %q", got.Language, tt.wantLang)
			}
			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
			if got.IsTerminal != tt.wantTerminal {
				t.Errorf("IsTerminal = %v, want %v", got.IsTerminal, tt.wantTerminal)
			}
			if got.ShowLineNumbers != tt.wantLineNumbers {
				t.Errorf("ShowLineNumbers = %v, want %v", got.ShowLineNumbers, tt.wantLineNumbers)
			}

			if tt.wantHighlight != nil {
				assertLineSet(t, "HighlightLines", got.HighlightLines, tt.wantHighlight)
			}
			if tt.wantInserted != nil {
				assertLineSet(t, "InsertedLines", got.InsertedLines, tt.wantInserted)
			}
			if tt.wantDeleted != nil {
				assertLineSet(t, "DeletedLines", got.DeletedLines, tt.wantDeleted)
			}
		})
	}
}

func assertLineSet(t *testing.T, label string, got, want map[int]bool) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got %d entries, want %d (got=%v, want=%v)", label, len(got), len(want), got, want)
		return
	}
	for k := range want {
		if !got[k] {
			t.Errorf("%s: missing line %d", label, k)
		}
	}
}
