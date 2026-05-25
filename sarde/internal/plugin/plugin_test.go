package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
)

func TestManager_RegisterAndRun(t *testing.T) {
	mgr := NewManager()

	var order []string
	mgr.Register(&Plugin{
		Name: "a",
		Hooks: PluginHooks{
			ConfigSetup: func(ctx *ConfigSetupContext) error {
				order = append(order, "a")
				return nil
			},
		},
	})
	mgr.Register(&Plugin{
		Name: "b",
		Hooks: PluginHooks{
			ConfigSetup: func(ctx *ConfigSetupContext) error {
				order = append(order, "b")
				return nil
			},
		},
	})

	cfg := config.Defaults()
	if err := mgr.RunConfigSetup(cfg); err != nil {
		t.Fatalf("RunConfigSetup failed: %v", err)
	}

	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Errorf("expected [a, b], got %v", order)
	}
}

func TestManager_NilHooksSkipped(t *testing.T) {
	mgr := NewManager()
	mgr.Register(&Plugin{
		Name:  "empty",
		Hooks: PluginHooks{}, // all nil
	})

	cfg := config.Defaults()
	if err := mgr.RunConfigSetup(cfg); err != nil {
		t.Fatalf("should not error on nil hooks: %v", err)
	}

	var pages []*engine.Page
	if err := mgr.RunContentLoaded(cfg, nil, &pages); err != nil {
		t.Fatalf("should not error on nil hooks: %v", err)
	}

	if err := mgr.RunBeforeRender(cfg, &engine.Page{}, &engine.RouteData{}, &engine.SiteContext{}); err != nil {
		t.Fatalf("should not error on nil hooks: %v", err)
	}

	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{Config: cfg, OutputDir: t.TempDir()}
	ctx.SetWarnings(&warnings)
	if err := mgr.RunBuildDone(ctx); err != nil {
		t.Fatalf("should not error on nil hooks: %v", err)
	}
}

func TestManager_BuildDoneParallel(t *testing.T) {
	mgr := NewManager()
	var counter int64

	for i := 0; i < 5; i++ {
		mgr.Register(&Plugin{
			Name: "parallel",
			Hooks: PluginHooks{
				BuildDone: func(ctx *BuildDoneContext) error {
					atomic.AddInt64(&counter, 1)
					return nil
				},
			},
		})
	}

	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: t.TempDir(),
	}
	ctx.SetWarnings(&warnings)

	if err := mgr.RunBuildDone(ctx); err != nil {
		t.Fatalf("RunBuildDone failed: %v", err)
	}

	if atomic.LoadInt64(&counter) != 5 {
		t.Errorf("expected 5 plugins to run, got %d", counter)
	}
}

func TestManager_BuildDoneParallelWarningsUseSharedLock(t *testing.T) {
	mgr := NewManager()
	for i := 0; i < 2; i++ {
		i := i
		mgr.Register(&Plugin{
			Name: fmt.Sprintf("warning-%d", i),
			Hooks: PluginHooks{
				BuildDone: func(ctx *BuildDoneContext) error {
					for n := 0; n < 500; n++ {
						ctx.AddWarning(engine.ValidationWarning{
							File:    fmt.Sprintf("%d-%d.md", i, n),
							Field:   "test",
							Message: "warning",
						})
					}
					return nil
				},
			},
		})
	}

	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: t.TempDir(),
	}
	ctx.SetWarnings(&warnings)

	if err := mgr.RunBuildDone(ctx); err != nil {
		t.Fatalf("RunBuildDone failed: %v", err)
	}
	if len(warnings) != 1000 {
		t.Fatalf("warnings = %d, want 1000", len(warnings))
	}
}

func TestManager_ContentLoaded_InjectPage(t *testing.T) {
	mgr := NewManager()
	mgr.Register(&Plugin{
		Name: "injector",
		Hooks: PluginHooks{
			ContentLoaded: func(ctx *ContentLoadedContext) error {
				ctx.InjectPage(&engine.Page{
					Title: "Virtual Page",
					Slug:  "virtual",
				})
				return nil
			},
		},
	})

	cfg := config.Defaults()
	pages := []*engine.Page{
		{Title: "Real Page", Slug: "real"},
	}

	if err := mgr.RunContentLoaded(cfg, nil, &pages); err != nil {
		t.Fatalf("RunContentLoaded failed: %v", err)
	}

	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
	if pages[1].Title != "Virtual Page" {
		t.Errorf("injected page title = %q, want Virtual Page", pages[1].Title)
	}
}

