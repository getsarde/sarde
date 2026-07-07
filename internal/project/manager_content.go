package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/editor"
)

// ListContent returns a list of content files in the given collection.
func (pm *ProjectManager) ListContent(collection string) ([]ContentSummary, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.state == StateClosed {
		return nil, fmt.Errorf("no project open")
	}

	contentDir := pm.contentDir()
	scanner := &content.Scanner{}
	files, err := scanner.DiscoverFiles(contentDir)
	if err != nil {
		return nil, err
	}

	var summaries []ContentSummary
	for _, cf := range files {
		if collection != "" && cf.CollectionName != collection {
			continue
		}
		// Parse frontmatter for summary fields.
		fm, body, err := readContentFile(contentDir, cf.RelPath)
		if err != nil {
			continue
		}
		title, draft, date, weight, _, _ := extractSummary(fm, body)
		summaries = append(summaries, ContentSummary{
			Path:   cf.RelPath,
			Title:  title,
			Draft:  draft,
			Date:   date,
			Order: weight,
		})
	}

	return summaries, nil
}

// ReadContent reads a single content file.
func (pm *ProjectManager) ReadContent(relPath string) (*ContentFile, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.state == StateClosed {
		return nil, fmt.Errorf("no project open")
	}

	contentDir := pm.contentDir()
	if err := validateContentPath(contentDir, relPath); err != nil {
		return nil, err
	}

	fm, body, err := readContentFile(contentDir, relPath)
	if err != nil {
		return nil, err
	}

	title, draft, date, _, wc, rt := extractSummary(fm, body)

	// Determine collection from path.
	collection := ""
	parts := strings.SplitN(filepath.ToSlash(relPath), "/", 2)
	if len(parts) > 1 {
		collection = parts[0]
	}

	return &ContentFile{
		Path:        relPath,
		Title:       title,
		Collection:  collection,
		Frontmatter: fm,
		Body:        body,
		Draft:       draft,
		Date:        date,
		WordCount:   wc,
		ReadingTime: rt,
	}, nil
}

// CreateContent creates a new content file in the given collection.
func (pm *ProjectManager) CreateContent(collection, title string) (*ContentFile, error) {
	pm.mu.Lock()
	if pm.state == StateClosed {
		pm.mu.Unlock()
		return nil, fmt.Errorf("no project open")
	}
	contentDir := pm.contentDir()
	projectDir := pm.projectDir
	pm.mu.Unlock()

	slug := slugify(title)
	if slug == "" {
		return nil, fmt.Errorf("cannot generate slug from title %q", title)
	}

	relPath := filepath.Join(collection, slug+".md")

	if err := validateContentPath(contentDir, relPath); err != nil {
		return nil, err
	}

	absPath := filepath.Join(contentDir, relPath)
	if _, err := os.Stat(absPath); err == nil {
		return nil, fmt.Errorf("file already exists: %s", relPath)
	}

	fm := scaffoldFrontmatter(projectDir, collection, title)
	if err := writeContentFile(contentDir, relPath, fm, "\n"); err != nil {
		return nil, err
	}

	pm.eventHub.Broadcast(Event{Type: "file:created", Data: map[string]any{"path": filepath.ToSlash(relPath)}})

	return &ContentFile{
		Path:        filepath.ToSlash(relPath),
		Title:       title,
		Collection:  collection,
		Frontmatter: fm,
		Body:        "\n",
		Draft:       true,
		Date:        time.Now(),
	}, nil
}

// SaveContent writes frontmatter and body to an existing content file.
func (pm *ProjectManager) SaveContent(relPath string, fm map[string]any, body string) error {
	pm.mu.Lock()
	if pm.state == StateClosed {
		pm.mu.Unlock()
		return fmt.Errorf("no project open")
	}
	contentDir := pm.contentDir()
	pm.mu.Unlock()

	if err := validateContentPath(contentDir, relPath); err != nil {
		return err
	}

	if err := writeContentFile(contentDir, relPath, fm, body); err != nil {
		return err
	}

	pm.eventHub.Broadcast(Event{Type: "file:changed", Data: map[string]any{"path": relPath}})
	return nil
}

