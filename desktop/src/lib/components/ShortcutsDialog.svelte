<script>
  import { ui } from '../stores/app.svelte.js'
  import AppDialog from './primitives/AppDialog.svelte'
  import { X } from 'lucide-svelte'

  const SHORTCUTS = [
    { section: 'General', items: [
      { keys: 'Ctrl+P', desc: 'Command Palette' },
      { keys: 'Ctrl+S', desc: 'Save file' },
      { keys: 'Ctrl+,', desc: 'Open Settings' },
      { keys: 'Ctrl+/', desc: 'Show Shortcuts' },
      { keys: 'Ctrl+\\', desc: 'Toggle left sidebar' },
    ]},
    { section: 'Editor', items: [
      { keys: 'Ctrl+F', desc: 'Find in file' },
      { keys: 'Ctrl+H', desc: 'Find & Replace in file' },
      { keys: 'Ctrl+B', desc: 'Bold' },
      { keys: 'Ctrl+I', desc: 'Italic' },
      { keys: 'Ctrl+K', desc: 'Insert link' },
      { keys: 'Ctrl+Z', desc: 'Undo' },
      { keys: 'Ctrl+Shift+Z', desc: 'Redo' },
    ]},
    { section: 'Tabs', items: [
      { keys: 'Ctrl+W', desc: 'Close tab' },
      { keys: 'Ctrl+Tab', desc: 'Next tab' },
      { keys: 'Ctrl+Shift+Tab', desc: 'Previous tab' },
    ]},
    { section: 'View', items: [
      { keys: 'Ctrl+Shift+M', desc: 'Cycle preview mode' },
      { keys: 'Ctrl+Shift+V', desc: 'Open in browser' },
      { keys: 'Ctrl+Shift+B', desc: 'Build site' },
      { keys: 'Ctrl+Shift+F', desc: 'Search in files' },
    ]},
    { section: 'Files', items: [
      { keys: 'Ctrl+Click', desc: 'Multi-select files' },
      { keys: 'Shift+Click', desc: 'Range select files' },
      { keys: 'Escape', desc: 'Clear selection' },
    ]},
  ]

  function close() {
    ui.shortcutsOpen = false
  }
</script>

<AppDialog open={ui.shortcutsOpen} onOpenChange={(v) => { if (!v) close() }} ariaLabel="Keyboard Shortcuts" width="520px">
  <div class="shortcuts-header">
    <h2>Keyboard Shortcuts</h2>
    <button class="close-btn" onclick={close} aria-label="Close"><X size={16} /></button>
  </div>
  <div class="shortcuts-body">
    {#each SHORTCUTS as group}
      <div class="shortcut-group">
        <h3 class="group-title">{group.section}</h3>
        {#each group.items as item}
          <div class="shortcut-row">
            <span class="shortcut-desc">{item.desc}</span>
            <kbd class="shortcut-keys">{item.keys}</kbd>
          </div>
        {/each}
      </div>
    {/each}
  </div>
</AppDialog>

<style>
  .shortcuts-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid var(--cr-border);
  }

  .shortcuts-header h2 {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    color: var(--cr-text);
  }

  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border: none;
    border-radius: var(--cr-radius-sm);
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
  }

  .close-btn:hover {
    color: var(--cr-text);
    background: var(--cr-hover);
  }

  .shortcuts-body {
    padding: 16px 20px;
    overflow-y: auto;
    max-height: 60vh;
  }

  .shortcut-group {
    margin-bottom: 16px;
  }

  .shortcut-group:last-child {
    margin-bottom: 0;
  }

  .group-title {
    margin: 0 0 8px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--cr-text-muted);
  }

  .shortcut-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 5px 0;
  }

  .shortcut-desc {
    font-size: 13px;
    color: var(--cr-text);
  }

  .shortcut-keys {
    font-family: var(--cr-font-mono);
    font-size: 11px;
    padding: 3px 8px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
    background: var(--cr-bg-input);
    color: var(--cr-text-muted);
    white-space: nowrap;
  }
</style>
