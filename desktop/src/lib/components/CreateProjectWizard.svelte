<script>
  import { open as openDialog } from '@tauri-apps/plugin-dialog'
  import { ArrowLeft, FolderOpen, Rocket } from 'lucide-svelte'
  import { projectCreate } from '../api.js'

  let { onCreated, onBack } = $props()

  let projectName = $state('')
  let location = $state('')
  let creating = $state(false)
  let error = $state('')

  async function pickLocation() {
    const selected = await openDialog({
      directory: true,
      title: 'Choose where to create the project',
    })
    if (selected) location = selected.replace(/\\/g, '/')
  }

  let fullPath = $derived(
    location && projectName.trim()
      ? `${location}/${projectName.trim()}`
      : ''
  )

  let canCreate = $derived(projectName.trim().length > 0 && location.length > 0 && !creating)

  async function create() {
    if (!canCreate) return
    creating = true
    error = ''
    const name = projectName.trim()
    const root = `${location}/${name}`

    try {
      await projectCreate(root, name)
      onCreated(root)
    } catch (e) {
      error = String(e)
    } finally {
      creating = false
    }
  }

  function onKeydown(e) {
    if (e.key === 'Enter' && canCreate) create()
    if (e.key === 'Escape') onBack()
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="wizard">
  <button class="back-link" onclick={onBack}>
    <ArrowLeft size={16} /> Back
  </button>

  <div class="wizard-card">
    <h2>Create New Project</h2>

    <div class="field">
      <label class="field-label" for="proj-name">Project Name</label>
      <input
        id="proj-name"
        type="text"
        class="field-input"
        placeholder="my-awesome-site"
        bind:value={projectName}
      />
    </div>

    <div class="field">
      <label class="field-label" for="proj-location">Location</label>
      <div class="location-row">
        <input
          id="proj-location"
          type="text"
          class="field-input location-input"
          placeholder="Choose a folder..."
          value={location}
          readonly
        />
        <button class="browse-btn" onclick={pickLocation}>
          <FolderOpen size={15} /> Browse
        </button>
      </div>
    </div>

    {#if fullPath}
      <p class="path-preview">Project will be created at: <code>{fullPath}</code></p>
    {/if}

    {#if error}
      <p class="error-msg">{error}</p>
    {/if}

    <button class="create-btn" onclick={create} disabled={!canCreate}>
      {#if creating}
        Creating...
      {:else}
        <Rocket size={16} /> Create Project
      {/if}
    </button>
  </div>
</div>

<style>
  .wizard {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 20px;
    padding: 40px;
  }

  .back-link {
    position: absolute;
    top: 20px;
    left: 20px;
    display: flex;
    align-items: center;
    gap: 6px;
    border: none;
    background: transparent;
    color: var(--color-text-muted, #6c7086);
    font-size: 13px;
    cursor: pointer;
    padding: 6px 10px;
    border-radius: 6px;
  }

  .back-link:hover {
    color: var(--color-text, #cdd6f4);
    background: var(--color-hover, rgba(255, 255, 255, 0.06));
  }

  .wizard-card {
    width: 420px;
    max-width: 90vw;
    padding: 32px;
    border: 1px solid var(--color-border, #313244);
    border-radius: 14px;
    background: var(--color-surface-alt, #181825);
  }

  h2 {
    margin: 0 0 24px;
    font-size: 20px;
    font-weight: 700;
    color: var(--color-text, #cdd6f4);
  }

  .field {
    margin-bottom: 18px;
  }

  .field-label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: var(--color-text-muted, #6c7086);
    margin-bottom: 6px;
  }

  .field-input {
    width: 100%;
    padding: 10px 12px;
    font-size: 14px;
    border: 1px solid var(--color-border, #313244);
    border-radius: 8px;
    background: var(--color-input, #11111b);
    color: var(--color-text, #cdd6f4);
    outline: none;
    box-sizing: border-box;
    font-family: inherit;
  }

  .field-input:focus {
    border-color: var(--color-accent, #89b4fa);
  }

  .location-row {
    display: flex;
    gap: 8px;
  }

  .location-input {
    flex: 1;
    cursor: default;
  }

  .browse-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 14px;
    border: 1px solid var(--color-border, #313244);
    border-radius: 8px;
    background: var(--color-surface, #1e1e2e);
    color: var(--color-text, #cdd6f4);
    font-size: 13px;
    font-family: inherit;
    cursor: pointer;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .browse-btn:hover {
    background: var(--color-hover, rgba(255, 255, 255, 0.06));
    border-color: var(--color-accent, #89b4fa);
  }

  .path-preview {
    font-size: 12px;
    color: var(--color-text-muted, #6c7086);
    margin: -8px 0 16px;
  }

  .path-preview code {
    color: var(--color-accent, #89b4fa);
    font-size: 11px;
    word-break: break-all;
  }

  .error-msg {
    font-size: 13px;
    color: var(--color-danger, #f38ba8);
    margin: -4px 0 12px;
    padding: 8px 10px;
    background: rgba(243, 139, 168, 0.1);
    border-radius: 6px;
  }

  .create-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    width: 100%;
    padding: 12px;
    border: none;
    border-radius: 10px;
    background: var(--color-accent, #89b4fa);
    color: var(--color-surface, #1e1e2e);
    font-size: 15px;
    font-weight: 700;
    font-family: inherit;
    cursor: pointer;
    transition: background 0.15s, transform 0.1s;
  }

  .create-btn:hover:not(:disabled) {
    background: #74c7ec;
    transform: translateY(-1px);
  }

  .create-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
