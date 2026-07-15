package server

import (
	"errors"
	"testing"
)

// While a rebuild runs, incoming changes must merge into the pending slot
// (kinds escalate, content paths union) instead of overwriting latest-wins;
// otherwise a template change could be dropped by a later content save.
func TestRebuilder_PendingMergeKeepsHigherPriorityKind(t *testing.T) {
	r := NewRebuilder(nil, "")
	r.mu.Lock()
	r.running = true
	r.mu.Unlock()

	if _, res := r.Rebuild(FileChange{Kind: ChangeTemplate, Path: "layouts/base.html"}); res != nil {
		t.Fatal("expected nil result while a rebuild is running")
	}
	if _, res := r.Rebuild(FileChange{Kind: ChangeContent, Path: "content/a.md", Paths: []string{"content/a.md"}}); res != nil {
		t.Fatal("expected nil result while a rebuild is running")
	}

	r.mu.Lock()
	pending := r.pending
	r.mu.Unlock()
	if pending == nil {
		t.Fatal("pending = nil, want merged change")
	}
	if pending.Kind != ChangeTemplate {
		t.Errorf("pending.Kind = %q, want %q (template must survive the merge)", pending.Kind, ChangeTemplate)
	}
	if len(pending.Paths) != 1 || pending.Paths[0] != "content/a.md" {
		t.Errorf("pending.Paths = %#v, want the content path union", pending.Paths)
	}
}

func TestRebuilder_PendingMergeEscalatesContentPlusStatic(t *testing.T) {
	r := NewRebuilder(nil, "")
	r.mu.Lock()
	r.running = true
	r.mu.Unlock()

	r.Rebuild(FileChange{Kind: ChangeStatic, Path: "public/logo.png"})
	r.Rebuild(FileChange{Kind: ChangeContent, Path: "content/a.md", Paths: []string{"content/a.md"}})

	r.mu.Lock()
	pending := r.pending
	r.mu.Unlock()
	if pending == nil {
		t.Fatal("pending = nil, want merged change")
	}
	if pending.Kind != ChangeStatic {
		t.Errorf("pending.Kind = %q, want %q (content+static must escalate to a full build)", pending.Kind, ChangeStatic)
	}
}

func TestToReloadMessage_Error(t *testing.T) {
	change := FileChange{Path: "content/blog/hello.md", Kind: ChangeContent}
	result := &RebuildResult{
		Success: false,
		Error:   errors.New("template parse error in layouts/blog/single.html"),
	}

	msg := ToReloadMessage(change, result, "")

	if msg.Type != ReloadError {
		t.Errorf("Type = %q, want %q", msg.Type, ReloadError)
	}
	if msg.Error == "" {
		t.Error("Error should be non-empty for failed builds")
	}
}

func TestToReloadMessage_CSS(t *testing.T) {
	change := FileChange{Path: "public/style.css", Kind: ChangeCSS}
	result := &RebuildResult{Success: true, PageCount: 10}

	msg := ToReloadMessage(change, result, "")

	if msg.Type != ReloadCSS {
		t.Errorf("Type = %q, want %q", msg.Type, ReloadCSS)
	}
}

func TestToReloadMessage_Content(t *testing.T) {
	change := FileChange{Path: "content/blog/hello.md", Kind: ChangeContent}
	result := &RebuildResult{Success: true, PageCount: 10}

	msg := ToReloadMessage(change, result, "")

	if msg.Type != ReloadFull {
		t.Errorf("Type = %q, want %q", msg.Type, ReloadFull)
	}
}

func TestToReloadMessage_Template(t *testing.T) {
	change := FileChange{Path: "layouts/blog/single.html", Kind: ChangeTemplate}
	result := &RebuildResult{Success: true, PageCount: 10}

	msg := ToReloadMessage(change, result, "")

	if msg.Type != ReloadFull {
		t.Errorf("Type = %q, want %q", msg.Type, ReloadFull)
	}
}

func TestToReloadMessage_Config(t *testing.T) {
	change := FileChange{Path: "sarde.yaml", Kind: ChangeConfig}
	result := &RebuildResult{Success: true, PageCount: 10}

	msg := ToReloadMessage(change, result, "")

	if msg.Type != ReloadFull {
		t.Errorf("Type = %q, want %q", msg.Type, ReloadFull)
	}
}
