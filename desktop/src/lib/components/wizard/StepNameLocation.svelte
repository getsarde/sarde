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
  <p class="path-preview">Project will be created at: <code>{location}/{projectName.trim()}</code></p>
{/if}

<style>
  .field { margin-bottom: 18px; }

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

  .field-input:focus { border-color: var(--color-accent, #89b4fa); }

  .location-row { display: flex; gap: 8px; }
  .location-input { flex: 1; cursor: default; }

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
    margin: -8px 0 0;
  }

  .path-preview code {
    color: var(--color-accent, #89b4fa);
    font-size: 11px;
    word-break: break-all;
  }
</style>
