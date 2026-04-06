<script>
  import { doc, sidecar } from '../stores/app.svelte.js'

  let saveLabel = $derived(doc.dirty ? 'Unsaved' : 'Saved')
  let saveClass = $derived(doc.dirty ? 'unsaved' : 'saved')
  let connectionLabel = $derived(sidecar.ready ? 'Connected' : 'Disconnected')
  let connectionClass = $derived(sidecar.ready ? 'connected' : 'disconnected')
</script>

<div class="status-bar">
  <div class="status-left">
    <span class="status-item {saveClass}">
      <span class="status-dot"></span>
      {saveLabel}
    </span>
    <span class="status-item {connectionClass}">
      <span class="status-dot"></span>
      {connectionLabel}
    </span>
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
    background: var(--bg-surface, #1e1e2e);
    border-top: 1px solid var(--border, #2e2e3e);
    font-size: 11px;
    color: var(--text-muted, #888);
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
    background: #22c55e;
  }

  .unsaved .status-dot {
    background: #f59e0b;
  }

  .connected .status-dot {
    background: #22c55e;
  }

  .disconnected .status-dot {
    background: #ef4444;
  }
</style>
