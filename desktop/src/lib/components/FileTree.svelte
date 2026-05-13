<script>
  import { readDir, readTextFile, mkdir, rename } from '@tauri-apps/plugin-fs'
  import { doc, tabs, project, switchToTab, closeTabById, addToast } from '../stores/app.svelte.js'
  import { readContent, createContentFile, createContentDir, deleteContent, renameContent } from '../api.js'
  import { ContextMenu } from 'bits-ui'
  import { ChevronRight, Folder, FolderOpen, FileText, File, FilePlus, FolderPlus, Pencil, Trash2, Copy } from 'lucide-svelte'

  let entries = $state([])
  let expandedDirs = $state(new Set())

  // Inline rename
  let renamingPath = $state(null)
  let renameValue = $state('')

  // Inline create (new file / folder)
  let creating = $state(null) // { parentPath, depth, type: 'file'|'folder' }
  let createValue = $state('')

  // Drag-and-drop
  let dragItem = $state(null)
  let dragOverPath = $state(null)

  let { basePath = '' } = $props()

  function rootPath() { return basePath || project.contentPath }

  $effect(() => {
    const path = rootPath()
    if (path) loadTree(path)
  })

  // --- Tree loading ---

  async function loadTree(dir) {
    try {
      entries = buildTree(await readDir(dir), dir)
    } catch (e) {
      console.error('Failed to read directory:', e)
      entries = []
    }
  }

  async function refreshTree() {
    await loadTree(rootPath())
    await reloadExpanded(entries)
  }

  async function reloadExpanded(items) {
    for (const item of items) {
      if (item.isDir && expandedDirs.has(item.path)) {
        try {
          item.children = buildTree(await readDir(item.path), item.path)
          await reloadExpanded(item.children)
        } catch {}
      }
    }
  }

  function buildTree(items, parentPath) {
    return [...items]
      .map(item => ({
        name: item.name,
        path: parentPath + '/' + item.name,
        isDir: item.isDirectory,
        children: [],
      }))
      .sort((a, b) => {
        if (a.isDir && !b.isDir) return -1
        if (!a.isDir && b.isDir) return 1
        return a.name.localeCompare(b.name)
      })
  }

  async function toggleDir(item) {
    if (expandedDirs.has(item.path)) {
      expandedDirs.delete(item.path)
    } else {
      try {
        item.children = buildTree(await readDir(item.path), item.path)
      } catch { item.children = [] }
      expandedDirs.add(item.path)
    }
    expandedDirs = new Set(expandedDirs)
  }

  async function openFile(item) {
    const contentPath = relContentPath(item.path)
    const existing = tabs.items.find(t => t.contentPath === contentPath)
    if (existing) { switchToTab(existing.id); return }
    try {
      const file = await readContent(contentPath)
      const content = file.raw ?? await readTextFile(item.path)
      const absPath = (file.absPath ?? item.path).replace(/\\/g, '/')
      const id = crypto.randomUUID()
      tabs.items = [...tabs.items, {
        id,
        name: item.name,
        path: absPath,
        contentPath: file.path ?? contentPath,
        dirty: false,
        cachedContent: content,
      }]
      tabs.activeId = id
      doc.content = content
      doc.filePath = absPath
      doc.contentPath = file.path ?? contentPath
      doc.dirty = false
      doc.cursorLine = 1
      doc.cursorCol = 1
      doc.wordCount = content.split(/\s+/).filter(w => w).length
      doc.readingTime = Math.max(1, Math.ceil(doc.wordCount / 250))
    } catch (e) {
      addToast('error', 'Could not open file')
    }
  }

  // --- Create new file / folder ---

  export function newFileAtRoot() { startCreate(rootPath(), 0, 'file') }
  export function newFolderAtRoot() { startCreate(rootPath(), 0, 'folder') }

  async function startCreate(parentPath, depth, type) {
    // Auto-expand the parent dir if it isn't already
    if (parentPath !== rootPath() && !expandedDirs.has(parentPath)) {
      const entry = findEntry(entries, parentPath)
      if (entry) await toggleDir(entry)
    }
    creating = { parentPath, depth, type }
    createValue = ''
  }

  async function confirmCreate() {
    if (!creating) return   // guard against double-fire (Enter + blur)
    const name = createValue.trim()
    if (!name) { creating = null; return }
    const parentPath = creating.parentPath
    const type = creating.type
    creating = null           // clear immediately to prevent re-entry
    createValue = ''
    const fullPath = parentPath + '/' + name
    const relPath = relContentPath(fullPath)
    try {
      if (type === 'file') {
        const title = name.replace(/\.\w+$/, '').replace(/^\d+[-_]/, '').replace(/[-_]/g, ' ')
        await createContentFile(relPath, `---\ntitle: ${title}\n---\n`)
        addToast('success', `Created ${name}`)
      } else {
        await createContentDir(relPath)
        addToast('success', `Created folder ${name}`)
      }
      await refreshTree()
    } catch (e) {
      addToast('error', `Failed to create: ${e}`)
    }
  }

  function cancelCreate() {
    creating = null
    createValue = ''
  }

  // --- Rename ---

  function startRename(item) {
    renamingPath = item.path
    renameValue = item.name
  }

  async function confirmRename(item) {
    if (!renamingPath) return   // guard against double-fire (Enter + blur)
    const name = renameValue.trim()
    renamingPath = null
    if (!name || name === item.name) return
    const dir = item.path.substring(0, item.path.lastIndexOf('/'))
    const newPath = dir + '/' + name
    try {
      const oldRel = relContentPath(item.path)
      const newRel = relContentPath(newPath)
      if (item.isDir) {
        await rename(item.path, newPath)
      } else {
        await renameContent(oldRel, newRel)
      }
      // Patch open tab path
      const tab = tabs.items.find(t => t.path === item.path || t.contentPath === oldRel)
      if (tab) {
        tab.path = newPath
        tab.contentPath = newRel
        tab.name = name
        if (doc.filePath === item.path || doc.contentPath === oldRel) {
          doc.filePath = newPath
          doc.contentPath = newRel
        }
      }
      await refreshTree()
    } catch (e) {
      addToast('error', `Rename failed: ${e}`)
    }
  }

  // --- Delete (soft — moves to .trash) ---

  async function deleteItem(item) {
    if (!item.isDir) {
      const contentPath = relContentPath(item.path)
      try {
        await deleteContent(contentPath)
        const tab = tabs.items.find(t => t.path === item.path || t.contentPath === contentPath)
        if (tab) closeTabById(tab.id)
        addToast('info', `"${item.name}" deleted`)
        await refreshTree()
      } catch (e) {
        addToast('error', `Delete failed: ${e}`)
      }
      return
    }

    const root = rootPath()
    const trashDir = root + '/.trash'
    try { await mkdir(trashDir) } catch {}
    const trashPath = trashDir + '/' + Date.now() + '-' + item.name
    try {
      await rename(item.path, trashPath)
      const tab = tabs.items.find(t => t.path === item.path)
      if (tab) closeTabById(tab.id)
      addToast('info', `"${item.name}" moved to .trash`)
      await refreshTree()
    } catch (e) {
      addToast('error', `Delete failed: ${e}`)
    }
  }

  // --- Copy path ---

  function copyPath(item) {
    navigator.clipboard.writeText(item.path)
    addToast('info', 'Path copied to clipboard')
  }

  // --- Drag-and-drop (reorder within same directory) ---

  function onDragStart(e, item) {
    if (item.isDir) return
    dragItem = item
    e.dataTransfer.effectAllowed = 'move'
  }

  function onDragOver(e, item, siblings) {
    if (!dragItem || dragItem.path === item.path || item.isDir) return
    const same = sameDir(dragItem.path, item.path)
    if (!same) return
    e.preventDefault()
    dragOverPath = item.path
  }

  function onDragLeave() {
    dragOverPath = null
  }

  async function onDrop(e, item, siblings) {
    e.preventDefault()
    dragOverPath = null
    if (!dragItem || dragItem.path === item.path) { dragItem = null; return }
    if (!sameDir(dragItem.path, item.path)) { dragItem = null; return }

    const dir = item.path.substring(0, item.path.lastIndexOf('/'))
    const files = siblings.filter(s => !s.isDir)
    const without = files.filter(f => f.path !== dragItem.path)
    const idx = without.findIndex(f => f.path === item.path)
    without.splice(idx === -1 ? without.length : idx, 0, dragItem)

    await renumberFiles(without, dir)
    dragItem = null
    await refreshTree()
  }

  async function renumberFiles(files, dir) {
    const prefixed = files.filter(f => /^\d+[-_]/.test(f.name))
    if (prefixed.length < 2) return
    for (let i = 0; i < prefixed.length; i++) {
      const f = prefixed[i]
      const m = f.name.match(/^(\d+)([-_].*)$/)
      if (!m) continue
      const newName = String(i + 1).padStart(m[1].length, '0') + m[2]
      if (newName !== f.name) {
        try { await renameContent(relContentPath(f.path), relContentPath(dir + '/' + newName)) } catch {}
      }
    }
  }

  // --- Helpers ---

  function sameDir(a, b) {
    return a.substring(0, a.lastIndexOf('/')) === b.substring(0, b.lastIndexOf('/'))
  }

  function findEntry(items, path) {
    for (const item of items) {
      if (item.path === path) return item
      if (item.children?.length) {
        const found = findEntry(item.children, path)
        if (found) return found
      }
    }
    return null
  }

  function relContentPath(absPath) {
    const root = rootPath().replace(/\\/g, '/').replace(/\/+$/, '')
    const full = absPath.replace(/\\/g, '/')
    return full.startsWith(root + '/') ? full.slice(root.length + 1) : full
  }

  /** Svelte action: focus + select the input on mount */
  function autoFocus(node) {
    node.focus()
    node.select()
  }

  function isMd(name) { return name.endsWith('.md') }
