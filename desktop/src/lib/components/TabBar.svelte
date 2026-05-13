<script>
  import { tabs, doc, switchToTab, closeTabById } from '../stores/app.svelte.js'
  import { ContextMenu } from 'bits-ui'

  function ctxCloseOthers(id) {
    const keep = tabs.items.find(t => t.id === id)
    if (!keep) return
    tabs.items.splice(0, tabs.items.length, keep)
    if (tabs.activeId !== id) {
      tabs.activeId = id
      doc.filePath = keep.path
      doc.contentPath = keep.contentPath ?? ''
      doc.content = keep.cachedContent ?? ''
      doc.dirty = keep.dirty ?? false
    }
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
  }

  async function ctxCopyPath(id) {
    const tab = tabs.items.find(t => t.id === id)
    if (tab) await navigator.clipboard.writeText(tab.path).catch(() => {})
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
      <ContextMenu.Root>
        <ContextMenu.Trigger class="tab-ctx-trigger">
          <div
            class="tab"
            class:active={tab.id === tabs.activeId}
            role="tab"
            tabindex="0"
            aria-selected={tab.id === tabs.activeId}
            onclick={() => switchToTab(tab.id)}
            onkeydown={(e) => e.key === 'Enter' && switchToTab(tab.id)}
          >
            {#if tab.dirty}<span class="tab-dot"></span>{/if}
            <span class="tab-name">{tab.name}</span>
            <button
              class="tab-close"
              onclick={(e) => { e.stopPropagation(); closeTabById(tab.id) }}
              aria-label="Close tab"
            >&times;</button>
          </div>
        </ContextMenu.Trigger>
        <ContextMenu.Portal>
          <ContextMenu.Content class="ctx-menu">
            <ContextMenu.Item class="ctx-item" onSelect={() => closeTabById(tab.id)}>Close</ContextMenu.Item>
            <ContextMenu.Item class="ctx-item" onSelect={() => ctxCloseOthers(tab.id)}>Close Others</ContextMenu.Item>
            <ContextMenu.Item class="ctx-item" onSelect={ctxCloseAll}>Close All</ContextMenu.Item>
            <ContextMenu.Separator class="ctx-sep" />
            <ContextMenu.Item class="ctx-item" onSelect={() => ctxCopyPath(tab.id)}>Copy Path</ContextMenu.Item>
          </ContextMenu.Content>
        </ContextMenu.Portal>
      </ContextMenu.Root>
    {/each}
  </div>
</div>

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

  :global(.tab-ctx-trigger) {
    display: contents;
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

  :global(.ctx-menu) {
    min-width: 160px;
    background: var(--cr-bg-surface);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    padding: 4px;
    font-size: 13px;
    z-index: 300;
  }

  :global(.ctx-item) {
    display: block;
    width: 100%;
    padding: 6px 10px;
    border: none;
    background: transparent;
    color: var(--cr-text);
    text-align: left;
    border-radius: var(--cr-radius-sm);
    cursor: pointer;
    font-size: 13px;
  }

  :global(.ctx-item[data-highlighted]) {
    background: var(--cr-bg-elevated);
    outline: none;
  }

  :global(.ctx-sep) {
    height: 1px;
    background: var(--cr-border);
    margin: 4px 0;
  }
</style>
