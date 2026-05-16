<script>
  import { listTaxonomies, renameTaxonomy, deleteTaxonomy } from '../api.js'
  import { addToast, project } from '../stores/app.svelte.js'
  import { Tag, Folder, Pencil, Trash2, Check, X, Loader, RefreshCw } from 'lucide-svelte'

  let data = $state({ tags: [], categories: [] })
  let loading = $state(true)
  let activeTab = $state('tags')

  let editingItem = $state(null)
  let editValue = $state('')

  $effect(() => {
    const _path = project.contentPath
    load()
  })

  async function load() {
    loading = true
    try {
      data = await listTaxonomies()
    } catch {
      data = { tags: [], categories: [] }
    } finally {
      loading = false
    }
  }

  function startEdit(item) {
    editingItem = item.name
    editValue = item.name
  }

  async function confirmRename(item) {
    const newName = editValue.trim()
    editingItem = null
    if (!newName || newName === item.name) return
    try {
      const count = await renameTaxonomy(activeTab, item.name, newName)
      addToast('success', `Renamed "${item.name}" → "${newName}" in ${count} file${count !== 1 ? 's' : ''}`)
      await load()
    } catch (e) {
      addToast('error', `Rename failed: ${e}`)
    }
  }

  async function deleteItem(item) {
    try {
      const count = await deleteTaxonomy(activeTab, item.name)
      addToast('success', `Removed "${item.name}" from ${count} file${count !== 1 ? 's' : ''}`)
      await load()
    } catch (e) {
      addToast('error', `Delete failed: ${e}`)
    }
  }

  let items = $derived(activeTab === 'tags' ? data.tags : data.categories)
  let totalItems = $derived(items.length)
</script>

<div class="taxonomy-panel">
  <div class="tax-header">
    <div class="tax-tabs">
      <button class="tax-tab" class:active={activeTab === 'tags'} onclick={() => activeTab = 'tags'}>
        <Tag size={12} /> Tags {data.tags.length > 0 ? `(${data.tags.length})` : ''}
      </button>
      <button class="tax-tab" class:active={activeTab === 'categories'} onclick={() => activeTab = 'categories'}>
        <Folder size={12} /> Categories {data.categories.length > 0 ? `(${data.categories.length})` : ''}
      </button>
    </div>
    <button class="tax-refresh" onclick={load} disabled={loading} title="Refresh">
      <RefreshCw size={12} />
    </button>
  </div>

  {#if loading}
    <div class="tax-loading"><Loader size={14} /> Loading…</div>
  {:else if items.length === 0}
    <p class="tax-empty">No {activeTab} found in content files.</p>
  {:else}
    <div class="tax-list">
      {#each items as item}
        <div class="tax-item">
          {#if editingItem === item.name}
            <input
              class="tax-edit-input"
              type="text"
              bind:value={editValue}
              onkeydown={(e) => { if (e.key === 'Enter') confirmRename(item); if (e.key === 'Escape') editingItem = null }}
            />
            <button class="tax-action confirm" onclick={() => confirmRename(item)}><Check size={11} /></button>
            <button class="tax-action" onclick={() => editingItem = null}><X size={11} /></button>
          {:else}
            <span class="tax-name">{item.name}</span>
            <span class="tax-count">{item.count}</span>
            <button class="tax-action" onclick={() => startEdit(item)} title="Rename"><Pencil size={11} /></button>
            <button class="tax-action delete" onclick={() => deleteItem(item)} title="Remove from all files"><Trash2 size={11} /></button>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .taxonomy-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .tax-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 8px;
    border-bottom: 1px solid var(--cr-border);
  }

  .tax-tabs {
    display: flex;
    gap: 2px;
  }

  .tax-tab {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 8px;
    border: none;
    border-radius: var(--cr-radius-sm);
    background: transparent;
    color: var(--cr-text-muted);
    font-size: 11px;
    font-family: inherit;
    cursor: pointer;
  }

  .tax-tab:hover {
    color: var(--cr-text);
    background: var(--cr-hover);
  }

  .tax-tab.active {
    color: var(--cr-accent);
    background: var(--cr-active);
    font-weight: 600;
  }

  .tax-refresh {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border: none;
    border-radius: var(--cr-radius-sm);
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
  }

  .tax-refresh:hover:not(:disabled) {
    color: var(--cr-text);
    background: var(--cr-hover);
  }

  .tax-refresh:disabled {
    opacity: 0.4;
  }

  .tax-loading {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 12px;
    font-size: 12px;
    color: var(--cr-text-muted);
  }

  .tax-empty {
    padding: 16px 12px;
    margin: 0;
    font-size: 12px;
    color: var(--cr-text-muted);
    text-align: center;
  }

  .tax-list {
    flex: 1;
    overflow-y: auto;
    padding: 4px;
  }

  .tax-item {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    border-radius: var(--cr-radius-sm);
  }

  .tax-item:hover {
    background: var(--cr-hover);
  }

  .tax-name {
    flex: 1;
    font-size: 12px;
    color: var(--cr-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tax-count {
    font-size: 10px;
    font-weight: 600;
    color: var(--cr-accent);
    background: var(--cr-active);
    border-radius: 8px;
    padding: 1px 6px;
    flex-shrink: 0;
  }

  .tax-action {
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
    opacity: 0;
    transition: opacity 0.1s;
    padding: 0;
  }

  .tax-item:hover .tax-action {
    opacity: 1;
  }

  .tax-action:hover {
    color: var(--cr-text);
    background: var(--cr-hover);
  }

  .tax-action.delete:hover {
    color: var(--cr-danger);
    background: rgba(243, 139, 168, 0.1);
  }

  .tax-action.confirm:hover {
    color: var(--cr-success);
  }

  .tax-edit-input {
    flex: 1;
    min-width: 0;
    padding: 2px 6px;
    font-size: 12px;
    font-family: inherit;
    border: 1px solid var(--cr-accent);
    border-radius: var(--cr-radius-sm);
    background: var(--cr-bg-base);
    color: var(--cr-text);
    outline: none;
  }
</style>