</script>

<svelte:window
  onkeydown={(e) => {
    if (e.key === 'Escape') { creating && cancelCreate(); renamingPath && (renamingPath = null) }
  }}
/>

{#snippet inlineInput(depth, placeholder, onconfirm, oncancel)}
  <li class="tree-item">
    <div class="tree-row" style="padding-left: {depth * 14 + 8 + 20}px">
      <input
        class="inline-input"
        type="text"
        {placeholder}
        bind:value={createValue}
        use:autoFocus
        onkeydown={(e) => { if (e.key === 'Enter') onconfirm(); if (e.key === 'Escape') oncancel() }}
        onblur={oncancel}
      />
    </div>
  </li>
{/snippet}

{#snippet renameInput(item)}
  <input
    class="inline-input tree-rename"
    type="text"
    bind:value={renameValue}
    use:autoFocus
    onkeydown={(e) => { if (e.key === 'Enter') confirmRename(item); if (e.key === 'Escape') renamingPath = null }}
    onblur={() => confirmRename(item)}
  />
{/snippet}

{#snippet treeNode(items, depth, parentPath)}
  <ul class="tree-list">
    {#each items as item}
      <li class="tree-item">
        {#if item.isDir}
          <ContextMenu.Root>
            <ContextMenu.Trigger class="tree-ctx-trigger">
              <button
                class="tree-row tree-dir"
                style="padding-left: {depth * 14 + 8}px"
                onclick={() => toggleDir(item)}
              >
                <span class="tree-arrow" class:expanded={expandedDirs.has(item.path)}>
                  <ChevronRight size={12} strokeWidth={2.5} />
                </span>
                <span class="tree-icon">
                  {#if expandedDirs.has(item.path)}<FolderOpen size={16} />{:else}<Folder size={16} />{/if}
                </span>
                {#if renamingPath === item.path}
                  {@render renameInput(item)}
                {:else}
                  <span class="tree-name">{item.name}</span>
                {/if}
              </button>
            </ContextMenu.Trigger>
            <ContextMenu.Portal>
              <ContextMenu.Content class="ctx-menu">
                <ContextMenu.Item class="ctx-item" onSelect={() => startCreate(item.path, depth + 1, 'file')}>
                  <FilePlus size={14} /> New File
                </ContextMenu.Item>
                <ContextMenu.Item class="ctx-item" onSelect={() => startCreate(item.path, depth + 1, 'folder')}>
                  <FolderPlus size={14} /> New Folder
                </ContextMenu.Item>
                <ContextMenu.Separator class="ctx-sep" />
                <ContextMenu.Item class="ctx-item" onSelect={() => startRename(item)}>
                  <Pencil size={14} /> Rename
                </ContextMenu.Item>
                <ContextMenu.Item class="ctx-item ctx-danger" onSelect={() => deleteItem(item)}>
                  <Trash2 size={14} /> Delete
                </ContextMenu.Item>
                <ContextMenu.Separator class="ctx-sep" />
                <ContextMenu.Item class="ctx-item" onSelect={() => copyPath(item)}>
                  <Copy size={14} /> Copy Path
                </ContextMenu.Item>
              </ContextMenu.Content>
            </ContextMenu.Portal>
          </ContextMenu.Root>

          {#if expandedDirs.has(item.path) && item.children}
            {@render treeNode(item.children, depth + 1, item.path)}
          {/if}

        {:else}
          <ContextMenu.Root>
            <ContextMenu.Trigger class="tree-ctx-trigger">
              <button
                class="tree-row tree-file"
                class:active={doc.filePath === item.path}
                class:drag-over={dragOverPath === item.path}
                style="padding-left: {depth * 14 + 8}px"
                draggable="true"
                onclick={() => openFile(item)}
                ondragstart={(e) => onDragStart(e, item)}
                ondragover={(e) => onDragOver(e, item, items)}
                ondragleave={onDragLeave}
                ondrop={(e) => onDrop(e, item, items)}
                ondragend={() => { dragItem = null; dragOverPath = null }}
              >
                <span class="tree-arrow-placeholder"></span>
                <span class="tree-icon">
                  {#if isMd(item.name)}<FileText size={16} />{:else}<File size={16} />{/if}
                </span>
                {#if renamingPath === item.path}
                  {@render renameInput(item)}
                {:else}
                  <span class="tree-name">{item.name}</span>
                {/if}
              </button>
            </ContextMenu.Trigger>
            <ContextMenu.Portal>
              <ContextMenu.Content class="ctx-menu">
                <ContextMenu.Item class="ctx-item" onSelect={() => startRename(item)}>
                  <Pencil size={14} /> Rename
                </ContextMenu.Item>
                <ContextMenu.Item class="ctx-item ctx-danger" onSelect={() => deleteItem(item)}>
                  <Trash2 size={14} /> Delete
                </ContextMenu.Item>
                <ContextMenu.Separator class="ctx-sep" />
                <ContextMenu.Item class="ctx-item" onSelect={() => copyPath(item)}>
                  <Copy size={14} /> Copy Path
                </ContextMenu.Item>
              </ContextMenu.Content>
            </ContextMenu.Portal>
          </ContextMenu.Root>
        {/if}
      </li>
    {/each}

    <!-- Inline create input at end of this directory's list -->
    {#if creating?.parentPath === parentPath}
      {@render inlineInput(
        depth,
        creating.type === 'file' ? 'filename.md' : 'folder-name',
        confirmCreate,
        cancelCreate
      )}
    {/if}
  </ul>
{/snippet}

<div class="file-tree">
  {#if entries.length === 0 && !creating}
    <p class="empty-msg">No files found.</p>
  {:else}
    {@render treeNode(entries, 0, rootPath())}
  {/if}
</div>

<style>
  .file-tree {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    padding: 4px 0;
  }

  .empty-msg {
    padding: 12px;
    font-size: 12px;
    color: var(--cr-text-muted);
    margin: 0;
  }

  .tree-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .tree-item {
    margin: 0;
    padding: 0;
  }

  :global(.tree-ctx-trigger) {
    display: contents;
  }

  .tree-row {
    display: flex;
    align-items: center;
    gap: 4px;
    width: 100%;
    padding-top: 3px;
    padding-bottom: 3px;
    padding-right: 8px;
    border: none;
    background: transparent;
    color: var(--cr-text);
    font-size: 13px;
    cursor: pointer;
    text-align: left;
    white-space: nowrap;
    border-radius: 0;
  }

  .tree-row:hover {
    background: var(--cr-hover);
  }

  .tree-file.active {
    background: var(--cr-active);
    color: var(--cr-accent);
  }

  .tree-file.drag-over {
    border-top: 2px solid var(--cr-accent);
  }

  .tree-arrow {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    flex-shrink: 0;
    transition: transform 0.15s ease;
    color: var(--cr-text-muted);
  }

  .tree-arrow.expanded {
    transform: rotate(90deg);
  }

  .tree-arrow-placeholder {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
  }

  .tree-icon {
    flex-shrink: 0;
    color: var(--cr-text-muted);
  }

  .tree-dir .tree-icon {
    color: var(--cr-accent);
  }

  .tree-name {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Inline inputs (rename / create) */
  .inline-input {
    flex: 1;
    min-width: 0;
    background: var(--cr-bg-input);
    border: 1px solid var(--cr-accent);
    border-radius: 3px;
    color: var(--cr-text);
    font-size: 13px;
    font-family: inherit;
    padding: 1px 5px;
    outline: none;
  }

  .tree-rename {
    margin-left: 2px;
  }

  /* Context menu (portaled — needs :global) */
  :global(.ctx-menu) {
    min-width: 160px;
    background: var(--cr-bg-base);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
    padding: 4px;
    overflow: hidden;
    z-index: 300;
  }

  :global(.ctx-item) {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 6px 10px;
    border: none;
    border-radius: 5px;
    background: transparent;
    color: var(--cr-text);
    font-size: 13px;
    font-family: inherit;
    text-align: left;
    cursor: pointer;
  }

  :global(.ctx-item[data-highlighted]) {
    background: var(--cr-hover);
    outline: none;
  }

  :global(.ctx-danger) {
    color: var(--cr-danger);
  }

  :global(.ctx-danger[data-highlighted]) {
    background: rgba(243, 139, 168, 0.1);
  }

  :global(.ctx-sep) {
    height: 1px;
    background: var(--cr-border);
    margin: 3px 0;
  }
</style>
