<script>
  import { open } from '@tauri-apps/plugin-dialog'
  import { ui, project } from '../stores/app.svelte.js'
  import FileTree from './FileTree.svelte'
  import SearchPanel from './SearchPanel.svelte'
  import { Folder, Search, GitBranch, Settings, FilePlus, FolderPlus, FolderOpen } from 'lucide-svelte'

  let fileTreeRef = $state(null)

  function togglePanel(panel) {
    ui.leftPanel = ui.leftPanel === panel ? null : panel
  }

  async function openFolder() {
    const selected = await open({ directory: true, multiple: false, title: 'Open Project Folder' })
    if (selected) {
      project.contentPath = selected
    }
  }
</script>

<aside class="left-sidebar">
  <div class="icon-strip">
    <button class="icon-btn" class:active={ui.leftPanel === 'files'} onclick={() => togglePanel('files')} title="Explorer">
      <Folder size={20} />
    </button>
    <button class="icon-btn" class:active={ui.leftPanel === 'search'} onclick={() => togglePanel('search')} title="Search">
      <Search size={20} />
    </button>
    <button class="icon-btn" class:active={ui.leftPanel === 'git'} onclick={() => togglePanel('git')} title="Source Control">
      <GitBranch size={20} />
    </button>

    <div class="icon-spacer"></div>

    <button class="icon-btn" onclick={() => (ui.settingsOpen = true)} title="Settings">
      <Settings size={20} />
    </button>
  </div>

  {#if ui.leftPanel}
    <div class="sidebar-panel">
      {#if ui.leftPanel === 'files'}
        <div class="panel-header">
          <span class="panel-title">Explorer</span>
          <div class="panel-actions">
            <button class="panel-action" title="New File" onclick={() => fileTreeRef?.newFileAtRoot()}>
              <FilePlus size={15} />
            </button>
            <button class="panel-action" title="New Folder" onclick={() => fileTreeRef?.newFolderAtRoot()}>
              <FolderPlus size={15} />
            </button>
            <button class="panel-action" title="Open Folder" onclick={openFolder}>
              <FolderOpen size={15} />
            </button>
          </div>
        </div>
        <FileTree bind:this={fileTreeRef} />

      {:else if ui.leftPanel === 'search'}
        <div class="panel-header">
          <span class="panel-title">Search</span>
        </div>
        <SearchPanel />

      {:else if ui.leftPanel === 'git'}
        <div class="panel-header">
          <span class="panel-title">Source Control</span>
        </div>
        <div class="git-placeholder">
          <p>No changes detected.</p>
        </div>
      {/if}
    </div>
  {/if}
</aside>

<style>
  .left-sidebar {
    display: flex;
    height: 100%;
    background: var(--color-surface, #1e1e2e);
    border-right: 1px solid var(--color-border, #313244);
  }

  .icon-strip {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 44px;
    padding: 6px 0;
    gap: 2px;
    background: var(--color-surface-alt, #181825);
    border-right: 1px solid var(--color-border, #313244);
  }

  .icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--color-text-muted, #6c7086);
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
  }

  .icon-btn:hover {
    color: var(--color-text, #cdd6f4);
    background: var(--color-hover, rgba(255, 255, 255, 0.06));
  }

  .icon-btn.active {
    color: var(--color-accent, #89b4fa);
    background: var(--color-active, rgba(137, 180, 250, 0.1));
  }

  .icon-spacer {
    flex: 1;
  }

  .sidebar-panel {
    width: 240px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-text-muted, #6c7086);
    border-bottom: 1px solid var(--color-border, #313244);
  }

  .panel-title {
    user-select: none;
  }

  .panel-actions {
    display: flex;
    gap: 2px;
  }

  .panel-action {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--color-text-muted, #6c7086);
    cursor: pointer;
  }

  .panel-action:hover {
    color: var(--color-text, #cdd6f4);
    background: var(--color-hover, rgba(255, 255, 255, 0.06));
  }

  .git-placeholder {
    padding: 12px;
    font-size: 12px;
    color: var(--color-text-muted, #6c7086);
    overflow-y: auto;
    flex: 1;
  }

  .git-placeholder p {
    margin: 0;
  }
</style>
