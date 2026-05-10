<script>
  import { ui, doc, warnings, runValidation, loadAssets } from '../stores/app.svelte.js'
  import { List, PencilLine, Image, TrendingUp, AlertTriangle, CheckCircle, RefreshCw, Code } from 'lucide-svelte'
  import PropertiesPanel from './PropertiesPanel.svelte'
  import MediaPanel from './MediaPanel.svelte'

  function togglePanel(panel) {
    ui.rightPanel = ui.rightPanel === panel ? null : panel
  }

  /** Parse headings from markdown content for table of contents */
  let headings = $derived.by(() => {
    if (!doc.content) return []
    const lines = doc.content.split('\n')
    const result = []
    let inFrontmatter = false
    for (const line of lines) {
      if (line.trim() === '---') {
        inFrontmatter = !inFrontmatter
        continue
      }
      if (inFrontmatter) continue
      const match = line.match(/^(#{1,6})\s+(.+)/)
      if (match) {
        result.push({ level: match[1].length, text: match[2].replace(/[#*`[\]]/g, '').trim() })
      }
    }
    return result
  })

  /** Document statistics */
  let stats = $derived({
    words: doc.wordCount,
    readingTime: doc.readingTime,
    characters: doc.content.length,
    lines: doc.content ? doc.content.split('\n').length : 0,
    headings: headings.length,
  })

  /** Group warnings by file */
  let warningsByFile = $derived.by(() => {
    const groups = new Map()
    for (const w of warnings.items) {
      const file = w.file || w.File || 'Unknown'
      if (!groups.has(file)) groups.set(file, [])
      groups.get(file).push(w)
    }
    return groups
  })
</script>

<aside class="right-sidebar">
  {#if ui.rightPanel}
    <div class="sidebar-panel">
      {#if ui.rightPanel === 'toc'}
        <div class="panel-header"><span class="panel-title">Table of Contents</span></div>
        <div class="panel-body">
          {#if headings.length === 0}
            <p class="empty-msg">No headings found.</p>
          {:else}
            <ul class="toc-list">
              {#each headings as h}
                <li class="toc-item" style="padding-left: {(h.level - 1) * 12 + 8}px">
                  <span class="toc-level">H{h.level}</span>
                  <span class="toc-text">{h.text}</span>
                </li>
              {/each}
            </ul>
          {/if}
        </div>

      {:else if ui.rightPanel === 'properties'}
        <div class="panel-header">
          <span class="panel-title">Properties</span>
          <button
            class="panel-action"
            class:active={ui.propertiesMode === 'yaml'}
            onclick={() => { ui.propertiesMode = ui.propertiesMode === 'form' ? 'yaml' : 'form' }}
            title={ui.propertiesMode === 'form' ? 'Switch to raw YAML' : 'Switch to form view'}
          >
            <Code size={13} />
          </button>
        </div>
        <div class="panel-body">
          <PropertiesPanel />
        </div>

      {:else if ui.rightPanel === 'assets'}
        <div class="panel-header">
          <span class="panel-title">Assets</span>
          <button class="panel-action" onclick={() => loadAssets()} title="Refresh">
            <RefreshCw size={13} />
          </button>
        </div>
        <div class="panel-body">
          <MediaPanel />
        </div>

      {:else if ui.rightPanel === 'stats'}
        <div class="panel-header"><span class="panel-title">Document Stats</span></div>
        <div class="panel-body">
          <div class="stats-grid">
            <div class="stat-card">
              <span class="stat-value">{stats.words}</span>
              <span class="stat-label">Words</span>
            </div>
            <div class="stat-card">
              <span class="stat-value">{stats.readingTime}m</span>
              <span class="stat-label">Read time</span>
            </div>
            <div class="stat-card">
              <span class="stat-value">{stats.characters}</span>
              <span class="stat-label">Characters</span>
            </div>
            <div class="stat-card">
              <span class="stat-value">{stats.lines}</span>
              <span class="stat-label">Lines</span>
            </div>
            <div class="stat-card">
              <span class="stat-value">{stats.headings}</span>
              <span class="stat-label">Headings</span>
            </div>
          </div>
        </div>
      {:else if ui.rightPanel === 'warnings'}
        <div class="panel-header">
          <span class="panel-title">Warnings</span>
          <button class="panel-action" onclick={runValidation} disabled={warnings.loading} title="Re-validate">
            <RefreshCw size={13} />
          </button>
        </div>
        <div class="panel-body">
          {#if warnings.loading}
            <p class="empty-msg">Validating...</p>
          {:else if warnings.items.length === 0}
            <div class="warnings-empty">
              <CheckCircle size={20} />
              <p>No warnings</p>
            </div>
          {:else}
            <div class="warnings-list">
              {#each [...warningsByFile] as [file, items]}
                <div class="warning-group">
                  <div class="warning-file">{file}</div>
                  {#each items as w}
                    <div class="warning-item" class:warning-error={(w.level || w.Level) === 'error'}>
                      <span class="warning-field">{w.field || w.Field || 'lint'}</span>
                      <span class="warning-msg">{w.message || w.Message}</span>
                    </div>
                  {/each}
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/if}
    </div>
  {/if}

  <div class="icon-strip">
    <button class="icon-btn" class:active={ui.rightPanel === 'toc'}        onclick={() => togglePanel('toc')}        title="Table of Contents"><List       size={20} /></button>
    <button class="icon-btn" class:active={ui.rightPanel === 'properties'} onclick={() => togglePanel('properties')} title="Properties">          <PencilLine size={20} /></button>
    <button class="icon-btn" class:active={ui.rightPanel === 'assets'}     onclick={() => togglePanel('assets')}     title="Assets">              <Image      size={20} /></button>
    <button class="icon-btn" class:active={ui.rightPanel === 'stats'}      onclick={() => togglePanel('stats')}      title="Stats">               <TrendingUp size={20} /></button>
    <button class="icon-btn" class:active={ui.rightPanel === 'warnings'}   onclick={() => togglePanel('warnings')}   title="Warnings">
      <span class="icon-btn-badge-wrap">
        <AlertTriangle size={20} />
        {#if warnings.items.length > 0}
          <span class="badge">{warnings.items.length}</span>
        {/if}
      </span>
    </button>
  </div>
</aside>

<style>
  .right-sidebar {
    display: flex;
    height: 100%;
    background: var(--cr-bg-base);
    border-left: 1px solid var(--cr-border);
  }

  .icon-strip {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 44px;
    padding: 6px 0;
    gap: 2px;
    background: var(--cr-bg-input);
    border-left: 1px solid var(--cr-border);
  }

  .icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border: none;
    border-radius: var(--cr-radius);
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
  }

  .icon-btn:hover {
    color: var(--cr-text);
    background: var(--cr-hover);
  }

  .icon-btn.active {
    color: var(--cr-accent);
    background: var(--cr-active);
  }

  .sidebar-panel {
    width: 240px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--cr-text-muted);
    border-bottom: 1px solid var(--cr-border);
  }

  .panel-title {
    user-select: none;
  }

  .panel-body {
    flex: 1;
    overflow-y: auto;
    padding: 8px 0;
  }

  .empty-msg {
    padding: 12px;
    margin: 0;
    font-size: 12px;
    color: var(--cr-text-muted);
  }

  /* TOC */
  .toc-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .toc-item {
    display: flex;
    align-items: center;
    gap: 6px;
    padding-top: 4px;
    padding-bottom: 4px;
    padding-right: 8px;
    font-size: 12px;
    color: var(--cr-text);
    cursor: pointer;
    border-radius: var(--cr-radius-sm);
  }

  .toc-item:hover {
    background: var(--cr-hover);
  }

  .toc-level {
    font-size: 9px;
    font-weight: 700;
    color: var(--cr-text-muted);
    min-width: 18px;
    text-align: center;
  }

  .toc-text {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Stats */
  .stats-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
    padding: 8px 12px;
  }

  .stat-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 12px 8px;
    border-radius: var(--cr-radius);
    background: var(--cr-bg-input);
    border: 1px solid var(--cr-border);
  }

  .stat-value {
    font-size: 18px;
    font-weight: 700;
    color: var(--cr-text);
  }

  .stat-label {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--cr-text-muted);
    margin-top: 2px;
  }

  /* Panel action button (e.g. refresh) */
  .panel-action {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border: none;
    border-radius: var(--cr-radius-sm);
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
  }

  .panel-action:hover {
    color: var(--cr-text);
    background: var(--cr-hover);
  }

  .panel-action:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .panel-action.active {
    color: var(--cr-accent);
  }

  /* Warnings */
  .warnings-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 24px 12px;
    color: #22c55e;
    font-size: 12px;
  }

  .warnings-empty p {
    margin: 0;
    color: var(--cr-text-muted);
  }

  .warnings-list {
    padding: 0 8px;
  }

  .warning-group {
    margin-bottom: 8px;
  }

  .warning-file {
    font-size: 11px;
    font-weight: 600;
    color: var(--cr-text-muted);
    padding: 4px 4px 2px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .warning-item {
    display: flex;
    flex-direction: column;
    gap: 1px;
    padding: 4px 4px 4px 10px;
    border-left: 2px solid #f59e0b;
    margin: 2px 0;
    border-radius: 0 var(--cr-radius-sm) var(--cr-radius-sm) 0;
  }

  .warning-item.warning-error {
    border-left-color: var(--cr-danger);
  }

  .warning-field {
    font-size: 10px;
    font-weight: 600;
    color: var(--cr-text-muted);
  }

  .warning-msg {
    font-size: 12px;
    color: var(--cr-text);
    word-break: break-word;
  }

  /* Badge on icon button */
  .icon-btn-badge-wrap {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .badge {
    position: absolute;
    top: -6px;
    right: -8px;
    min-width: 14px;
    height: 14px;
    padding: 0 3px;
    border-radius: 7px;
    background: #f59e0b;
    color: #000;
    font-size: 9px;
    font-weight: 700;
    line-height: 14px;
    text-align: center;
  }
</style>
