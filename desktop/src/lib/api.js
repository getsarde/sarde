// Coderoo Desktop — Tauri IPC API client.
// All calls go directly to the Rust backend via invoke().

import { invoke } from '@tauri-apps/api/core'
import { listen } from '@tauri-apps/api/event'

// ---------------------------------------------------------------------------
// Project lifecycle
// ---------------------------------------------------------------------------

export const projectOpen = (dir) => invoke('open_project', { dir })
export const projectCreate = (dir, title = '') => invoke('create_project', { dir, title })
export const projectClose = () => invoke('close_project')
export const projectInfo = () => invoke('get_project_info')
export const listRecentProjects = () => invoke('list_recent_projects')

// ---------------------------------------------------------------------------
// Content CRUD
// ---------------------------------------------------------------------------

export const listContent = (collection = null) => invoke('list_content', { collection })
export const readContent = (path) => invoke('read_content', { path })
export const saveContent = (path, frontmatter, body) => invoke('save_content', { path, frontmatter, body })
export const createContent = (collection, title) => invoke('create_content', { collection, title })
export const deleteContent = (path) => invoke('delete_content', { path })
export const renameContent = (oldPath, newPath) => invoke('rename_content', { oldPath, newPath })

// ---------------------------------------------------------------------------
// Config & schema
// ---------------------------------------------------------------------------

export const getConfig = () => invoke('get_config')
export const updateConfig = (settings) => invoke('update_config', { settings })
export const getCollections = () => invoke('get_collections')
export const fetchSchema = (collection) => invoke('get_schema', { collection })

// ---------------------------------------------------------------------------
// Build & preview
// ---------------------------------------------------------------------------

export const build = () => invoke('run_build')
export const startPreview = () => invoke('start_preview')
export const stopPreview = () => invoke('stop_preview')

// ---------------------------------------------------------------------------
// Tauri event listeners (replaces WebSocket connectEvents)
// ---------------------------------------------------------------------------

export const onBuildLog = (cb) => listen('build:log', (e) => cb(e.payload))
export const onBuildComplete = (cb) => listen('build:complete', (e) => cb(e.payload))
export const onBuildError = (cb) => listen('build:error', (e) => cb(e.payload))
export const onPreviewReady = (cb) => listen('preview:ready', (e) => cb(e.payload))
export const onPreviewStopped = (cb) => listen('preview:stopped', (e) => cb(e.payload))

// ---------------------------------------------------------------------------
// Validate, deploy, import
// ---------------------------------------------------------------------------

export const validate = () => invoke('validate_project')
export const deploy = (provider = null) => invoke('deploy', { provider })
export const importObsidian = (vaultPath, collection = '') => invoke('import_obsidian', { vaultPath, collection })

// Not yet available — requires Go-side markdown rendering endpoint.
export const renderMarkdown = (markdown) => Promise.reject(new Error('Not yet implemented'))
