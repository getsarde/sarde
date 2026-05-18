<script>
  import { assets, loadAssets, doc, addToast } from '../stores/app.svelte.js'
  import { assetUpload, assetDelete, assetGetThumbnail } from '../api.js'
  import { Upload, Trash2, ImagePlus, FileText, Film, Loader } from 'lucide-svelte'
  import { Tabs } from 'bits-ui'

  let thumbnailCache = $state(new Map())
  let selectedAsset = $state(null)
  let uploading = $state(false)

  let bundleAssets = $derived(assets.items.filter(a => a.bundleOwner))
  let sharedAssets = $derived(assets.items.filter(a => !a.bundleOwner))

  // Load assets when the panel mounts and items are empty.
  $effect(() => {
    if (assets.items.length === 0 && !assets.loading) {
      loadAssets()
    }
  })

  function changeScope(scope) {
    thumbnailCache = new Map()
    selectedAsset = null
    loadAssets(scope)
  }

  async function loadThumbnail(path) {
    if (thumbnailCache.has(path)) return
    thumbnailCache.set(path, 'loading')
    thumbnailCache = new Map(thumbnailCache) // trigger reactivity
    try {
      const result = await assetGetThumbnail(path)
      thumbnailCache.set(path, result)
      thumbnailCache = new Map(thumbnailCache)
    } catch {
      thumbnailCache.set(path, 'error')
      thumbnailCache = new Map(thumbnailCache)
    }
  }

  $effect(() => {
    for (const asset of assets.items) {
      if (isImageMime(asset.mimeType) && !thumbnailCache.has(asset.path)) {
        loadThumbnail(asset.path)
      }
    }
  })

  async function handleUpload(target) {
    if (uploading) return
    uploading = true

    const destination = { target }
    if (target === 'bundle' && doc.contentPath) {
      // Extract directory portion from the content file path.
      const parts = doc.contentPath.replace(/\\/g, '/').split('/')
      // Remove the filename to get the directory.
      parts.pop()
      // Remove 'content/' prefix if present — the Rust side joins with content_dir.
      destination.bundlePath = parts.join('/') || '.'
    }

    try {
      const uploaded = await assetUpload(destination)
      if (uploaded.length > 0) {
        addToast('success', `Uploaded ${uploaded.length} file${uploaded.length > 1 ? 's' : ''}`)
        loadAssets()
      }
    } catch (e) {
      addToast('error', `Upload failed: ${e}`)
    } finally {
      uploading = false
    }
  }

  function insertAsset(asset) {
    if (!doc.contentPath) {
      addToast('warning', 'No file open to insert into')
      return
    }

    let imgPath
    if (asset.path.startsWith('static/')) {
      // Shared assets: SSG convention — static/ maps to site root.
      imgPath = '/' + asset.path.slice(7)
    } else {
      // Bundle asset: use just the filename if same directory.
      imgPath = asset.filename
    }

    const isImage = asset.mimeType.startsWith('image/')
    doc.insertText = isImage ? `![${asset.filename}](${imgPath})` : `[${asset.filename}](${imgPath})`
    selectedAsset = null
    addToast('info', 'Inserted into editor')
  }

  async function deleteAsset(asset) {
    if (!confirm(`Delete "${asset.filename}"? This cannot be undone.`)) return
    try {
      await assetDelete(asset.path)
      addToast('info', `Deleted ${asset.filename}`)
      selectedAsset = null
      thumbnailCache.delete(asset.path)
      thumbnailCache = new Map(thumbnailCache)
      loadAssets()
    } catch (e) {
      addToast('error', `Delete failed: ${e}`)
    }
  }

  function formatSize(bytes) {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  function isImageMime(mime) {
    return mime.startsWith('image/')
  }
</script>

<div class="media-panel">
  <Tabs.Root value={assets.scope} onValueChange={(v) => changeScope(v)}>
    <Tabs.List class="scope-tabs">
      <Tabs.Trigger value="all" class="scope-tab">All</Tabs.Trigger>
      <Tabs.Trigger value="bundle" class="scope-tab">Bundle</Tabs.Trigger>
      <Tabs.Trigger value="shared" class="scope-tab">Shared</Tabs.Trigger>
    </Tabs.List>
  </Tabs.Root>

  <div class="upload-area">
    {#if doc.contentPath}
      <button class="upload-btn" onclick={() => handleUpload('bundle')} disabled={uploading} title="Upload to current page folder">
        <ImagePlus size={13} />
        Page
      </button>
    {/if}
    <button class="upload-btn" onclick={() => handleUpload('shared')} disabled={uploading} title="Upload to static/ folder">
      <Upload size={13} />
      Shared
    </button>
  </div>

  {#if assets.loading}
    <div class="media-status">
      <Loader size={16} />
      <span>Loading...</span>
    </div>
  {:else if assets.items.length === 0}
    <div class="media-status">
      <span>No assets found.</span>
    </div>
  {:else}
    <div class="asset-list">
      {#if (assets.scope === 'all' || assets.scope === 'bundle') && bundleAssets.length > 0}
        <div class="group-label">Bundle</div>
        <div class="asset-grid">
          {#each bundleAssets as asset (asset.path)}
            {@const thumb = thumbnailCache.get(asset.path)}

            <button
              class="asset-tile"
              class:selected={selectedAsset?.path === asset.path}
              onclick={() => { selectedAsset = selectedAsset?.path === asset.path ? null : asset }}
              title={`${asset.filename}\n${formatSize(asset.sizeBytes)}${asset.dimensions ? `\n${asset.dimensions.width}x${asset.dimensions.height}` : ''}`}
            >
              <div class="tile-thumb">
                {#if thumb && thumb.startsWith('data:')}
                  <img src={thumb} alt={asset.filename} />
                {:else if thumb === 'loading'}
                  <Loader size={14} />
                {:else if isImageMime(asset.mimeType)}
                  <ImagePlus size={18} />
                {:else if asset.mimeType === 'application/pdf'}
                  <FileText size={18} />
                {:else if asset.mimeType.startsWith('video/')}
                  <Film size={18} />
                {:else}
                  <FileText size={18} />
                {/if}
              </div>
              <span class="tile-name">{asset.filename}</span>
            </button>
          {/each}
        </div>
      {/if}

      {#if (assets.scope === 'all' || assets.scope === 'shared') && sharedAssets.length > 0}
        <div class="group-label">Shared</div>
        <div class="asset-grid">
          {#each sharedAssets as asset (asset.path)}
            {@const thumb = thumbnailCache.get(asset.path)}

            <button
              class="asset-tile"
              class:selected={selectedAsset?.path === asset.path}
              onclick={() => { selectedAsset = selectedAsset?.path === asset.path ? null : asset }}
              title={`${asset.filename}\n${formatSize(asset.sizeBytes)}${asset.dimensions ? `\n${asset.dimensions.width}x${asset.dimensions.height}` : ''}`}
            >
              <div class="tile-thumb">
                {#if thumb && thumb.startsWith('data:')}
                  <img src={thumb} alt={asset.filename} />
                {:else if thumb === 'loading'}
                  <Loader size={14} />
                {:else if isImageMime(asset.mimeType)}
                  <ImagePlus size={14} />
                {:else if asset.mimeType === 'application/pdf'}
                  <FileText size={18} />
                {:else if asset.mimeType.startsWith('video/')}
                  <Film size={18} />
                {:else}
                  <FileText size={18} />
                {/if}
              </div>
              <span class="tile-name">{asset.filename}</span>
            </button>
          {/each}
        </div>
      {/if}
    </div>
  {/if}

  {#if selectedAsset}
    <div class="asset-actions">
      <div class="action-info">
        <span class="action-name">{selectedAsset.filename}</span>
        <span class="action-meta">{formatSize(selectedAsset.sizeBytes)}{selectedAsset.dimensions ? ` · ${selectedAsset.dimensions.width}×${selectedAsset.dimensions.height}` : ''}</span>
      </div>
      <div class="action-buttons">
        <button class="action-btn insert" onclick={() => insertAsset(selectedAsset)} title="Insert at cursor">
          <ImagePlus size={13} />
          Insert
        </button>
        <button class="action-btn delete" onclick={() => deleteAsset(selectedAsset)} title="Delete asset">
          <Trash2 size={13} />
        </button>
      </div>
    </div>
  {/if}
</div>

<style>
  .media-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    gap: 0;
  }

  :global(.scope-tabs) {
    display: flex;
    gap: 2px;
    padding: 6px 8px;
    border-bottom: 1px solid var(--cr-border);
  }

  :global(.scope-tab) {
    flex: 1;
    padding: 4px 0;
    border: none;
    border-radius: var(--cr-radius-sm);
    background: transparent;
    color: var(--cr-text-muted);
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.12s, color 0.12s;
  }

  :global(.scope-tab:hover) {
    background: var(--cr-hover);
    color: var(--cr-text);
  }

  :global(.scope-tab[data-state="active"]) {
    background: var(--cr-active);
    color: var(--cr-accent);
  }

  .upload-area {
    display: flex;
    gap: 4px;
    padding: 6px 8px;
    border-bottom: 1px solid var(--cr-border);
  }

  .upload-btn {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    padding: 5px 0;
    border: 1px dashed var(--cr-border);
    border-radius: var(--cr-radius-sm);
    background: transparent;
    color: var(--cr-text-muted);
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.12s, color 0.12s, border-color 0.12s;
  }

  .upload-btn:hover:not(:disabled) {
    background: var(--cr-hover);
    color: var(--cr-text);
    border-color: var(--cr-accent);
  }

  .upload-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .media-status {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 24px 12px;
    color: var(--cr-text-muted);
    font-size: 12px;
  }

  .asset-list {
    flex: 1;
    overflow-y: auto;
    padding: 4px 0;
  }

  .group-label {
    padding: 6px 10px 2px;
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--cr-text-muted);
  }

  .asset-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 4px;
    padding: 4px 8px;
  }

  .asset-tile {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 6px 4px;
    border: 1px solid transparent;
    border-radius: var(--cr-radius);
    background: transparent;
    cursor: pointer;
    transition: background 0.12s, border-color 0.12s;
    color: var(--cr-text-muted);
  }

  .asset-tile:hover {
    background: var(--cr-hover);
  }

  .asset-tile.selected {
    background: var(--cr-active);
    border-color: var(--cr-accent);
  }

  .tile-thumb {
    width: 80px;
    height: 60px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--cr-radius-sm);
    background: var(--cr-bg-input);
    overflow: hidden;
  }

  .tile-thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: var(--cr-radius-sm);
  }

  .tile-name {
    font-size: 10px;
    color: var(--cr-text);
    max-width: 90px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    text-align: center;
  }

  .asset-actions {
    border-top: 1px solid var(--cr-border);
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .action-info {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .action-name {
    font-size: 11px;
    font-weight: 600;
    color: var(--cr-text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .action-meta {
    font-size: 10px;
    color: var(--cr-text-muted);
  }

  .action-buttons {
    display: flex;
    gap: 4px;
  }

  .action-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    padding: 4px 10px;
    border: none;
    border-radius: var(--cr-radius-sm);
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.12s;
  }

  .action-btn.insert {
    flex: 1;
    background: var(--cr-accent);
    color: #000;
  }

  .action-btn.insert:hover {
    background: var(--cr-accent-hover);
  }

  .action-btn.delete {
    background: transparent;
    color: var(--cr-text-muted);
    border: 1px solid var(--cr-border);
  }

  .action-btn.delete:hover {
    color: var(--cr-danger);
    border-color: var(--cr-danger);
    background: rgba(243, 139, 168, 0.1);
  }
</style>
