<script>
  import { readDir, readTextFile } from '@tauri-apps/plugin-fs'
  import { sidecar, doc, tabs, switchToTab, addToast } from '../stores/app.svelte.js'
  import { Search, FileText, Loader } from 'lucide-svelte'

  /** Flat index of all .md files under contentPath */
  let fileIndex = $state(/** @type {Array<{path:string,name:string,relPath:string,lines:string[]}>} */ ([]))
  let indexedRoot = $state('')
  let indexing = $state(false)

  let query = $state('')
  let results = $state(/** @type {Array<{path:string,name:string,relPath:string,matches:Array<{lineNum:number,lineText:string,col:number}>}>} */ ([]))
  let searching = $state(false)
  let searchTimer = null

  // Rebuild index when content path changes
  $effect(() => {
    const root = sidecar.contentPath
    if (root && root !== indexedRoot) {
      fileIndex = []
      indexedRoot = root
      results = []
    }
  })

  // Debounced search on query change
  $effect(() => {
    const q = query  // track dependency
    clearTimeout(searchTimer)
    if (!q.trim()) { results = []; return }
    searchTimer = setTimeout(runSearch, 250)
    return () => clearTimeout(searchTimer)
  })

  let indexPromise = null

  async function ensureIndex() {
    const root = sidecar.contentPath
    if (!root) return
    if (indexedRoot === root && fileIndex.length > 0) return
    if (indexPromise) return indexPromise  // deduplicate concurrent calls
    indexing = true
    indexPromise = collectFiles(root, root).then(files => {
      if (sidecar.contentPath === root) {  // guard against root change mid-flight
        fileIndex = files
        indexedRoot = root
      }
    }).finally(() => {
      indexing = false
      indexPromise = null
    })
    return indexPromise
  }

  async function collectFiles(dir, root) {
    dir = dir.replace(/\\/g, '/')
    root = root.replace(/\\/g, '/')
    const acc = []
    try {
      const items = await readDir(dir)
      for (const item of items) {
        if (item.name.startsWith('.')) continue   // skip .trash, .git, etc.
        const fullPath = dir + '/' + item.name
        if (item.isDirectory) {
          const children = await collectFiles(fullPath, root)
          acc.push(...children)
        } else if (item.name.endsWith('.md')) {
          try {
            const content = await readTextFile(fullPath)
            acc.push({
              path: fullPath,
              name: item.name,
              relPath: fullPath.replace(root, '').replace(/^[/\\]/, ''),
              lines: content.split('\n'),
            })
          } catch {}
        }
      }
    } catch {}
    return acc
  }

  async function runSearch() {
    const q = query.trim()
    if (!q) { results = []; return }

    searching = true
    try {
      await ensureIndex()

      const lowerQ = q.toLowerCase()
      const found = []

      for (const file of fileIndex) {
        const matches = []
        for (let i = 0; i < file.lines.length; i++) {
          const col = file.lines[i].toLowerCase().indexOf(lowerQ)
          if (col !== -1) {
            matches.push({ lineNum: i + 1, lineText: file.lines[i], col })
          }
        }
        if (matches.length > 0) {
          found.push({ path: file.path, name: file.name, relPath: file.relPath, matches })
        }
      }

      results = found
    } finally {
      searching = false
    }
  }

  async function openResult(filePath, fileName, lineNum) {
    const existing = tabs.items.find(t => t.path === filePath)
    if (existing) {
      switchToTab(existing.id)
    } else {
      try {
        const content = await readTextFile(filePath)
        const id = crypto.randomUUID()
        tabs.items = [...tabs.items, { id, name: fileName, path: filePath, dirty: false, cachedContent: content }]
        tabs.activeId = id
        doc.content = content
        doc.filePath = filePath
        doc.dirty = false
        doc.wordCount = content.split(/\s+/).filter(w => w).length
        doc.readingTime = Math.max(1, Math.ceil(doc.wordCount / 250))
      } catch {
        addToast('error', 'Could not open file')
        return
      }
    }
    doc.targetLine = lineNum
  }

  /** Split a line into before/match/after segments for highlighting, trimming long lines */
  function splitMatch(lineText, col, qLen) {
    const max = 120
    let start = Math.max(0, col - 40)
    const before = (start > 0 ? '…' : '') + lineText.slice(start, col)
    const match  = lineText.slice(col, col + qLen)
    const rawAfter = lineText.slice(col + qLen)
    const after = rawAfter.slice(0, max - before.length - match.length) + (rawAfter.length > max ? '…' : '')
    return { before, match, after }
  }

  let totalMatches = $derived(results.reduce((s, r) => s + r.matches.length, 0))
  let isLoading = $derived(indexing || searching)
</script>

