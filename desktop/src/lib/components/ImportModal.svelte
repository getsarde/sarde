<script>
  import { open as openDialog } from '@tauri-apps/plugin-dialog'
  import { ui, addToast } from '../stores/app.svelte.js'
  import { importObsidian } from '../api.js'
  import { X, Loader, FolderOpen, CheckCircle, AlertCircle, FileDown } from 'lucide-svelte'

  let status = $state('idle') // 'idle' | 'importing' | 'success' | 'error'
  let vaultPath = $state('')
  let collection = $state('')
  let result = $state(null)
  let errorMsg = $state('')

  let vaultName = $derived(vaultPath ? vaultPath.split('/').pop() : '')
  let canImport = $derived(!!vaultPath)

  // Reset form when modal opens
  $effect(() => {
    if (ui.importOpen) {
      status = 'idle'
      vaultPath = ''
      collection = ''
      result = null
      errorMsg = ''
    }
  })

  function close() {
    ui.importOpen = false
  }

  function onKeydown(e) {
    if (e.key === 'Escape') close()
  }

  async function pickVault() {
    const selected = await openDialog({
      directory: true,
      title: 'Select Obsidian Vault',
    })
    if (selected) vaultPath = selected.replace(/\\/g, '/')
  }

  async function doImport() {
    status = 'importing'
    errorMsg = ''
    result = null
    try {
      const resp = await importObsidian(vaultPath, collection)
      result = resp ?? {}
      status = 'success'
      addToast('success', `Imported ${result.notesConverted ?? 0} notes from ${vaultName}`)
    } catch (e) {
      status = 'error'
      errorMsg = e.message || 'Import failed'
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="modal-overlay" onclick={(e) => { if (e.target === e.currentTarget) close() }} onkeydown={onKeydown} role="dialog" aria-modal="true" aria-label="Import Obsidian Vault" tabindex="-1">
  <div class="modal-content">
    <div class="modal-header">
      <h2>Import Obsidian Vault</h2>
      <button class="modal-close" onclick={close} title="Close">
        <X size={16} />
      </button>
    </div>

    <div class="modal-body">
      {#if status === 'idle'}
        <div class="field">
          <label class="field-label" for="vault-path">Vault Path</label>
          <div class="location-row">
            <input
              id="vault-path"
              type="text"
              class="field-input"
              placeholder="Select an Obsidian vault folder..."
              value={vaultPath}
              readonly
            />
            <button class="browse-btn" onclick={pickVault}>
              <FolderOpen size={15} /> Browse
            </button>
          </div>
        </div>

        <div class="field">
          <label class="field-label" for="import-collection">Collection Name</label>
          <input
            id="import-collection"
            type="text"
            class="field-input"
            placeholder={vaultName || 'Optional — defaults to vault name'}
            bind:value={collection}
          />
          <span class="field-hint">Leave blank to use the vault folder name.</span>
        </div>

        <button class="btn btn-primary" onclick={doImport} disabled={!canImport}>
          <FileDown size={15} /> Import
        </button>

      {:else if status === 'importing'}
        <div class="status-state">
          <div class="status-icon spinning">
            <Loader size={32} />
          </div>
          <p class="status-message">Importing from {vaultName}...</p>
        </div>

      {:else if status === 'success'}
        <div class="status-state">
          <div class="status-icon success">
            <CheckCircle size={32} />
          </div>
          <p class="status-message">Import complete</p>
          <div class="stats-grid">
            <div class="stat">
              <span class="stat-value">{result.notesConverted ?? 0}</span>
              <span class="stat-label">Notes</span>
            </div>
            <div class="stat">
              <span class="stat-value">{result.imagesCopied ?? 0}</span>
              <span class="stat-label">Images</span>
            </div>
            <div class="stat">
              <span class="stat-value">{result.linksConverted ?? 0}</span>
              <span class="stat-label">Links</span>
            </div>
            <div class="stat">
              <span class="stat-value">{result.itemsSkipped ?? 0}</span>
              <span class="stat-label">Skipped</span>
            </div>
          </div>
          <button class="btn btn-secondary" onclick={close}>Close</button>
        </div>

      {:else if status === 'error'}
        <div class="status-state">
          <div class="status-icon error">
            <AlertCircle size={32} />
          </div>
          <p class="status-message error-text">{errorMsg}</p>
          <div class="btn-group">
            <button class="btn btn-primary" onclick={() => { status = 'idle' }}>Try Again</button>
            <button class="btn btn-secondary" onclick={close}>Close</button>
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
  }

  .modal-content {
    width: 420px;
    max-width: 90vw;
    display: flex;
    flex-direction: column;
    background: var(--cr-bg-base);
    border: 1px solid var(--cr-border);
    border-radius: 12px;
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.4);
    overflow: hidden;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid var(--cr-border);
  }

  .modal-header h2 {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    color: var(--cr-text);
  }

  .modal-close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    border: none;
    border-radius: var(--cr-radius);
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
  }

  .modal-close:hover {
    background: var(--cr-hover);
    color: var(--cr-text);
  }

  .modal-body {
    padding: 20px;
  }

  .field {
    margin-bottom: 14px;
  }

  .field-label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: var(--cr-text-muted);
    margin-bottom: 6px;
  }

  .field-input {
    width: 100%;
    padding: 8px 10px;
    font-size: 13px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: var(--cr-bg-input);
    color: var(--cr-text);
    box-sizing: border-box;
  }

  .field-input:focus {
    outline: none;
    border-color: var(--cr-accent);
  }

  .field-hint {
    display: block;
    margin-top: 4px;
    font-size: 11px;
    color: var(--cr-text-muted);
  }

  .location-row {
    display: flex;
    gap: 8px;
  }

  .location-row .field-input {
    flex: 1;
    min-width: 0;
  }

  .browse-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 12px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: var(--cr-bg-base);
    color: var(--cr-text);
    font-size: 13px;
    cursor: pointer;
    white-space: nowrap;
  }

  .browse-btn:hover {
    background: var(--cr-hover);
    border-color: var(--cr-accent);
  }

  .btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    width: 100%;
    padding: 10px 20px;
    border: none;
    border-radius: var(--cr-radius);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .btn:hover {
    opacity: 0.9;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    background: var(--cr-accent);
    color: #fff;
  }

  .btn-secondary {
    background: var(--cr-hover);
    color: var(--cr-text);
  }

  .btn-group {
    display: flex;
    gap: 8px;
    width: 100%;
  }

  .btn-group .btn {
    flex: 1;
  }

  .status-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    padding: 12px 0;
    text-align: center;
  }

  .status-icon {
    color: var(--cr-accent);
  }

  .status-icon.success {
    color: #22c55e;
  }

  .status-icon.error {
    color: var(--cr-danger);
  }

  .status-icon.spinning {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .status-message {
    margin: 0;
    font-size: 14px;
    color: var(--cr-text-muted);
  }

  .error-text {
    color: var(--cr-danger);
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 12px;
    width: 100%;
  }

  .stat {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
  }

  .stat-value {
    font-size: 20px;
    font-weight: 600;
    color: var(--cr-text);
  }

  .stat-label {
    font-size: 11px;
    color: var(--cr-text-muted);
  }
</style>
