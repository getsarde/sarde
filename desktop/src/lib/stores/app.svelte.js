// Coderoo Desktop — shared reactive state (Svelte 5 runes)
// This file uses the .svelte.js extension to enable rune syntax at module level.

// ---------------------------------------------------------------------------
// Sidecar connection (Go backend served over HTTP)
// ---------------------------------------------------------------------------
export const sidecar = $state({
  url: '',
  ready: false,
  previewUrl: '',  // dev preview server URL (separate from IPC API)
})

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
  settingsOpen: false,
  settingsSection: 'general',
  commandPaletteOpen: false,
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
// File tree — populated by sidecar /api/files responses
// ---------------------------------------------------------------------------
export const fileTree = $state({
  root: null,       // tree node: { name, path, type, children?, expanded? }
  loading: false,
})

// ---------------------------------------------------------------------------
// Site configuration — loaded/saved via sarde IPC API
// ---------------------------------------------------------------------------
import { getConfig as apiGetConfig, updateConfig as apiUpdateConfig } from '../api.js'

export const siteConfig = $state({
  loaded: false,
  saving: false,
  data: /** @type {any} */ (null),
})

export async function loadSiteConfig() {
  if (!sidecar.ready) return
  siteConfig.loaded = false
  siteConfig.data = null
  try {
    const resp = await apiGetConfig()
    siteConfig.data = resp?.data ?? {}
    siteConfig.loaded = true
  } catch (e) {
    console.error('Failed to load site config:', e)
    siteConfig.data = {}
    siteConfig.loaded = true
  }
}

export async function saveSiteConfig() {
  if (siteConfig.saving || !siteConfig.data) return
  siteConfig.saving = true
  try {
    await apiUpdateConfig(siteConfig.data)
    addToast('success', 'Settings saved')
  } catch (e) {
    addToast('error', `Failed to save settings: ${e}`)
  } finally {
    siteConfig.saving = false
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
// Project metadata
// ---------------------------------------------------------------------------
export const project = $state({
  name: '',
  root: '',
  config: null,
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
