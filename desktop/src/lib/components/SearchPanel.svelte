<script>
  import { readDir, readTextFile, writeTextFile } from '@tauri-apps/plugin-fs'
  import { project, doc, tabs, switchToTab, addToast } from '../stores/app.svelte.js'
  import { readContent, saveContent } from '../api.js'
  import { Search, FileText, Loader, Replace, ChevronDown, ChevronRight, CaseSensitive, Regex } from 'lucide-svelte'

  /** Flat index of all .md files under contentPath */
  let fileIndex = $state(/** @type {Array<{path:string,name:string,relPath:string,lines:string[]}>} */ ([]))
  let indexedRoot = $state('')
  let indexing = $state(false)

  let query = $state('')
  let results = $state(/** @type {Array<{path:string,name:string,relPath:string,matches:Array<{lineNum:number,lineText:string,col:number}>}>} */ ([]))
  let searching = $state(false)
  let searchTimer = null

  let replaceQuery = $state('')
  let showReplace = $state(false)
  let useRegex = $state(false)
  let caseSensitive = $state(false)
  let replacing = $state(false)
  let collapsedFiles = $state(new Set())

  // Rebuild index when content path changes
  $effect(() => {
    const root = project.contentPath
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
    const root = project.contentPath
    if (!root) return
    if (indexedRoot === root && fileIndex.length > 0) return
    if (indexPromise) return indexPromise  // deduplicate concurrent calls
    indexing = true
    indexPromise = collectFiles(root, root).then(files => {
      if (project.contentPath === root) {  // guard against root change mid-flight
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
            const eol = content.includes('\r\n') ? '\r\n' : '\n'
            acc.push({
              path: fullPath,
              name: item.name,
              relPath: fullPath.replace(root, '').replace(/^[/\\]/, ''),
              lines: content.split(eol),
              eol,
            })
          } catch {}
        }
      }
    } catch {}
    return acc
  }

  function buildMatcher(q) {
    if (useRegex) {
      try {
        return new RegExp(q, caseSensitive ? 'g' : 'gi')
      } catch {
        return null
      }
    }
    return null
  }

  async function runSearch() {
    const q = query.trim()
    if (!q) { results = []; return }

    searching = true
    try {
      await ensureIndex()

      const regex = buildMatcher(q)
      const found = []

      for (const file of fileIndex) {
        const matches = []
        for (let i = 0; i < file.lines.length; i++) {
          const line = file.lines[i]
          if (regex) {
            regex.lastIndex = 0
            const m = regex.exec(line)
            if (m) {
              matches.push({ lineNum: i + 1, lineText: line, col: m.index })
            }
          } else {
            const haystack = caseSensitive ? line : line.toLowerCase()
            const needle = caseSensitive ? q : q.toLowerCase()
            const col = haystack.indexOf(needle)
            if (col !== -1) {
              matches.push({ lineNum: i + 1, lineText: line, col })
            }
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
    const contentPath = relContentPath(filePath)
    const existing = tabs.items.find(t => t.path === filePath || t.contentPath === contentPath)
    if (existing) {
      switchToTab(existing.id)
    } else {
      try {
        const file = await readContent(contentPath)
        const content = file.raw ?? await readTextFile(filePath)
        const absPath = (file.absPath ?? filePath).replace(/\\/g, '/')
        const id = crypto.randomUUID()
        tabs.items.push({
          id,
          name: fileName,
          path: absPath,
          contentPath: file.path ?? contentPath,
          dirty: false,
          cachedContent: content,
        })
        switchToTab(id)
      } catch {
        addToast('error', 'Could not open file')
        return
      }
    }
    doc.targetLine = lineNum
  }

  function toggleFileCollapse(path) {
    const next = new Set(collapsedFiles)
    if (next.has(path)) next.delete(path)
    else next.add(path)
    collapsedFiles = next
  }

  async function replaceInFile(filePath, matchIndex) {
    const q = query.trim()
    const r = replaceQuery
    if (!q) return

    replacing = true
    try {
      const file = fileIndex.find(f => f.path === filePath)
      if (!file) return

      const result = results.find(res => res.path === filePath)
      if (!result || !result.matches[matchIndex]) return
      const match = result.matches[matchIndex]

      const lineIdx = match.lineNum - 1
      const line = file.lines[lineIdx]
      let newLine
      if (useRegex) {
        try {
          const re = new RegExp(q, caseSensitive ? '' : 'i')
          newLine = line.replace(re, r)
        } catch { return }
      } else {
        const col = match.col
        const matchLen = q.length
        newLine = line.slice(0, col) + r + line.slice(col + matchLen)
      }

      file.lines[lineIdx] = newLine
      const newContent = file.lines.join(file.eol || '\n')
      await writeTextFile(filePath, newContent)

      const openTab = tabs.items.find(t => t.path === filePath)
      if (openTab) {
        openTab.cachedContent = newContent
        if (tabs.activeId === openTab.id) {
          doc.content = newContent
          doc.externalUpdate = doc.externalUpdate + 1
        }
      }

      runSearch()
      addToast('success', `Replaced in ${file.name}:${match.lineNum}`)
    } catch {
      addToast('error', 'Replace failed')
    } finally {
      replacing = false
    }
  }

  async function replaceAllInFile(filePath) {
    const q = query.trim()
    const r = replaceQuery
    if (!q) return

    replacing = true
    try {
      const file = fileIndex.find(f => f.path === filePath)
      if (!file) return

      let count = 0
      for (let i = 0; i < file.lines.length; i++) {
        if (useRegex) {
          try {
            const re = new RegExp(q, caseSensitive ? 'g' : 'gi')
            const before = file.lines[i]
            file.lines[i] = before.replace(re, r)
            if (file.lines[i] !== before) count++
          } catch { return }
        } else {
          let cursor = 0
          while (true) {
            const hay = caseSensitive ? file.lines[i] : file.lines[i].toLowerCase()
            const needle = caseSensitive ? q : q.toLowerCase()
            const col = hay.indexOf(needle, cursor)
            if (col === -1) break
            file.lines[i] = file.lines[i].slice(0, col) + r + file.lines[i].slice(col + q.length)
            cursor = col + r.length
            count++
          }
        }
      }

      const newContent = file.lines.join(file.eol || '\n')
      await writeTextFile(filePath, newContent)

      const openTab = tabs.items.find(t => t.path === filePath)
      if (openTab) {
        openTab.cachedContent = newContent
        if (tabs.activeId === openTab.id) {
          doc.content = newContent
          doc.externalUpdate = doc.externalUpdate + 1
        }
      }

      runSearch()
      addToast('success', `Replaced ${count} occurrence${count !== 1 ? 's' : ''} in ${file.name}`)
    } catch {
      addToast('error', 'Replace failed')
    } finally {
      replacing = false
    }
  }

  async function replaceAll() {
    const q = query.trim()
    if (!q || results.length === 0) return

    const matchCount = results.reduce((sum, r) => sum + r.matches.length, 0)
    const fileCount = results.length
    if (!confirm(`Replace ${matchCount} occurrence${matchCount !== 1 ? 's' : ''} in ${fileCount} file${fileCount !== 1 ? 's' : ''}?`)) return

    replacing = true
    try {
      let totalCount = 0
      let fileCount = 0

      for (const result of [...results]) {
        const file = fileIndex.find(f => f.path === result.path)
        if (!file) continue

        let changed = false
        for (let i = 0; i < file.lines.length; i++) {
          if (useRegex) {
            try {
              const re = new RegExp(q, caseSensitive ? 'g' : 'gi')
              const before = file.lines[i]
              file.lines[i] = before.replace(re, replaceQuery)
              if (file.lines[i] !== before) { changed = true; totalCount++ }
            } catch { return }
          } else {
            let cursor = 0
            while (true) {
              const hay = caseSensitive ? file.lines[i] : file.lines[i].toLowerCase()
              const needle = caseSensitive ? q : q.toLowerCase()
              const col = hay.indexOf(needle, cursor)
              if (col === -1) break
              file.lines[i] = file.lines[i].slice(0, col) + replaceQuery + file.lines[i].slice(col + q.length)
              cursor = col + replaceQuery.length
              changed = true
              totalCount++
            }
          }
        }

        if (changed) {
          fileCount++
          const newContent = file.lines.join(file.eol || '\n')
          await writeTextFile(result.path, newContent)

          const openTab = tabs.items.find(t => t.path === result.path)
          if (openTab) {
            openTab.cachedContent = newContent
            if (tabs.activeId === openTab.id) {
              doc.content = newContent
              doc.externalUpdate = doc.externalUpdate + 1
            }
          }
        }
      }

      runSearch()
      addToast('success', `Replaced ${totalCount} occurrence${totalCount !== 1 ? 's' : ''} in ${fileCount} file${fileCount !== 1 ? 's' : ''}`)
    } catch {
      addToast('error', 'Replace all failed')
    } finally {
      replacing = false
    }
  }

  function relContentPath(absPath) {
    const root = project.contentPath.replace(/\\/g, '/').replace(/\/+$/, '')
    const full = absPath.replace(/\\/g, '/')
    return full.startsWith(root + '/') ? full.slice(root.length + 1) : full
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
  <div class="search-header">
    <div class="search-input-row">
      <button class="toggle-replace-btn" onclick={() => showReplace = !showReplace} title="Toggle Replace">
        {#if showReplace}
          <ChevronDown size={12} />
        {:else}
          <ChevronRight size={12} />
        {/if}
      </button>
      <Search size={14} class="search-icon" />
      <input
        class="search-input"
        type="text"
        placeholder="Search in files…"
        bind:value={query}
      />
      <button
        class="search-option-btn"
        class:active={caseSensitive}
        onclick={() => { caseSensitive = !caseSensitive; if (query.trim()) runSearch() }}
        title="Match Case"
      >
        <CaseSensitive size={14} />
      </button>
      <button
        class="search-option-btn"
        class:active={useRegex}
        onclick={() => { useRegex = !useRegex; if (query.trim()) runSearch() }}
        title="Use Regular Expression"
      >
        <Regex size={14} />
      </button>
      {#if isLoading}
        <span class="search-spinner"><Loader size={13} /></span>
      {/if}
    </div>

    {#if showReplace}
      <div class="replace-input-row">
        <Replace size={14} class="replace-icon" />
        <input
          class="search-input"
          type="text"
          placeholder="Replace…"
          bind:value={replaceQuery}
        />
        <button
          class="replace-all-btn"
          onclick={replaceAll}
          disabled={replacing || !query.trim() || results.length === 0}
          title="Replace All"
        >
          Replace All
        </button>
      </div>
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
    {#each results as file, fileIdx}
      {@const collapsed = collapsedFiles.has(file.path)}
      <div class="result-file">
        <div class="result-file-header">
          <button class="collapse-btn" onclick={() => toggleFileCollapse(file.path)}>
            {#if collapsed}
              <ChevronRight size={11} />
            {:else}
              <ChevronDown size={11} />
            {/if}
          </button>
          <FileText size={13} />
          <span class="result-file-name">{file.name}</span>
          <span class="result-file-path">{file.relPath}</span>
          {#if showReplace}
            <button
              class="file-replace-btn"
              onclick={() => replaceAllInFile(file.path)}
              disabled={replacing}
              title="Replace all in this file"
            >
              <Replace size={11} />
            </button>
          {/if}
          <span class="result-match-count">{file.matches.length}</span>
        </div>

        {#if !collapsed}
          {#each file.matches as m, matchIdx}
            {@const seg = splitMatch(m.lineText.trimStart(), Math.max(0, m.col - (m.lineText.length - m.lineText.trimStart().length)), query.trim().length)}
            <div class="result-line-row">
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
              {#if showReplace}
                <button
                  class="match-replace-btn"
                  onclick={() => replaceInFile(file.path, matchIdx)}
                  disabled={replacing}
                  title="Replace this match"
                >
                  <Replace size={11} />
                </button>
              {/if}
            </div>
          {/each}
        {/if}
      </div>
    {/each}

    {#if !project.contentPath}
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

  .search-header {
    border-bottom: 1px solid var(--cr-border);
  }

  .search-input-row {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 6px 8px;
  }

  .search-input-row :global(.search-icon) {
    flex-shrink: 0;
    color: var(--cr-text-muted);
  }

  .search-input {
    flex: 1;
    border: none;
    background: transparent;
    font-size: 13px;
    color: var(--cr-text);
    outline: none;
    font-family: inherit;
    min-width: 0;
  }

  .search-input::placeholder {
    color: var(--cr-text-muted);
  }

  .toggle-replace-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    border: none;
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
    border-radius: var(--cr-radius-sm);
    flex-shrink: 0;
  }

  .toggle-replace-btn:hover {
    color: var(--cr-text);
    background: var(--cr-hover);
  }

  .search-option-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border: 1px solid transparent;
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
    border-radius: var(--cr-radius-sm);
    flex-shrink: 0;
  }

  .search-option-btn:hover {
    color: var(--cr-text);
    background: var(--cr-hover);
  }

  .search-option-btn.active {
    color: var(--cr-accent);
    border-color: var(--cr-accent);
    background: var(--cr-active);
  }

  .replace-input-row {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 8px 6px 30px;
  }

  .replace-input-row :global(.replace-icon) {
    flex-shrink: 0;
    color: var(--cr-text-muted);
  }

  .replace-all-btn {
    flex-shrink: 0;
    padding: 2px 8px;
    font-size: 11px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
    background: var(--cr-bg-elevated);
    color: var(--cr-text);
    cursor: pointer;
    white-space: nowrap;
  }

  .replace-all-btn:hover:not(:disabled) {
    border-color: var(--cr-accent);
    color: var(--cr-accent);
  }

  .replace-all-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .search-spinner {
    flex-shrink: 0;
    color: var(--cr-text-muted);
    display: flex;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .search-summary {
    padding: 4px 10px 6px;
    font-size: 11px;
    color: var(--cr-text-muted);
    border-bottom: 1px solid var(--cr-border);
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
    color: var(--cr-text-muted);
    font-size: 11px;
    position: sticky;
    top: 0;
    background: var(--cr-bg-base);
    z-index: 1;
  }

  .result-file-header :global(svg) {
    flex-shrink: 0;
    color: var(--cr-accent);
  }

  .collapse-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    border: none;
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
    border-radius: var(--cr-radius-sm);
    flex-shrink: 0;
  }

  .collapse-btn:hover {
    color: var(--cr-text);
  }

  .result-file-name {
    font-weight: 600;
    color: var(--cr-text);
    white-space: nowrap;
  }

  .result-file-path {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    opacity: 0.6;
  }

  .file-replace-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    border: none;
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
    border-radius: var(--cr-radius-sm);
    flex-shrink: 0;
  }

  .file-replace-btn:hover:not(:disabled) {
    color: var(--cr-accent);
    background: var(--cr-hover);
  }

  .file-replace-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .result-match-count {
    background: var(--cr-active);
    color: var(--cr-accent);
    border-radius: 10px;
    padding: 0 6px;
    font-size: 10px;
    font-weight: 600;
    flex-shrink: 0;
  }

  .result-line-row {
    display: flex;
    align-items: center;
  }

  .result-line {
    display: flex;
    align-items: baseline;
    gap: 8px;
    flex: 1;
    min-width: 0;
    padding: 2px 10px 2px 14px;
    border: none;
    background: transparent;
    cursor: pointer;
    text-align: left;
    font-family: var(--cr-font-mono);
    font-size: 12px;
    color: var(--cr-text);
    white-space: nowrap;
  }

  .result-line:hover {
    background: var(--cr-hover);
  }

  .match-replace-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    border: none;
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
    border-radius: var(--cr-radius-sm);
    flex-shrink: 0;
    margin-right: 4px;
  }

  .match-replace-btn:hover:not(:disabled) {
    color: var(--cr-accent);
    background: var(--cr-hover);
  }

  .match-replace-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .result-linenum {
    flex-shrink: 0;
    font-size: 11px;
    color: var(--cr-text-muted);
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
    color: var(--cr-text-muted);
  }

  .result-highlight {
    color: var(--cr-text);
    background: rgba(137, 180, 250, 0.25);
    border-radius: 2px;
    padding: 0 1px;
  }

  .search-hint {
    padding: 12px;
    margin: 0;
    font-size: 12px;
    color: var(--cr-text-muted);
  }
</style>
