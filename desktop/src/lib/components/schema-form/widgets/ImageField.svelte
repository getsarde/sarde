<script>
  import FieldWrapper from './FieldWrapper.svelte'
  import { assetList, assetGetThumbnail } from '../../../api.js'
  import { Image, Upload, X } from 'lucide-svelte'

  let { name, label = name, value = '', required = false, error = '', disabled = false, onchange } = $props()

  let browsing = $state(false)
  let assets = $state([])
  let thumbnails = $state(new Map())
  let loading = $state(false)

  async function openBrowser() {
    if (disabled) return
    browsing = true
    loading = true
    try {
      const all = await assetList('all')
      assets = all.filter(a => a.mimeType?.startsWith('image/'))
      // Load thumbnails for visible assets
      for (const asset of assets.slice(0, 30)) {
        loadThumbnail(asset.path)
      }
    } catch {
      assets = []
    } finally {
      loading = false
    }
  }

  async function loadThumbnail(path) {
    if (thumbnails.has(path)) return
    thumbnails.set(path, 'loading')
    thumbnails = new Map(thumbnails)
    try {
      const thumb = await assetGetThumbnail(path)
      thumbnails.set(path, thumb)
    } catch {
      thumbnails.set(path, 'error')
    }
    thumbnails = new Map(thumbnails)
  }

  function selectAsset(asset) {
    // Compute relative path for insertion
    let imgPath
    if (asset.path.startsWith('static/')) {
      imgPath = '/' + asset.path.slice(7)
    } else {
      imgPath = asset.filename
    }
    onchange(imgPath)
    browsing = false
  }

  function clearValue() {
    onchange('')
  }
</script>

<FieldWrapper {name} {label} {required} {error}>
  <div class="image-field">
    <div class="input-row">
      <input
        class="field-input"
        id={name}
        type="text"
        {value}
        {disabled}
        placeholder="Image path..."
        onchange={(e) => onchange(e.target.value)}
      />
      {#if value}
        <button class="icon-btn" onclick={clearValue} title="Clear"><X size={13} /></button>
      {/if}
      <button class="icon-btn browse" onclick={openBrowser} title="Browse assets" {disabled}>
        <Image size={13} />
      </button>
    </div>

    {#if browsing}
      <div class="asset-browser">
        <div class="browser-header">
          <span class="browser-title">Select Image</span>
          <button class="icon-btn" onclick={() => browsing = false}><X size={13} /></button>
        </div>
        {#if loading}
          <div class="browser-loading">Loading assets...</div>
        {:else if assets.length === 0}
          <div class="browser-empty">No images found in project.</div>
        {:else}
          <div class="thumbnail-grid">
            {#each assets as asset}
              {@const thumb = thumbnails.get(asset.path)}
              <button
                class="thumb-item"
                class:selected={value === asset.filename || value === '/' + asset.path.slice(7)}
                onclick={() => selectAsset(asset)}
                title={asset.filename}
              >
                {#if thumb && thumb !== 'loading' && thumb !== 'error' && !thumb.startsWith('icon:')}
                  <img src={thumb} alt={asset.filename} />
                {:else}
                  <div class="thumb-placeholder"><Image size={20} /></div>
                {/if}
                <span class="thumb-name">{asset.filename}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>
</FieldWrapper>

<style>
  .image-field {
    position: relative;
  }

  .input-row {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .field-input {
    flex: 1;
    box-sizing: border-box;
    padding: 5px 8px;
    font-size: 12px;
    color: var(--cr-text);
    background: var(--cr-bg-input);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
    outline: none;
    font-family: inherit;
  }

  .field-input:focus {
    border-color: var(--cr-accent);
  }

  .icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
    background: var(--cr-bg-elevated);
    color: var(--cr-text-muted);
    cursor: pointer;
    flex-shrink: 0;
  }

  .icon-btn:hover:not(:disabled) {
    color: var(--cr-text);
    border-color: var(--cr-accent);
  }

  .icon-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .icon-btn.browse {
    background: var(--cr-accent);
    border-color: var(--cr-accent);
    color: #fff;
  }

  .icon-btn.browse:hover:not(:disabled) {
    filter: brightness(1.1);
    color: #fff;
    border-color: var(--cr-accent);
  }

  .asset-browser {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    z-index: 50;
    background: var(--cr-bg-base);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
    margin-top: 4px;
    max-height: 280px;
    display: flex;
    flex-direction: column;
  }

  .browser-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 10px;
    border-bottom: 1px solid var(--cr-border);
  }

  .browser-title {
    font-size: 11px;
    font-weight: 600;
    color: var(--cr-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .browser-loading,
  .browser-empty {
    padding: 20px;
    text-align: center;
    font-size: 12px;
    color: var(--cr-text-muted);
  }

  .thumbnail-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
    gap: 6px;
    padding: 8px;
    overflow-y: auto;
    flex: 1;
  }

  .thumb-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 6px;
    border: 1px solid transparent;
    border-radius: var(--cr-radius-sm);
    background: transparent;
    cursor: pointer;
    transition: border-color 0.12s, background 0.12s;
  }

  .thumb-item:hover {
    background: var(--cr-bg-elevated);
    border-color: var(--cr-border);
  }

  .thumb-item.selected {
    border-color: var(--cr-accent);
    background: var(--cr-active);
  }

  .thumb-item img {
    width: 56px;
    height: 56px;
    object-fit: cover;
    border-radius: 4px;
  }

  .thumb-placeholder {
    width: 56px;
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--cr-bg-surface);
    border-radius: 4px;
    color: var(--cr-text-muted);
  }

  .thumb-name {
    font-size: 10px;
    color: var(--cr-text-muted);
    max-width: 72px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: center;
  }
</style>