// ListRevisions returns the revision history for a content file, newest first.
func (pm *ProjectManager) ListRevisions(relPath string) ([]RevisionSummary, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.state == StateClosed {
		return nil, fmt.Errorf("no project open")
	}
	contentDir := pm.contentDir()
	if err := validateContentPath(contentDir, relPath); err != nil {
		return nil, err
	}
	absPath := filepath.Join(contentDir, filepath.FromSlash(relPath))
	revs := editor.ListRevisions(absPath)

	out := make([]RevisionSummary, 0, len(revs))
	for _, r := range revs {
		out = append(out, RevisionSummary{
			ID:        filepath.Base(r.Path),
			Timestamp: r.Timestamp,
			Size:      r.Size,
		})
	}
	return out, nil
}

// RestoreRevision overwrites the content file with the named revision's contents.
// The current version is snapshotted first so the restore is itself reversible.
func (pm *ProjectManager) RestoreRevision(relPath, revisionID string) error {
	pm.mu.Lock()
	if pm.state == StateClosed {
		pm.mu.Unlock()
		return fmt.Errorf("no project open")
	}
	contentDir := pm.contentDir()
	pm.mu.Unlock()

	if err := validateContentPath(contentDir, relPath); err != nil {
		return err
	}
	if strings.ContainsAny(revisionID, "/\\") || strings.Contains(revisionID, "..") {
		return fmt.Errorf("invalid revision id")
	}
	absPath := filepath.Join(contentDir, filepath.FromSlash(relPath))
	revPath := filepath.Join(filepath.Dir(absPath), ".revisions", revisionID)
	if _, err := os.Stat(revPath); err != nil {
		return fmt.Errorf("revision not found")
	}
	if err := editor.RestoreRevision(absPath, revPath); err != nil {
		return err
	}
	pm.eventHub.Broadcast(Event{Type: "file:changed", Data: map[string]any{"path": relPath}})
	return nil
}

// DeleteContent removes a content file.
func (pm *ProjectManager) DeleteContent(relPath string) error {
	pm.mu.Lock()
	if pm.state == StateClosed {
		pm.mu.Unlock()
		return fmt.Errorf("no project open")
	}
	contentDir := pm.contentDir()
	pm.mu.Unlock()

	if err := validateContentPath(contentDir, relPath); err != nil {
		return err
	}

	absPath := filepath.Join(contentDir, filepath.FromSlash(relPath))
	if err := os.Remove(absPath); err != nil {
		return err
	}

	pm.eventHub.Broadcast(Event{Type: "file:deleted", Data: map[string]any{"path": relPath}})
	return nil
}

// RenameContent moves a content file to a new path.
func (pm *ProjectManager) RenameContent(oldPath, newPath string) error {
	pm.mu.Lock()
	if pm.state == StateClosed {
		pm.mu.Unlock()
		return fmt.Errorf("no project open")
	}
	contentDir := pm.contentDir()
	pm.mu.Unlock()

	if err := validateContentPath(contentDir, oldPath); err != nil {
		return err
	}
	if err := validateContentPath(contentDir, newPath); err != nil {
		return err
	}

	absOld := filepath.Join(contentDir, filepath.FromSlash(oldPath))
	absNew := filepath.Join(contentDir, filepath.FromSlash(newPath))

	if err := os.MkdirAll(filepath.Dir(absNew), 0o755); err != nil {
		return err
	}
	if err := os.Rename(absOld, absNew); err != nil {
		return err
	}

	pm.eventHub.Broadcast(Event{Type: "file:renamed", Data: map[string]any{"oldPath": oldPath, "newPath": newPath}})
	return nil
}
