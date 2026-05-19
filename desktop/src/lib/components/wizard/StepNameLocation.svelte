<script>
  import { open as openDialog } from '@tauri-apps/plugin-dialog'
  import { FolderOpen } from 'lucide-svelte'

  let { projectName = '', location = '', onNameChange, onLocationChange } = $props()

  async function pickLocation() {
    const selected = await openDialog({
      directory: true,
      title: 'Choose where to create the project',
    })
    if (selected) onLocationChange(selected.replace(/\\/g, '/'))
  }
</script>

<div class="field">
  <label class="field-label" for="proj-name">Project Name</label>
  <input
    id="proj-name"
    type="text"
    class="field-input"
    placeholder="my-awesome-site"
    value={projectName}
    oninput={(e) => onNameChange(e.target.value)}
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

{#if location && projectName.trim()}
  <p class="path-preview">Project will be created at: <code title="{location}/{projectName.trim()}">{location}/{projectName.trim()}</code></p>
{/if}

<style>
  .field { margin-bottom: 18px; }

  .field-label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: var(--cr-text-muted);
    margin-bottom: 6px;
  }

  .field-input {
    width: 100%;
    padding: 10px 12px;
    font-size: 14px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: var(--cr-bg-input);
    color: var(--cr-text);
    outline: none;
    box-sizing: border-box;
    font-family: inherit;
  }

  .field-input:focus { border-color: var(--cr-accent); }

  .location-row { display: flex; gap: 8px; }
  .location-input { flex: 1; cursor: default; }

  .browse-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 14px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: var(--cr-bg-base);
    color: var(--cr-text);
    font-size: 13px;
    font-family: inherit;
    cursor: pointer;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .browse-btn:hover {
    background: var(--cr-hover);
    border-color: var(--cr-accent);
  }

  .path-preview {
    font-size: 12px;
    color: var(--cr-text-muted);
    margin: -4px 0 0;
  }

  .path-preview code {
    color: var(--cr-accent);
    font-family: var(--cr-font-mono);
    font-size: 11px;
    display: block;
    margin-top: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
