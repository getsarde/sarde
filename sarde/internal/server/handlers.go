package server

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/frostybee/sarde/internal/build"
	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/deploy"
	"github.com/frostybee/sarde/internal/importer"
	"github.com/frostybee/sarde/internal/project"
)

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Project lifecycle
// ---------------------------------------------------------------------------

func (s *APIServer) handleProjectOpen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dir string `json:"dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	if req.Dir == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "dir is required")
		return
	}

	info, err := s.pm.OpenProject(req.Dir)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PATH", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *APIServer) handleProjectCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dir   string `json:"dir"`
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	if req.Dir == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "dir is required")
		return
	}

	info, err := s.pm.CreateProject(req.Dir, project.CreateOpts{Title: req.Title})
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PATH", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, info)
}

func (s *APIServer) handleProjectClose(w http.ResponseWriter, r *http.Request) {
	if err := s.pm.CloseProject(); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (s *APIServer) handleProjectInfo(w http.ResponseWriter, r *http.Request) {
	cfg := s.pm.GetConfig()
	if cfg == nil {
		writeError(w, http.StatusBadRequest, "PROJECT_NOT_OPEN", "no project is open")
		return
	}

	state := s.pm.State()
	cols, _ := s.pm.GetCollections()

	writeJSON(w, http.StatusOK, map[string]any{
		"state":       state.String(),
		"title":       cfg.Site.Title,
		"collections": cols,
	})
}

// ---------------------------------------------------------------------------
// Content CRUD
// ---------------------------------------------------------------------------

func (s *APIServer) handleListContent(w http.ResponseWriter, r *http.Request) {
	collection := r.URL.Query().Get("collection")
	summaries, err := s.pm.ListContent(collection)
	if err != nil {
		writeError(w, http.StatusBadRequest, "PROJECT_NOT_OPEN", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summaries)
}

func (s *APIServer) handleCreateContent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Collection string `json:"collection"`
		Title      string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	if req.Collection == "" || req.Title == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "collection and title are required")
		return
	}

	file, err := s.pm.CreateContent(req.Collection, req.Title)
	if err != nil {
		writeError(w, http.StatusBadRequest, "FILE_EXISTS", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, file)
}

func (s *APIServer) handleReadContent(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "path is required")
		return
	}

	file, err := s.pm.ReadContent(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "FILE_NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, file)
}

func (s *APIServer) handleSaveContent(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "path is required")
		return
	}

	var req struct {
		Frontmatter map[string]any `json:"frontmatter"`
		Body        string         `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}

	if err := s.pm.SaveContent(path, req.Frontmatter, req.Body); err != nil {
		writeError(w, http.StatusBadRequest, "FILE_NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (s *APIServer) handleDeleteContent(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "path is required")
		return
	}

	if err := s.pm.DeleteContent(path); err != nil {
		writeError(w, http.StatusNotFound, "FILE_NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (s *APIServer) handleListRevisions(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "path is required")
		return
	}
	revs, err := s.pm.ListRevisions(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "FILE_NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, revs)
}

func (s *APIServer) handleRestoreRevision(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "path is required")
		return
	}
	var req struct {
		RevisionID string `json:"revisionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RevisionID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "revisionId is required")
		return
	}
	if err := s.pm.RestoreRevision(path, req.RevisionID); err != nil {
		writeError(w, http.StatusBadRequest, "FILE_NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (s *APIServer) handleRenameContent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPath string `json:"oldPath"`
		NewPath string `json:"newPath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	if req.OldPath == "" || req.NewPath == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "oldPath and newPath are required")
		return
	}

	if err := s.pm.RenameContent(req.OldPath, req.NewPath); err != nil {
		writeError(w, http.StatusBadRequest, "FILE_NOT_FOUND", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

// ---------------------------------------------------------------------------
// Build & Preview
// ---------------------------------------------------------------------------

func (s *APIServer) handleBuild(w http.ResponseWriter, r *http.Request) {
	result, err := s.pm.Build()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "BUILD_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"pageCount": result.PageCount,
		"duration":  result.Duration.String(),
		"outputDir": result.OutputDir,
	})
}

func (s *APIServer) handleValidate(w http.ResponseWriter, r *http.Request) {
	result, err := s.pm.Validate()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "BUILD_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *APIServer) handlePreviewStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Port int `json:"port"`
	}
	json.NewDecoder(r.Body).Decode(&req) // optional body

	port, err := s.pm.StartPreview(req.Port)
	if err != nil {
		writeError(w, http.StatusBadRequest, "PREVIEW_RUNNING", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"port": port,
		"url":  "http://localhost:" + strconv.Itoa(port),
	})
}

