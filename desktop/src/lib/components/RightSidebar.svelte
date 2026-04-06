<script>
  import { ui, doc } from '../stores/app.svelte.js'
  import { List, PencilLine, Image, TrendingUp } from 'lucide-svelte'

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

  /** Parse frontmatter properties */
  let frontmatter = $derived.by(() => {
    if (!doc.content) return []
    const fmMatch = doc.content.match(/^---\n([\s\S]*?)\n---/)
    if (!fmMatch) return []
    const entries = []
    for (const line of fmMatch[1].split('\n')) {
      const kv = line.match(/^(\w[\w-]*):\s*(.*)/)
      if (kv) entries.push({ key: kv[1], value: kv[2] })
    }
    return entries
  })

  /** Document statistics */
  let stats = $derived({
    words: doc.wordCount,
    readingTime: doc.readingTime,
    characters: doc.content.length,
    lines: doc.content ? doc.content.split('\n').length : 0,
    headings: headings.length,
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
        <div class="panel-header"><span class="panel-title">Properties</span></div>
        <div class="panel-body">
          {#if frontmatter.length === 0}
            <p class="empty-msg">No frontmatter found.</p>
          {:else}
            <dl class="prop-list">
              {#each frontmatter as prop}
                <div class="prop-row">
                  <dt class="prop-key">{prop.key}</dt>
                  <dd class="prop-value">{prop.value}</dd>
                </div>
              {/each}
            </dl>
          {/if}
        </div>

      {:else if ui.rightPanel === 'assets'}
        <div class="panel-header"><span class="panel-title">Assets</span></div>
        <div class="panel-body">
          <p class="empty-msg">Drag files here or browse to add assets.</p>
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
      {/if}
    </div>
  {/if}

  <div class="icon-strip">
    <button class="icon-btn" class:active={ui.rightPanel === 'toc'}        onclick={() => togglePanel('toc')}        title="Table of Contents"><List       size={20} /></button>
    <button class="icon-btn" class:active={ui.rightPanel === 'properties'} onclick={() => togglePanel('properties')} title="Properties">          <PencilLine size={20} /></button>
    <button class="icon-btn" class:active={ui.rightPanel === 'assets'}     onclick={() => togglePanel('assets')}     title="Assets">              <Image      size={20} /></button>
    <button class="icon-btn" class:active={ui.rightPanel === 'stats'}      onclick={() => togglePanel('stats')}      title="Stats">               <TrendingUp size={20} /></button>
  </div>
</aside>

<style>
  .right-sidebar {
    display: flex;
    height: 100%;
    background: var(--color-surface, #1e1e2e);
    border-left: 1px solid var(--color-border, #313244);
  }

  .icon-strip {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 44px;
    padding: 6px 0;
    gap: 2px;
    background: var(--color-surface-alt, #181825);
    border-left: 1px solid var(--color-border, #313244);
  }

  .icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--color-text-muted, #6c7086);
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
  }

  .icon-btn:hover {
    color: var(--color-text, #cdd6f4);
    background: var(--color-hover, rgba(255, 255, 255, 0.06));
  }

  .icon-btn.active {
    color: var(--color-accent, #89b4fa);
    background: var(--color-active, rgba(137, 180, 250, 0.1));
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
    color: var(--color-text-muted, #6c7086);
    border-bottom: 1px solid var(--color-border, #313244);
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
    color: var(--color-text-muted, #6c7086);
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
    color: var(--color-text, #cdd6f4);
    cursor: pointer;
    border-radius: 4px;
  }

  .toc-item:hover {
    background: var(--color-hover, rgba(255, 255, 255, 0.06));
  }

  .toc-level {
    font-size: 9px;
    font-weight: 700;
    color: var(--color-text-muted, #6c7086);
    min-width: 18px;
    text-align: center;
  }

  .toc-text {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Properties */
  .prop-list {
    margin: 0;
    padding: 0 12px;
  }

  .prop-row {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 6px 0;
    border-bottom: 1px solid var(--color-border, #313244);
  }

  .prop-row:last-child {
    border-bottom: none;
  }

  .prop-key {
    font-size: 11px;
    font-weight: 600;
    color: var(--color-text-muted, #6c7086);
  }

  .prop-value {
    margin: 0;
    font-size: 12px;
    color: var(--color-text, #cdd6f4);
    word-break: break-word;
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
    border-radius: 6px;
    background: var(--color-surface-alt, #181825);
    border: 1px solid var(--color-border, #313244);
  }

  .stat-value {
    font-size: 18px;
    font-weight: 700;
    color: var(--color-text, #cdd6f4);
  }

  .stat-label {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-text-muted, #6c7086);
    margin-top: 2px;
  }
</style>
