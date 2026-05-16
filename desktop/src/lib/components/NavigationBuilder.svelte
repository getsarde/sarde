<script>
  import { readNav, saveNav, deleteNav } from '../api.js'
  import { addToast, project } from '../stores/app.svelte.js'
  import { GripVertical, Plus, Trash2, ExternalLink, FolderOpen, FileText, ChevronDown, ChevronRight, RotateCcw, Loader, Pencil, Check, X } from 'lucide-svelte'

  let items = $state([])
  let source = $state('auto')
  let loading = $state(true)
  let saving = $state(false)

  let dragItem = $state(null)
  let dragOverId = $state(null)
  let dragPosition = $state(null)
  let collapsedGroups = $state(new Set())

  let editingId = $state(null)
  let editLabel = $state('')
  let editPath = $state('')

  $effect(() => {
    const _path = project.contentPath
    load()
    return () => clearTimeout(saveTimer)
  })

  async function load() {
    loading = true
    try {
      const result = await readNav()
      source = result.source
      items = Array.isArray(result.items) ? assignIds(result.items) : []
    } catch {
      items = []
      source = 'auto'
    } finally {
      loading = false
    }
  }

  function assignIds(arr) {
    return arr.map(item => ({
      ...item,
      id: item.id || crypto.randomUUID(),
      children: item.children ? assignIds(item.children) : undefined,
    }))
  }

  function stripIds(arr) {
    return arr.map(item => {
      const { id, ...rest } = item
      if (rest.children) {
        rest.children = stripIds(rest.children)
      }
      return rest
    })
  }

  async function save() {
    saving = true
    try {
      await saveNav(stripIds(items))
      source = 'file'
      addToast('success', 'Navigation saved')
    } catch {
      addToast('error', 'Failed to save navigation')
    } finally {
      saving = false
    }
  }

  async function resetToAuto() {
    try {
      await deleteNav()
      await load()
      addToast('success', 'Navigation reset to auto-generated')
    } catch {
      addToast('error', 'Failed to reset navigation')
    }
  }

  function addItem() {
    items = [...items, {
      id: crypto.randomUUID(),
      label: 'New Link',
      path: '/',
      auto: false,
    }]
    scheduleSave()
  }

  function addGroup() {
    items = [...items, {
      id: crypto.randomUUID(),
      label: 'New Group',
      path: '',
      auto: false,
      children: [],
    }]
    scheduleSave()
  }

  function removeItem(id) {
    items = removeFromTree(items, id)
    scheduleSave()
  }

  function removeFromTree(arr, id) {
    return arr
      .filter(item => item.id !== id)
      .map(item => ({
        ...item,
        children: item.children ? removeFromTree(item.children, id) : undefined,
      }))
  }

  function startEdit(item) {
    editingId = item.id
    editLabel = item.label
    editPath = item.path || ''
  }

  function confirmEdit(item) {
    item.label = editLabel
    item.path = editPath
    editingId = null
    items = [...items]
    scheduleSave()
  }

  function cancelEdit() {
    editingId = null
  }

  let saveTimer = null
  function scheduleSave() {
    clearTimeout(saveTimer)
    saveTimer = setTimeout(save, 600)
  }

  function toggleGroup(id) {
    const next = new Set(collapsedGroups)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    collapsedGroups = next
  }

  // --- HTML5 Drag & Drop ---
  function onDragStart(e, item, parentArr) {
    dragItem = { item, parentArr }
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', item.id)
  }

  function onDragOver(e, targetId, position) {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    dragOverId = targetId
    dragPosition = position
  }

  function onDragLeave() {
    dragOverId = null
    dragPosition = null
  }

  function onDrop(e, targetItem, targetArr, position) {
    e.preventDefault()
    if (!dragItem || dragItem.item.id === targetItem.id) {
      dragOverId = null
      dragPosition = null
      dragItem = null
      return
    }

    items = removeFromTree(items, dragItem.item.id)

    const freshTargetArr = findParentArr(items, targetItem.id) || items
    const targetIdx = freshTargetArr.findIndex(i => i.id === targetItem.id)

    if (position === 'inside' && targetItem.children) {
      targetItem.children = [...targetItem.children, dragItem.item]
    } else if (position === 'before') {
      freshTargetArr.splice(targetIdx, 0, dragItem.item)
    } else {
      freshTargetArr.splice(targetIdx + 1, 0, dragItem.item)
    }

    items = [...items]
    dragOverId = null
    dragPosition = null
    dragItem = null
    scheduleSave()
  }

  function findParentArr(arr, id) {
    for (const item of arr) {
      if (item.id === id) return arr
      if (item.children) {
        const found = findParentArr(item.children, id)
        if (found) return found
      }
    }
    return null
  }

  function onDragEnd() {
    dragItem = null
    dragOverId = null
    dragPosition = null
  }

  function isExternal(path) {
    return path && (path.startsWith('http://') || path.startsWith('https://'))
  }