func (s *APIServer) handlePreviewStop(w http.ResponseWriter, r *http.Request) {
	if err := s.pm.StopPreview(); err != nil {
		writeError(w, http.StatusBadRequest, "PREVIEW_NOT_RUNNING", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

func (s *APIServer) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.pm.GetConfig()
	if cfg == nil {
		writeError(w, http.StatusBadRequest, "PROJECT_NOT_OPEN", "no project is open")
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *APIServer) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var input project.SettingsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}

	if err := s.pm.UpdateSettings(input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONFIG", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (s *APIServer) handleGetCollections(w http.ResponseWriter, r *http.Request) {
	cols, err := s.pm.GetCollections()
	if err != nil {
		writeError(w, http.StatusBadRequest, "PROJECT_NOT_OPEN", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cols)
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

func (s *APIServer) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	collection := r.PathValue("collection")
	if collection == "" {
		writeJSON(w, http.StatusOK, nil)
		return
	}

	schema, err := s.pm.GetSchema(collection)
	if err != nil {
		writeError(w, http.StatusBadRequest, "SCHEMA_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, schema)
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

func (s *APIServer) handleRenderMarkdown(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Markdown string `json:"markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}

	result, err := s.pm.RenderMarkdown(req.Markdown)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// Deploy
// ---------------------------------------------------------------------------

func (s *APIServer) handleDeploy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"` // optional override
	}
	json.NewDecoder(r.Body).Decode(&req) // body is optional

	cfg := s.pm.GetConfig()
	if cfg == nil {
		writeError(w, http.StatusBadRequest, "PROJECT_NOT_OPEN", "no project is open")
		return
	}

	deployCfg := cfg.Deploy
	if req.Provider != "" {
		deployCfg.Provider = req.Provider
	}

	deployer, err := deploy.NewDeployer(deployCfg)
	if err != nil {
		writeError(w, http.StatusBadRequest, "DEPLOY_CONFIG_ERROR", err.Error())
		return
	}

	outputDir := cfg.Build.Output
	projectDir := s.pm.ProjectDir()
	outputDir, err = build.ResolveOutputDir(projectDir, outputDir)
	if err != nil {
		writeError(w, http.StatusBadRequest, "OUTPUT_CONFIG_ERROR", err.Error())
		return
	}

	if err := deployer.Deploy(outputDir); err != nil {
		writeError(w, http.StatusInternalServerError, "DEPLOY_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "deployed",
		"provider": deployer.Name(),
	})
}

// ---------------------------------------------------------------------------
// Import
// ---------------------------------------------------------------------------

func (s *APIServer) handleImportObsidian(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VaultPath  string `json:"vault_path"`
		Collection string `json:"collection"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	if req.VaultPath == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "vault_path is required")
		return
	}

	projectDir := s.pm.ProjectDir()
	if projectDir == "" {
		writeError(w, http.StatusBadRequest, "PROJECT_NOT_OPEN", "no project is open")
		return
	}

	collection := req.Collection
	if collection == "" {
		collection = filepath.Base(req.VaultPath)
	}

	contentDir := filepath.Join(projectDir, consts.DirContent)
	result, err := importer.ImportObsidian(req.VaultPath, collection, contentDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "IMPORT_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
