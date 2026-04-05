package server

import (
	"errors"
	"testing"
)

func TestToReloadMessage_Error(t *testing.T) {
	change := FileChange{Path: "content/blog/hello.md", Kind: ChangeContent}
	result := &RebuildResult{
		Success: false,
		Error:   errors.New("template parse error in layouts/blog/single.html"),
	}

	msg := ToReloadMessage(change, result)

	if msg.Type != ReloadError {
		t.Errorf("Type = %q, want %q", msg.Type, ReloadError)
	}
	if msg.Error == "" {
		t.Error("Error should be non-empty for failed builds")
	}
}

func TestToReloadMessage_CSS(t *testing.T) {
	change := FileChange{Path: "static/style.css", Kind: ChangeCSS}
	result := &RebuildResult{Success: true, PageCount: 10}

	msg := ToReloadMessage(change, result)

	if msg.Type != ReloadCSS {
		t.Errorf("Type = %q, want %q", msg.Type, ReloadCSS)
	}
}

func TestToReloadMessage_Content(t *testing.T) {
	change := FileChange{Path: "content/blog/hello.md", Kind: ChangeContent}
	result := &RebuildResult{Success: true, PageCount: 10}

	msg := ToReloadMessage(change, result)

	if msg.Type != ReloadFull {
		t.Errorf("Type = %q, want %q", msg.Type, ReloadFull)
	}
}

func TestToReloadMessage_Template(t *testing.T) {
	change := FileChange{Path: "layouts/blog/single.html", Kind: ChangeTemplate}
	result := &RebuildResult{Success: true, PageCount: 10}

	msg := ToReloadMessage(change, result)

	if msg.Type != ReloadFull {
		t.Errorf("Type = %q, want %q", msg.Type, ReloadFull)
	}
}

func TestToReloadMessage_Config(t *testing.T) {
	change := FileChange{Path: "site.yaml", Kind: ChangeConfig}
	result := &RebuildResult{Success: true, PageCount: 10}

	msg := ToReloadMessage(change, result)

	if msg.Type != ReloadFull {
		t.Errorf("Type = %q, want %q", msg.Type, ReloadFull)
	}
}
