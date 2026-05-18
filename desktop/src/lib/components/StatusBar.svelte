<script>
  import { doc, ui, warnings, preview, buildLog, toggleBuildLog } from '../stores/app.svelte.js'

  let saveLabel = $derived(doc.dirty ? 'Unsaved' : 'Saved')
  let saveClass = $derived(doc.dirty ? 'unsaved' : 'saved')
  let previewLabel = $derived(preview.running ? `Preview :${preview.port}` : 'Preview off')
  let previewClass = $derived(preview.running ? 'connected' : 'disconnected')
</script>

<div class="status-bar">
  <div class="status-left">
    <span class="status-item {saveClass}">
      <span class="status-dot"></span>
      {saveLabel}
    </span>
    <button class="status-item status-btn {previewClass}" onclick={toggleBuildLog} title="Toggle build log">
      <span class="status-dot"></span>
      {previewLabel}
    </button>
    {#if warnings.items.length > 0}
      <button class="status-item status-btn warnings-indicator" onclick={() => { ui.rightPanel = 'warnings' }}>
        <span class="status-dot"></span>
        {warnings.items.length} {warnings.items.length === 1 ? 'warning' : 'warnings'}
      </button>
    {/if}
  </div>

  <div class="status-right">
    <span class="status-item">
      Ln {doc.cursorLine}, Col {doc.cursorCol}
    </span>
    <span class="status-item">
      {doc.wordCount} {doc.wordCount === 1 ? 'word' : 'words'}
    </span>
    <span class="status-item">
      {doc.readingTime} min read
    </span>
  </div>
</div>

<style>
  .status-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 24px;
    padding: 0 12px;
    background: var(--cr-bg-surface);
    border-top: 1px solid var(--cr-border);
    font-size: 11px;
    color: var(--cr-text-muted);
    user-select: none;
    flex-shrink: 0;
  }

  .status-left,
  .status-right {
    display: flex;
    align-items: center;
    gap: 14px;
  }

  .status-item {
    display: flex;
    align-items: center;
    gap: 5px;
    white-space: nowrap;
  }

  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .saved .status-dot {
    background: var(--cr-success);
  }

  .unsaved .status-dot {
    background: var(--cr-warning);
  }

  .connected .status-dot {
    background: var(--cr-success);
  }

  .disconnected .status-dot {
    background: var(--cr-danger);
  }

  .status-btn {
    background: none;
    border: none;
    padding: 0;
    font: inherit;
    color: inherit;
    cursor: pointer;
  }

  .status-btn:hover {
    color: var(--cr-text);
  }

  .warnings-indicator .status-dot {
    background: var(--cr-warning);
  }
</style>
