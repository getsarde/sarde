package filetree

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestClose_Aliases(t *testing.T) {
	for _, closer := range []string{":::", ":::/file-tree", ":::/filetree", ":::/FileTree"} {
		gm := goldmark.New(goldmark.WithExtensions(&Extension{}))
		var buf bytes.Buffer
		if err := gm.Convert([]byte(":::file-tree\n- src/\n  - main.go\n"+closer+"\nafter\n"), &buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if strings.Contains(out, closer) {
			t.Errorf("closer %q rendered as text:\n%s", closer, out)
		}
		if i, j := strings.LastIndex(out, "</div>"), strings.Index(out, "<p>after</p>"); j < i {
			t.Errorf("closer %q did not close the file tree:\n%s", closer, out)
		}
	}
}