</script>

{#if loading}
  <div class="nav-loading"><Loader size={16} /><span>Loading navigation…</span></div>
{:else}
  <div class="nav-builder">
    <div class="nav-toolbar">
      <span class="nav-source-badge" class:auto={source === 'auto'}>
        {source === 'auto' ? 'Auto-generated' : 'Custom'}
      </span>
      <div class="nav-toolbar-actions">
        <button class="nav-tool-btn" onclick={addItem} title="Add link">
          <Plus size={13} /> Link
        </button>
        <button class="nav-tool-btn" onclick={addGroup} title="Add group">
          <Plus size={13} /> Group
        </button>
        {#if source === 'file'}
          <button class="nav-tool-btn reset" onclick={resetToAuto} title="Reset to auto-generated">
            <RotateCcw size={13} /> Reset
          </button>
        {/if}
      </div>
    </div>

    <div class="nav-tree">
      {#if items.length === 0}
        <p class="nav-empty">No navigation items. Add links or groups above.</p>
      {/if}

      {#each items as item, idx (item.id)}
        {@const isGroup = Array.isArray(item.children)}
        {@const isCollapsed = collapsedGroups.has(item.id)}
        {@const isEditing = editingId === item.id}
        <div
          class="nav-item"
          class:drop-before={dragOverId === item.id && dragPosition === 'before'}
          class:drop-after={dragOverId === item.id && dragPosition === 'after'}
          class:drop-inside={dragOverId === item.id && dragPosition === 'inside'}
          class:auto-item={item.auto}
        >
          <div class="nav-item-row" role="listitem"
            draggable={!item.auto}
            ondragstart={(e) => onDragStart(e, item, items)}
            ondragend={onDragEnd}
            ondragover={(e) => onDragOver(e, item.id, isGroup ? 'inside' : 'after')}
            ondragleave={onDragLeave}
            ondrop={(e) => onDrop(e, item, items, isGroup ? 'inside' : 'after')}
          >
            {#if !item.auto}
              <span class="drag-handle"><GripVertical size={12} /></span>
            {:else}
              <span class="drag-handle locked"></span>
            {/if}

            {#if isGroup}
              <button class="collapse-toggle" onclick={() => toggleGroup(item.id)}>
                {#if isCollapsed}
                  <ChevronRight size={12} />
                {:else}
                  <ChevronDown size={12} />
                {/if}
              </button>
              <FolderOpen size={13} class="nav-icon" />
            {:else}
              {#if isExternal(item.path)}
                <ExternalLink size={13} class="nav-icon" />
              {:else}
                <FileText size={13} class="nav-icon" />
              {/if}
            {/if}

            {#if isEditing}
              <input
                class="edit-input label"
                type="text"
                bind:value={editLabel}
                onkeydown={(e) => { if (e.key === 'Enter') confirmEdit(item); if (e.key === 'Escape') cancelEdit() }}
              />
              {#if !isGroup}
                <input
                  class="edit-input path"
                  type="text"
                  bind:value={editPath}
                  placeholder="/path/"
                  onkeydown={(e) => { if (e.key === 'Enter') confirmEdit(item); if (e.key === 'Escape') cancelEdit() }}
                />
              {/if}
              <button class="nav-action-btn confirm" onclick={() => confirmEdit(item)}><Check size={12} /></button>
              <button class="nav-action-btn" onclick={cancelEdit}><X size={12} /></button>
            {:else}
              <span class="nav-label">{item.label}</span>
              {#if item.path}
                <span class="nav-path">{item.path}</span>
              {/if}
              {#if !item.auto}
                <button class="nav-action-btn" onclick={() => startEdit(item)} title="Edit"><Pencil size={11} /></button>
                <button class="nav-action-btn delete" onclick={() => removeItem(item.id)} title="Remove"><Trash2 size={11} /></button>
              {/if}
            {/if}
          </div>

          <!-- Drop zones for reordering -->
          <div
            class="drop-zone top" role="separator"
            ondragover={(e) => onDragOver(e, item.id, 'before')}
            ondragleave={onDragLeave}
            ondrop={(e) => onDrop(e, item, items, 'before')}
          ></div>
          <div
            class="drop-zone bottom" role="separator"
            ondragover={(e) => onDragOver(e, item.id, 'after')}
            ondragleave={onDragLeave}
            ondrop={(e) => onDrop(e, item, items, 'after')}
          ></div>

          {#if isGroup && !isCollapsed}
            <div class="nav-children">
              {#if item.children.length === 0}
                <p class="nav-children-empty">Drop items here</p>
              {/if}
              {#each item.children as child, childIdx (child.id)}
                {@const isChildEditing = editingId === child.id}
                <div
                  class="nav-item child"
                  class:drop-before={dragOverId === child.id && dragPosition === 'before'}
                  class:drop-after={dragOverId === child.id && dragPosition === 'after'}
                  class:auto-item={child.auto}
                >
                  <div class="nav-item-row" role="listitem"
                    draggable={!child.auto}
                    ondragstart={(e) => onDragStart(e, child, item.children)}
                    ondragend={onDragEnd}
                    ondragover={(e) => onDragOver(e, child.id, 'after')}
                    ondragleave={onDragLeave}
                    ondrop={(e) => onDrop(e, child, item.children, 'after')}
                  >
                    {#if !child.auto}
                      <span class="drag-handle"><GripVertical size={12} /></span>
                    {:else}
                      <span class="drag-handle locked"></span>
                    {/if}
                    {#if isExternal(child.path)}
                      <ExternalLink size={13} class="nav-icon" />
                    {:else}
                      <FileText size={13} class="nav-icon" />
                    {/if}

                    {#if isChildEditing}
                      <input
                        class="edit-input label"
                        type="text"
                        bind:value={editLabel}
                        onkeydown={(e) => { if (e.key === 'Enter') confirmEdit(child); if (e.key === 'Escape') cancelEdit() }}
                      />
                      <input
                        class="edit-input path"
                        type="text"
                        bind:value={editPath}
                        placeholder="/path/"
                        onkeydown={(e) => { if (e.key === 'Enter') confirmEdit(child); if (e.key === 'Escape') cancelEdit() }}
                      />
                      <button class="nav-action-btn confirm" onclick={() => confirmEdit(child)}><Check size={12} /></button>
                      <button class="nav-action-btn" onclick={cancelEdit}><X size={12} /></button>
                    {:else}
                      <span class="nav-label">{child.label}</span>
                      {#if child.path}
                        <span class="nav-path">{child.path}</span>
                      {/if}
                      {#if !child.auto}
                        <button class="nav-action-btn" onclick={() => startEdit(child)} title="Edit"><Pencil size={11} /></button>
                        <button class="nav-action-btn delete" onclick={() => removeItem(child.id)} title="Remove"><Trash2 size={11} /></button>
                      {/if}
                    {/if}
                  </div>

                  <div
                    class="drop-zone top" role="separator"
                    ondragover={(e) => onDragOver(e, child.id, 'before')}
                    ondragleave={onDragLeave}
                    ondrop={(e) => onDrop(e, child, item.children, 'before')}
                  ></div>
                  <div
                    class="drop-zone bottom" role="separator"
                    ondragover={(e) => onDragOver(e, child.id, 'after')}
                    ondragleave={onDragLeave}
                    ondrop={(e) => onDrop(e, child, item.children, 'after')}
                  ></div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/each}
    </div>

    {#if saving}
      <div class="nav-saving"><Loader size={12} /> Saving…</div>
    {/if}
  </div>
{/if}

<style>
  .nav-loading {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--cr-text-muted);
    font-size: 13px;
    padding: 16px 0;
  }

  .nav-builder {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .nav-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .nav-source-badge {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    padding: 3px 8px;
    border-radius: 10px;
    background: var(--cr-active);
    color: var(--cr-accent);
  }

  .nav-source-badge.auto {
    background: rgba(166, 227, 161, 0.1);
    color: var(--cr-success);
  }

  .nav-toolbar-actions {
    display: flex;
    gap: 4px;
  }

  .nav-tool-btn {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 10px;
    font-size: 11px;
    font-family: inherit;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
    background: var(--cr-bg-elevated);
    color: var(--cr-text-muted);
    cursor: pointer;
  }

  .nav-tool-btn:hover {
    color: var(--cr-text);
    border-color: var(--cr-accent);
  }

  .nav-tool-btn.reset {
    color: var(--cr-warning);
    border-color: var(--cr-warning);
  }

  .nav-tool-btn.reset:hover {
    background: rgba(249, 226, 175, 0.08);
  }

  .nav-tree {
    display: flex;
    flex-direction: column;
    gap: 1px;
    max-height: 320px;
    overflow-y: auto;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    padding: 4px;
    background: var(--cr-bg-input);
  }

  .nav-empty,
  .nav-children-empty {
    margin: 0;
    padding: 12px;
    text-align: center;
    font-size: 12px;
    color: var(--cr-text-muted);
  }

  .nav-children-empty {
    padding: 6px 12px;
    font-size: 11px;
    font-style: italic;
  }

  .nav-item {
    position: relative;
    border-radius: var(--cr-radius-sm);
  }

  .nav-item.drop-before {
    box-shadow: inset 0 2px 0 var(--cr-accent);
  }

  .nav-item.drop-after {
    box-shadow: inset 0 -2px 0 var(--cr-accent);
  }

  .nav-item.drop-inside {
    background: var(--cr-active);
  }

  .nav-item.auto-item {
    opacity: 0.7;
  }

  .nav-item-row {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 8px;
    border-radius: var(--cr-radius-sm);
    cursor: default;
  }

  .nav-item-row:hover {
    background: var(--cr-hover);
  }

  .nav-item-row[draggable="true"] {
    cursor: grab;
  }

  .nav-item-row[draggable="true"]:active {
    cursor: grabbing;
  }

  .drag-handle {
    display: flex;
    align-items: center;
    color: var(--cr-text-muted);
    flex-shrink: 0;
    width: 12px;
  }

  .drag-handle.locked {
    visibility: hidden;
  }

  .nav-item-row :global(.nav-icon) {
    flex-shrink: 0;
    color: var(--cr-text-muted);
  }

  .collapse-toggle {
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
    padding: 0;
  }

  .collapse-toggle:hover {
    color: var(--cr-text);
  }

  .nav-label {
    font-size: 12px;
    font-weight: 500;
    color: var(--cr-text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .nav-path {
    flex: 1;
    font-size: 11px;
    color: var(--cr-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .nav-action-btn {
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
    padding: 0;
    opacity: 0;
    transition: opacity 0.1s;
  }

  .nav-item-row:hover .nav-action-btn {
    opacity: 1;
  }

  .nav-action-btn:hover {
    color: var(--cr-text);
    background: var(--cr-hover);
  }

  .nav-action-btn.delete:hover {
    color: var(--cr-danger);
    background: rgba(243, 139, 168, 0.1);
  }

  .nav-action-btn.confirm:hover {
    color: var(--cr-success);
  }

  .edit-input {
    padding: 2px 6px;
    font-size: 12px;
    font-family: inherit;
    border: 1px solid var(--cr-accent);
    border-radius: var(--cr-radius-sm);
    background: var(--cr-bg-base);
    color: var(--cr-text);
    outline: none;
  }

  .edit-input.label {
    width: 120px;
  }

  .edit-input.path {
    flex: 1;
    min-width: 0;
  }

  .drop-zone {
    position: absolute;
    left: 0;
    right: 0;
    height: 6px;
    z-index: 5;
  }

  .drop-zone.top {
    top: -3px;
  }

  .drop-zone.bottom {
    bottom: -3px;
  }

  .nav-children {
    padding-left: 24px;
  }

  .nav-saving {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: var(--cr-text-muted);
    padding: 4px 0;
  }
</style>
