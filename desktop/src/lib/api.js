// Coderoo Desktop — API client for the Go sidecar (sarde IPC server).
// All endpoints defined in internal/server/api.go.

import { sidecar } from './stores/app.svelte.js'

function baseUrl() {
  return sidecar.url
}

async function request(method, path, body = null) {
  const url = baseUrl() + path
  const opts = {
    method,
    headers: { 'Content-Type': 'application/json' },
  }
  if (body !== null) {
    opts.body = JSON.stringify(body)
  }
  const resp = await fetch(url, opts)
  const data = await resp.json()
  if (!resp.ok) {
    const msg = data?.error?.message || data?.error || resp.statusText
    throw new Error(msg)
  }
  return data
}

// ---------------------------------------------------------------------------
// Project lifecycle
// ---------------------------------------------------------------------------

export async function projectOpen(dir) {
  return request('POST', '/api/project/open', { dir })
}

export async function projectCreate(dir, title = '') {
  return request('POST', '/api/project/create', { dir, title })
}

export async function projectClose() {
  return request('POST', '/api/project/close')
}

export async function projectInfo() {
  return request('GET', '/api/project/info')
}

// ---------------------------------------------------------------------------
// Content CRUD
// ---------------------------------------------------------------------------

export async function listContent(collection = '') {
  const q = collection ? `?collection=${encodeURIComponent(collection)}` : ''
  return request('GET', `/api/content${q}`)
}

export async function readContent(path) {
  return request('GET', `/api/content/${encodeURIComponent(path)}`)
}

export async function saveContent(path, frontmatter, body) {
  return request('PUT', `/api/content/${encodeURIComponent(path)}`, { frontmatter, body })
}

export async function createContent(collection, title) {
  return request('POST', '/api/content', { collection, title })
}

export async function deleteContent(path) {
  return request('DELETE', `/api/content/${encodeURIComponent(path)}`)
}

export async function renameContent(oldPath, newPath) {
  return request('POST', '/api/content/rename', { old_path: oldPath, new_path: newPath })
}

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

export async function getConfig() {
  return request('GET', '/api/config')
}

export async function updateConfig(settings) {
  return request('PATCH', '/api/config', settings)
}

export async function getCollections() {
  return request('GET', '/api/collections')
}

// ---------------------------------------------------------------------------
// Build & preview
// ---------------------------------------------------------------------------

export async function build() {
  return request('POST', '/api/build')
}

export async function validate() {
  return request('POST', '/api/build/validate')
}

export async function startPreview(port = 3000) {
  return request('POST', '/api/preview/start', { port })
}

export async function stopPreview() {
  return request('POST', '/api/preview/stop')
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

export async function renderMarkdown(markdown) {
  return request('POST', '/api/render/markdown', { markdown })
}

// ---------------------------------------------------------------------------
// WebSocket events
// ---------------------------------------------------------------------------

export function connectEvents(onMessage) {
  const wsUrl = baseUrl().replace(/^http/, 'ws') + '/api/events'
  const ws = new WebSocket(wsUrl)
  ws.onmessage = (e) => {
    try {
      const msg = JSON.parse(e.data)
      onMessage(msg)
    } catch (_) {}
  }
  ws.onclose = () => {
    // Reconnect after 2s.
    setTimeout(() => connectEvents(onMessage), 2000)
  }
  return ws
}
