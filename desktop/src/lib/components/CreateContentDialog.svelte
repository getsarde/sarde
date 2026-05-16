<script>
  import { ui, tabs, doc, addToast, switchToTab } from '../stores/app.svelte.js'
  import { createContent, getCollections } from '../api.js'
  import AppDialog from './primitives/AppDialog.svelte'
  import { onMount } from 'svelte'

  let open = $derived(!!ui.createContentType)
  let contentType = $derived(ui.createContentType || 'blog')

  let title = $state('')
  let collection = $state('')
  let creating = $state(false)
  let collections = $state([])

  const TYPE_LABELS = { blog: 'Blog Post', docs: 'Doc Page', page: 'Page' }
  const TYPE_COLLECTIONS = { blog: ['posts', 'blog', 'articles'], docs: ['docs', 'documentation', 'guides'] }

  $effect(() => {
    if (open) {
      title = ''
      collection = ''
      loadCollections()
    }
  })

  async function loadCollections() {
    try {
      const cols = await getCollections()
      collections = cols || []
      // Auto-select the first matching collection for this content type
      const preferred = TYPE_COLLECTIONS[contentType] || []
      const match = collections.find(c => preferred.includes(c.name))
      collection = match?.name || collections[0]?.name || ''
    } catch {
      collections = []
    }
  }

  function slugify(text) {
    return text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
  }

  let slug = $derived(slugify(title))
  let canCreate = $derived(title.trim().length > 0 && collection.length > 0)

  function close() {
    ui.createContentType = null
  }

  async function handleCreate() {
    if (!canCreate || creating) return
    creating = true
    try {
      const file = await createContent(collection, title.trim())
      addToast('success', `Created ${file.title || title}`)

      // Open the new file in a tab
      const id = crypto.randomUUID()
      const absPath = (file.absPath || file.path).replace(/\\/g, '/')
      tabs.items.push({
        id,
        name: file.title || title,
        path: absPath,
        contentPath: file.path,
        dirty: false,
        cachedContent: file.raw || '',
      })
      switchToTab(id)

      close()
    } catch (e) {
      addToast('error', `Failed to create: ${e}`)
    } finally {
      creating = false
    }
  }

  function onKeydown(e) {
    if (e.key === 'Enter' && canCreate && !creating) {
      e.preventDefault()
      handleCreate()
    }
  }
</script>

<AppDialog {open} onOpenChange={(v) => { if (!v) close() }} ariaLabel="Create content" width="420px">
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="create-dialog" onkeydown={onKeydown}>
    <div class="dialog-header">
      <h3>New {TYPE_LABELS[contentType] || 'Content'}</h3>
    </div>

    <div class="dialog-body">
      <div class="field">
        <label for="create-title">Title</label>
        <!-- svelte-ignore a11y_autofocus -->
        <input
          id="create-title"
          type="text"
          bind:value={title}
          placeholder="Enter a title..."
          autofocus
        />
        {#if slug}
          <span class="slug-preview">Slug: {slug}</span>
        {/if}
      </div>

      <div class="field">
        <label for="create-collection">Collection</label>
        <select id="create-collection" bind:value={collection}>
          {#each collections as col}
            <option value={col.name}>{col.title} ({col.pageCount} pages)</option>
          {/each}
        </select>
      </div>
    </div>

    <div class="dialog-footer">
      <button class="btn btn-secondary" onclick={close}>Cancel</button>
      <button class="btn btn-primary" onclick={handleCreate} disabled={!canCreate || creating}>
        {creating ? 'Creating...' : 'Create'}
      </button>
    </div>
  </div>
</AppDialog>

<style>
  .create-dialog {
    display: flex;
    flex-direction: column;
  }

  .dialog-header {
    padding: 16px 20px 0;
  }

  .dialog-header h3 {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
    color: var(--cr-text);
  }

  .dialog-body {
    padding: 16px 20px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .field label {
    font-size: 11px;
    font-weight: 600;
    color: var(--cr-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .field input,
  .field select {
    padding: 7px 10px;
    font-size: 13px;
    color: var(--cr-text);
    background: var(--cr-bg-input);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
    outline: none;
    font-family: inherit;
  }

  .field input:focus,
  .field select:focus {
    border-color: var(--cr-accent);
  }

  .slug-preview {
    font-size: 11px;
    color: var(--cr-text-muted);
    font-family: var(--cr-font-mono);
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 0 20px 16px;
  }

  .btn {
    padding: 7px 16px;
    border: none;
    border-radius: var(--cr-radius-sm);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
  }

  .btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .btn-primary {
    background: var(--cr-accent);
    color: var(--cr-bg-base);
  }

  .btn-primary:hover:not(:disabled) {
    filter: brightness(1.1);
  }

  .btn-secondary {
    background: var(--cr-bg-elevated);
    color: var(--cr-text);
  }

  .btn-secondary:hover {
    background: var(--cr-bg-surface);
  }
</style>