<div class="search-panel">
  <div class="search-input-row">
    <Search size={14} class="search-icon" />
    <input
      class="search-input"
      type="text"
      placeholder="Search in files…"
      bind:value={query}
    />
    {#if isLoading}
      <span class="search-spinner"><Loader size={13} /></span>
    {/if}
  </div>

  {#if query.trim() && !isLoading}
    <div class="search-summary">
      {#if results.length === 0}
        No results for "{query}"
      {:else}
        {totalMatches} result{totalMatches !== 1 ? 's' : ''} in {results.length} file{results.length !== 1 ? 's' : ''}
      {/if}
    </div>
  {/if}

  <div class="results-list">
    {#each results as file}
      <div class="result-file">
        <div class="result-file-header">
          <FileText size={13} />
          <span class="result-file-name">{file.name}</span>
          <span class="result-file-path">{file.relPath}</span>
          <span class="result-match-count">{file.matches.length}</span>
        </div>

        {#each file.matches as m}
          {@const seg = splitMatch(m.lineText.trimStart(), Math.max(0, m.col - (m.lineText.length - m.lineText.trimStart().length)), query.trim().length)}
          <button
            class="result-line"
            onclick={() => openResult(file.path, file.name, m.lineNum)}
            title="{file.relPath}:{m.lineNum}"
          >
            <span class="result-linenum">{m.lineNum}</span>
            <span class="result-text">
              <span class="result-before">{seg.before}</span><span class="result-highlight">{seg.match}</span><span class="result-after">{seg.after}</span>
            </span>
          </button>
        {/each}
      </div>
    {/each}

    {#if !sidecar.contentPath}
      <p class="search-hint">Open a project folder first.</p>
    {:else if !query.trim()}
      <p class="search-hint">Type to search across all .md files.</p>
    {/if}
  </div>
</div>

<style>
  .search-panel {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
  }

  .search-input-row {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px;
    border-bottom: 1px solid var(--color-border, #313244);
  }

  .search-input-row :global(.search-icon) {
    flex-shrink: 0;
    color: var(--color-text-muted, #6c7086);
  }

  .search-input {
    flex: 1;
    border: none;
    background: transparent;
    font-size: 13px;
    color: var(--color-text, #cdd6f4);
    outline: none;
    font-family: inherit;
  }

  .search-input::placeholder {
    color: var(--color-text-muted, #6c7086);
  }

  .search-spinner {
    flex-shrink: 0;
    color: var(--color-text-muted, #6c7086);
    display: flex;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .search-summary {
    padding: 4px 10px 6px;
    font-size: 11px;
    color: var(--color-text-muted, #6c7086);
    border-bottom: 1px solid var(--color-border, #313244);
  }

  .results-list {
    flex: 1;
    overflow-y: auto;
    padding: 4px 0;
  }

  .result-file {
    margin-bottom: 4px;
  }

  .result-file-header {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 5px 10px 3px;
    color: var(--color-text-muted, #6c7086);
    font-size: 11px;
    position: sticky;
    top: 0;
    background: var(--color-surface, #1e1e2e);
    z-index: 1;
  }

  .result-file-header :global(svg) {
    flex-shrink: 0;
    color: var(--color-accent, #89b4fa);
  }

  .result-file-name {
    font-weight: 600;
    color: var(--color-text, #cdd6f4);
    white-space: nowrap;
  }

  .result-file-path {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    opacity: 0.6;
  }

  .result-match-count {
    background: var(--color-active, rgba(137, 180, 250, 0.15));
    color: var(--color-accent, #89b4fa);
    border-radius: 10px;
    padding: 0 6px;
    font-size: 10px;
    font-weight: 600;
    flex-shrink: 0;
  }

  .result-line {
    display: flex;
    align-items: baseline;
    gap: 8px;
    width: 100%;
    padding: 2px 10px 2px 14px;
    border: none;
    background: transparent;
    cursor: pointer;
    text-align: left;
    font-family: var(--font-mono, monospace);
    font-size: 12px;
    color: var(--color-text, #cdd6f4);
    white-space: nowrap;
  }

  .result-line:hover {
    background: var(--color-hover, rgba(255, 255, 255, 0.06));
  }

  .result-linenum {
    flex-shrink: 0;
    font-size: 11px;
    color: var(--color-text-muted, #6c7086);
    min-width: 28px;
    text-align: right;
    user-select: none;
  }

  .result-text {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .result-before,
  .result-after {
    color: var(--color-text-muted, #6c7086);
  }

  .result-highlight {
    color: var(--color-text, #cdd6f4);
    background: rgba(137, 180, 250, 0.25);
    border-radius: 2px;
    padding: 0 1px;
  }

  .search-hint {
    padding: 12px;
    margin: 0;
    font-size: 12px;
    color: var(--color-text-muted, #6c7086);
  }
</style>
