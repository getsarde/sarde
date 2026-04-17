// Coderoo Desktop — shared reactive state (Svelte 5 runes)
// This file uses the .svelte.js extension to enable rune syntax at module level.

// ---------------------------------------------------------------------------
// Preview server state (Go `coderoo serve` process managed by Rust)
// ---------------------------------------------------------------------------
export const preview = $state({
  port: 0,        // port the preview server is listening on (0 = not running)
  running: false,
})

// ---------------------------------------------------------------------------
// Build log — captures output from Go sidecar during build/serve
// ---------------------------------------------------------------------------
export const buildLog = $state({
  entries: [],     // { timestamp, text, type }[]
  visible: false,
  building: false,
})

const MAX_LOG_ENTRIES = 500

function pushLog(text, type = 'info') {
  buildLog.entries.push({ timestamp: Date.now(), text, type })
  if (buildLog.entries.length > MAX_LOG_ENTRIES) {
    buildLog.entries.splice(0, buildLog.entries.length - MAX_LOG_ENTRIES)
  }
}

export function clearBuildLog() {
  buildLog.entries.length = 0
}

export function toggleBuildLog() {
  buildLog.visible = !buildLog.visible
}

/** Register Tauri event listeners that keep preview + build state in sync. Call once on mount. */
export function setupPreviewListeners() {
  onPreviewReady((port) => {
    preview.port = port
    preview.running = true
    pushLog(`Preview server ready on port ${port}`, 'success')
    addToast('success', `Preview running at localhost:${port}`)
  })
  onPreviewStopped(() => {
    preview.port = 0
    preview.running = false
    pushLog('Preview server stopped', 'info')
  })
  onPreviewCrashed((code) => {
    preview.port = 0
    preview.running = false
    pushLog(`Preview server crashed (exit code: ${code})`, 'error')
    addToast('error', 'Preview server crashed')
  })
  onBuildLog((msg) => {
    pushLog(msg, 'info')
  })
  onBuildComplete((result) => {
    buildLog.building = false
    const text = result?.duration
      ? `Build complete in ${result.duration}`
      : 'Build complete'
    pushLog(text, 'success')
  })
  onBuildError((err) => {
    buildLog.building = false
    const msg = typeof err === 'string' ? err : err?.message ?? 'Unknown error'
    pushLog(`Build error: ${msg}`, 'error')
    addToast('error', `Build failed: ${msg}`)
  })
}

// ---------------------------------------------------------------------------
// Tabs — each item: { id, name, path, dirty, cachedContent }
// ---------------------------------------------------------------------------
export const tabs = $state({
  items: [],
  activeId: null,
})

// ---------------------------------------------------------------------------
// UI panel visibility and overlay state
// ---------------------------------------------------------------------------
export const ui = $state({
  leftPanel: 'files',       // 'files' | 'search' | 'git' | null
  rightPanel: 'toc',        // 'toc' | 'properties' | 'assets' | 'stats' | null
  propertiesMode: 'form',   // 'form' | 'yaml'
  previewMode: 'editor',    // 'editor' | 'split' | 'preview'
  settingsOpen: false,
  settingsSection: 'general',
  commandPaletteOpen: false,
  deployOpen: false,
  importOpen: false,
})

// ---------------------------------------------------------------------------
// Markdown preview state
// ---------------------------------------------------------------------------
export const mdPreview = $state({
  html: '',
  rendering: false,
  error: null,
})

// ---------------------------------------------------------------------------
// Active document state — populated by the editor component
// ---------------------------------------------------------------------------
export const doc = $state({
  content: '',
  filePath: '',
  dirty: false,
  wordCount: 0,
  readingTime: 0,
  cursorLine: 1,
  cursorCol: 1,
  targetLine: 0,  // set by search panel; CodeEditor scrolls to this line then resets to 0
  externalUpdate: 0, // bumped by PropertiesPanel to signal CodeEditor to reload content
  insertText: '',    // set by MediaPanel; CodeEditor inserts at cursor then clears
})

// ---------------------------------------------------------------------------
// Toast notifications — array of { id, type, message, duration }
// ---------------------------------------------------------------------------
export const toasts = $state({
  items: [],
})

let _toastId = 0

/**
 * Push a toast notification.
 * @param {'info'|'success'|'warning'|'error'} type
 * @param {string} message
 * @param {number} duration  ms before auto-dismiss (0 = sticky)
 */
export function addToast(type, message, duration = 4000) {
  const id = ++_toastId
  toasts.items.push({ id, type, message, duration })
  if (duration > 0) {
    setTimeout(() => removeToast(id), duration)
  }
}

export function removeToast(id) {
  const idx = toasts.items.findIndex(t => t.id === id)
  if (idx !== -1) toasts.items.splice(idx, 1)
}

// ---------------------------------------------------------------------------
// File tree
// ---------------------------------------------------------------------------
export const fileTree = $state({
  root: null,       // tree node: { name, path, type, children?, expanded? }
  loading: false,
})

// ---------------------------------------------------------------------------
// Site configuration — loaded/saved via sarde IPC API
// ---------------------------------------------------------------------------
import { getConfig as apiGetConfig, updateConfig as apiUpdateConfig, onPreviewReady, onPreviewStopped, onPreviewCrashed, onBuildLog, onBuildComplete, onBuildError } from '../api.js'