func TestManager_ConfigSetup_TemplateFuncs(t *testing.T) {
	mgr := NewManager()
	mgr.Register(&Plugin{
		Name: "custom-funcs",
		Hooks: PluginHooks{
			ConfigSetup: func(ctx *ConfigSetupContext) error {
				ctx.AddTemplateFunc("myFunc", func() string { return "hello" })
				return nil
			},
		},
	})

	cfg := config.Defaults()
	if err := mgr.RunConfigSetup(cfg); err != nil {
		t.Fatalf("RunConfigSetup failed: %v", err)
	}

	funcs := mgr.TemplateFuncs()
	if funcs["myFunc"] == nil {
		t.Error("expected myFunc to be registered")
	}
}

func TestManager_RegisterBuiltins(t *testing.T) {
	mgr := NewManager()
	mgr.RegisterBuiltins([]string{"sitemap", "robots"}, nil)

	if len(mgr.plugins) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(mgr.plugins))
	}
	if mgr.plugins[0].Name != "sitemap" {
		t.Errorf("first plugin = %q, want sitemap", mgr.plugins[0].Name)
	}
	if mgr.plugins[1].Name != "robots" {
		t.Errorf("second plugin = %q, want robots", mgr.plugins[1].Name)
	}
}

func TestManager_RegisterBuiltins_UnknownSkipped(t *testing.T) {
	mgr := NewManager()
	mgr.RegisterBuiltins([]string{"sitemap", "nonexistent", "robots"}, nil)

	if len(mgr.plugins) != 2 {
		t.Errorf("expected 2 plugins (unknown skipped), got %d", len(mgr.plugins))
	}
}

func TestManager_SharedStore(t *testing.T) {
	mgr := NewManager()

	mgr.Register(&Plugin{
		Name: "producer",
		Hooks: PluginHooks{
			ConfigSetup: func(ctx *ConfigSetupContext) error {
				ctx.Set("key", "value-from-producer")
				return nil
			},
		},
	})
	mgr.Register(&Plugin{
		Name: "consumer",
		Hooks: PluginHooks{
			ContentLoaded: func(ctx *ContentLoadedContext) error {
				val := ctx.Get("key")
				if val != "value-from-producer" {
					t.Errorf("expected value-from-producer, got %v", val)
				}
				return nil
			},
		},
	})

	cfg := config.Defaults()
	mgr.RunConfigSetup(cfg)

	var pages []*engine.Page
	mgr.RunContentLoaded(cfg, nil, &pages)
}

func TestManager_SharedStoreConcurrentBeforeRender(t *testing.T) {
	mgr := NewManager()
	mgr.Register(&Plugin{
		Name: "store",
		Hooks: PluginHooks{
			BeforeRender: func(ctx *BeforeRenderContext) error {
				ctx.Set("key", "value")
				if got := ctx.Get("key"); got != "value" {
					t.Errorf("shared store value = %v, want value", got)
				}
				return nil
			},
		},
	})

	cfg := config.Defaults()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := mgr.RunBeforeRender(cfg, &engine.Page{}, &engine.RouteData{}, &engine.SiteContext{}); err != nil {
				t.Errorf("RunBeforeRender failed: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestBuildDoneContext_WriteFile(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
	}
	ctx.SetWarnings(&warnings)

	err := ctx.WriteFile("test/output.txt", []byte("hello"))
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify file exists.
	data, err := readTestFile(outDir, "test/output.txt")
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("file content = %q, want hello", string(data))
	}
}

func TestBuildDoneContext_WriteFileRejectsTraversal(t *testing.T) {
	parent := t.TempDir()
	outDir := filepath.Join(parent, "dist")
	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
	}

	if err := ctx.WriteFile("../escape.txt", []byte("bad")); err == nil {
		t.Fatal("expected traversal write to fail")
	}
	if _, err := os.Stat(filepath.Join(parent, "escape.txt")); !os.IsNotExist(err) {
		t.Fatalf("plugin write escaped output dir, stat err = %v", err)
	}
}

func TestBuildDoneContext_AddWarning(t *testing.T) {
	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{Config: config.Defaults(), OutputDir: t.TempDir()}
	ctx.SetWarnings(&warnings)

	ctx.AddWarning(engine.ValidationWarning{File: "test.md", Field: "title"})
	ctx.AddWarning(engine.ValidationWarning{File: "other.md", Field: "date"})

	if len(warnings) != 2 {
		t.Errorf("expected 2 warnings, got %d", len(warnings))
	}
}

func readTestFile(base, rel string) ([]byte, error) {
	return os.ReadFile(filepath.Join(base, filepath.FromSlash(rel)))
}
