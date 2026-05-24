<script>
  import { ui } from '../stores/app.svelte.js'
  import { FilePlus, Search } from 'lucide-svelte'

  function newFile() {
    window.dispatchEvent(new CustomEvent('sarde:new-file'))
  }

  function openSearch() {
    ui.leftPanel = 'search'
  }
</script>

<div class="empty-editor">
  <div class="empty-content">
    <h2>No file open</h2>
    <p>Open a file from the Explorer or create a new one to get started.</p>

    <div class="shortcuts">
      <button class="shortcut-card" onclick={newFile}>
        <FilePlus size={20} />
        <span>New File</span>
      </button>
      <button class="shortcut-card" onclick={openSearch}>
        <Search size={20} />
        <span>Search Files</span>
      </button>
    </div>

    <div class="keyhints">
      <div class="keyhint"><kbd>Ctrl+N</kbd> New file</div>
      <div class="keyhint"><kbd>Ctrl+Shift+P</kbd> Command palette</div>
      <div class="keyhint"><kbd>Ctrl+Shift+F</kbd> Search in files</div>
      <div class="keyhint"><kbd>Ctrl+\</kbd> Toggle sidebar</div>
      <div class="keyhint"><kbd>Ctrl+,</kbd> Settings</div>
    </div>
  </div>
</div>

<style>
  .empty-editor {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 40px;
  }

  .empty-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    max-width: 400px;
    text-align: center;
  }

  h2 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: var(--cr-text);
  }

  p {
    margin: 0;
    font-size: 13px;
    color: var(--cr-text-muted);
    line-height: 1.5;
  }

  .shortcuts {
    display: flex;
    gap: 12px;
    margin-top: 8px;
  }

  .shortcut-card {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 18px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: var(--cr-bg-input);
    color: var(--cr-text);
    font-size: 13px;
    font-family: inherit;
    cursor: pointer;
    transition: border-color 0.15s;
  }

  .shortcut-card:hover {
    border-color: var(--cr-accent);
  }

  .shortcut-card :global(svg) {
    color: var(--cr-accent);
  }

  .keyhints {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 8px 16px;
    margin-top: 16px;
  }

  .keyhint {
    font-size: 12px;
    color: var(--cr-text-muted);
    display: flex;
    align-items: center;
    gap: 6px;
  }

  kbd {
    padding: 2px 6px;
    border-radius: var(--cr-radius-sm);
    font-size: 11px;
    font-family: inherit;
    background: var(--cr-bg-input);
    border: 1px solid var(--cr-border);
    color: var(--cr-text-muted);
  }
</style>
