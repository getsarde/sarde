<script>
  import { tabs, doc, switchToTab, closeTabById } from '../stores/app.svelte.js'

  // Context menu state
  let ctx = $state({ open: false, x: 0, y: 0, tabId: null })

  function openCtx(e, id) {
    e.preventDefault()
    ctx = { open: true, x: e.clientX, y: e.clientY, tabId: id }
  }

  function closeCtx() { ctx.open = false }

  function ctxCloseOthers(id) {
    const keep = tabs.items.find(t => t.id === id)
    if (!keep) { closeCtx(); return }
    tabs.items.splice(0, tabs.items.length, keep)
    if (tabs.activeId !== id) {
      tabs.activeId = id
      doc.filePath = keep.path
      doc.contentPath = keep.contentPath ?? ''
      doc.content = keep.cachedContent ?? ''
      doc.dirty = keep.dirty ?? false
    }
    closeCtx()
  }

  function ctxCloseAll() {
    tabs.items.splice(0, tabs.items.length)
    tabs.activeId = null
    doc.filePath = ''
    doc.contentPath = ''
    doc.content = ''
    doc.dirty = false
    doc.wordCount = 0
    doc.readingTime = 0
    closeCtx()
  }

  async function ctxCopyPath(id) {
    const tab = tabs.items.find(t => t.id === id)
    if (tab) await navigator.clipboard.writeText(tab.path).catch(() => {})
    closeCtx()
  }

  function onGlobalKeydown(e) {
    const ctrl = e.ctrlKey || e.metaKey
    if (ctrl && e.key === 'Tab') {
      e.preventDefault()
      const idx = tabs.items.findIndex(t => t.id === tabs.activeId)
      if (idx !== -1 && tabs.items.length > 1) {
        switchToTab(tabs.items[(idx + 1) % tabs.items.length].id)
      }
    } else if (ctrl && e.key === 'w') {
      e.preventDefault()
      if (tabs.activeId) closeTabById(tabs.activeId)
    }
  }
</script>

<svelte:window onkeydown={onGlobalKeydown} />

<div class="tab-bar">
  <div class="tab-list">
    {#each tabs.items as tab (tab.id)}
      <div
        class="tab"
        class:active={tab.id === tabs.activeId}
        role="tab"
        tabindex="0"
        aria-selected={tab.id === tabs.activeId}
        onclick={() => switchToTab(tab.id)}
        onkeydown={(e) => e.key === 'Enter' && switchToTab(tab.id)}
        oncontextmenu={(e) => openCtx(e, tab.id)}
      >
        {#if tab.dirty}<span class="tab-dot"></span>{/if}
        <span class="tab-name">{tab.name}</span>
        <button
          class="tab-close"
          onclick={(e) => { e.stopPropagation(); closeTabById(tab.id) }}
          aria-label="Close tab"
        >&times;</button>
      </div>
    {/each}
  </div>

</div>

<!-- Right-click context menu -->
{#if ctx.open}
  <div
    class="ctx-overlay"
    onclick={closeCtx}
    onkeydown={(e) => e.key === 'Escape' && closeCtx()}
    role="presentation"
  ></div>
  <div class="ctx-menu" style="left: {ctx.x}px; top: {ctx.y}px">
    <button class="ctx-item" onclick={() => { closeTabById(ctx.tabId); closeCtx() }}>Close</button>
    <button class="ctx-item" onclick={() => ctxCloseOthers(ctx.tabId)}>Close Others</button>
    <button class="ctx-item" onclick={ctxCloseAll}>Close All</button>
    <div class="ctx-sep"></div>
    <button class="ctx-item" onclick={() => ctxCopyPath(ctx.tabId)}>Copy Path</button>
  </div>
{/if}

<style>
  .tab-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 38px;
    background: var(--cr-bg-surface);
    border-bottom: 1px solid var(--cr-border);
    padding: 0 4px;
    gap: 8px;
    user-select: none;
  }

  .tab-list {
    display: flex;
    align-items: center;
    gap: 2px;
    overflow-x: auto;
    flex: 1;
    min-width: 0;
  }

  .tab {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    border: none;
    background: transparent;
    color: var(--cr-text-muted);
    font-size: 12px;
    border-radius: 4px 4px 0 0;
    cursor: pointer;
    white-space: nowrap;
    position: relative;
    transition: background 0.15s, color 0.15s;
  }

  .tab:hover {
    background: var(--cr-bg-elevated);
    color: var(--cr-text);
  }

  .tab.active {
    background: var(--cr-bg-base);
    color: var(--cr-text);
  }

  .tab-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--cr-accent);
    flex-shrink: 0;
  }

  .tab-name {
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .tab-close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    border: none;
    background: transparent;
    color: var(--cr-text-muted);
    font-size: 14px;
    line-height: 1;
    border-radius: 3px;
    cursor: pointer;
    padding: 0;
    opacity: 0;
    transition: opacity 0.1s, background 0.1s;
  }

  .tab:hover .tab-close {
    opacity: 1;
  }

  .tab-close:hover {
    background: var(--cr-bg-surface);
    color: var(--cr-text);
  }

  /* Context menu */
  .ctx-overlay {
    position: fixed;
    inset: 0;
    z-index: 299;
  }

  .ctx-menu {
    position: fixed;
    z-index: 300;
    min-width: 160px;
    background: var(--cr-bg-surface);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    padding: 4px;
    font-size: 13px;
  }

  .ctx-item {
    display: block;
    width: 100%;
    padding: 6px 10px;
    border: none;
    background: transparent;
    color: var(--cr-text);
    text-align: left;
    border-radius: var(--cr-radius-sm);
    cursor: pointer;
  }

  .ctx-item:hover {
    background: var(--cr-bg-elevated);
  }

  .ctx-sep {
    height: 1px;
    background: var(--cr-border);
    margin: 4px 0;
  }
</style>