export const siteConfig = $state({
  loaded: false,
  saving: false,
  data: /** @type {any} */ (null),
  snapshot: /** @type {string|null} */ (null), // JSON snapshot taken on load/save
})

/** True when the config has unsaved changes. */
export function isConfigDirty() {
  return siteConfig.data && siteConfig.snapshot
    ? JSON.stringify(siteConfig.data) !== siteConfig.snapshot
    : false
}

function takeSnapshot() {
  siteConfig.snapshot = siteConfig.data ? JSON.stringify(siteConfig.data) : null
}

export async function loadSiteConfig() {
  siteConfig.loaded = false
  siteConfig.data = null
  siteConfig.snapshot = null
  try {
    const resp = await apiGetConfig()
    siteConfig.data = resp ?? {}
    siteConfig.loaded = true
    takeSnapshot()
  } catch (e) {
    console.error('Failed to load site config:', e)
    siteConfig.data = {}
    siteConfig.loaded = true
    takeSnapshot()
  }
}

export async function saveSiteConfig() {
  if (siteConfig.saving || !siteConfig.data) return
  siteConfig.saving = true
  try {
    await apiUpdateConfig(siteConfig.data)
    takeSnapshot()
    addToast('success', 'Settings saved')
  } catch (e) {
    addToast('error', `Failed to save settings: ${e}`)
  } finally {
    siteConfig.saving = false
  }
}

// ---------------------------------------------------------------------------
// Validation warnings
// ---------------------------------------------------------------------------
import { validate as apiValidate } from '../api.js'

export const warnings = $state({
  items: [],       // { file, field, message, level }[]
  loading: false,
})

export async function runValidation() {
  if (warnings.loading) return
  warnings.loading = true
  try {
    const resp = await apiValidate()
    warnings.items = resp?.warnings ?? []
    const count = warnings.items.length
    if (count > 0) {
      addToast('warning', `${count} warning${count === 1 ? '' : 's'} found`)
    } else {
      addToast('success', 'No warnings')
    }
  } catch (e) {
    addToast('error', `Validation failed: ${e.message}`)
  } finally {
    warnings.loading = false
  }
}

// ---------------------------------------------------------------------------
// Search state
// ---------------------------------------------------------------------------
export const search = $state({
  query: '',
  results: [],
  loading: false,
})

// ---------------------------------------------------------------------------
// Assets (media panel)
// ---------------------------------------------------------------------------
import { assetList as apiAssetList } from '../api.js'

export const assets = $state({
  items: [],       // AssetInfo[]
  loading: false,
  scope: 'all',    // 'all' | 'bundle' | 'shared'
})

export async function loadAssets(scope = assets.scope) {
  if (assets.loading) return
  assets.loading = true
  assets.scope = scope
  try {
    assets.items = await apiAssetList(scope)
  } catch (e) {
    addToast('error', `Failed to load assets: ${e}`)
    assets.items = []
  } finally {
    assets.loading = false
  }
}

// ---------------------------------------------------------------------------
// Project metadata
// ---------------------------------------------------------------------------
export const project = $state({
  name: '',
  root: '',
  config: null,
  contentPath: '',  // absolute path to the content/ directory
})

// ---------------------------------------------------------------------------
// Tab management helpers
// ---------------------------------------------------------------------------

/** Switch the active tab, preserving/restoring per-tab editor state. */
export function switchToTab(newId) {
  if (!newId || newId === tabs.activeId) return

  // Snapshot outgoing tab's current state
  const outgoing = tabs.items.find(t => t.id === tabs.activeId)
  if (outgoing) {
    outgoing.cachedContent = doc.content
    outgoing.dirty = doc.dirty
    outgoing.savedCursorLine = doc.cursorLine
    outgoing.savedCursorCol = doc.cursorCol
  }

  tabs.activeId = newId

  const incoming = tabs.items.find(t => t.id === newId)
  if (!incoming) return

  doc.filePath = incoming.path
  doc.dirty = incoming.dirty ?? false
  doc.content = incoming.cachedContent ?? ''
  doc.cursorLine = incoming.savedCursorLine ?? 1
  doc.cursorCol = incoming.savedCursorCol ?? 1
  const words = doc.content.split(/\s+/).filter(w => w).length
  doc.wordCount = words
  doc.readingTime = Math.max(1, Math.ceil(words / 250))
}

/** Close a tab by id, switching to an adjacent tab if it was active. */
export function closeTabById(id) {
  const idx = tabs.items.findIndex(t => t.id === id)
  if (idx === -1) return

  const wasActive = tabs.activeId === id
  tabs.items.splice(idx, 1)

  if (wasActive) {
    const next = tabs.items[Math.min(idx, tabs.items.length - 1)]
    if (next) {
      // Load next tab without snapshot (outgoing tab is gone)
      tabs.activeId = next.id
      doc.filePath = next.path
      doc.dirty = next.dirty ?? false
      doc.content = next.cachedContent ?? ''
      doc.cursorLine = next.savedCursorLine ?? 1
      doc.cursorCol = next.savedCursorCol ?? 1
      const words = doc.content.split(/\s+/).filter(w => w).length
      doc.wordCount = words
      doc.readingTime = Math.max(1, Math.ceil(words / 250))
    } else {
      tabs.activeId = null
      doc.filePath = ''
      doc.content = ''
      doc.dirty = false
      doc.wordCount = 0
      doc.readingTime = 0
    }
  }
}
