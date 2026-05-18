<script>
  import { open } from '@tauri-apps/plugin-dialog'
  import { ui } from '../stores/app.svelte.js'
  import FileTree from './FileTree.svelte'
  import SearchPanel from './SearchPanel.svelte'
  import TaxonomyPanel from './TaxonomyPanel.svelte'
  import AppTooltip from './primitives/AppTooltip.svelte'
  import { Folder, Search, GitBranch, Tags, Settings, FilePlus, FolderPlus, FolderOpen } from 'lucide-svelte'

  let fileTreeRef = $state(null)

  function togglePanel(panel) {
    ui.leftPanel = ui.leftPanel === panel ? null : panel
  }

  async function openFolder() {
    const selected = await open({ directory: true, multiple: false, title: 'Open Project Folder' })
    if (selected) {
      window.dispatchEvent(new CustomEvent('coderoo:open-project', { detail: selected }))
    }
  }
</script>

<aside class="left-sidebar">
  <div class="icon-strip" role="toolbar" aria-label="Sidebar navigation">
    <AppTooltip content="Explorer">
      <button class="icon-btn" class:active={ui.leftPanel === 'files'} onclick={() => togglePanel('files')}>
        <Folder size={20} />
      </button>
    </AppTooltip>
    <AppTooltip content="Search">
      <button class="icon-btn" class:active={ui.leftPanel === 'search'} onclick={() => togglePanel('search')}>
        <Search size={20} />
      </button>
    </AppTooltip>
    <AppTooltip content="Source Control">
      <button class="icon-btn" class:active={ui.leftPanel === 'git'} onclick={() => togglePanel('git')}>
        <GitBranch size={20} />
      </button>
    </AppTooltip>
    <AppTooltip content="Tags & Categories">
      <button class="icon-btn" class:active={ui.leftPanel === 'taxonomy'} onclick={() => togglePanel('taxonomy')}>
        <Tags size={20} />
      </button>
    </AppTooltip>

    <div class="icon-spacer"></div>

    <AppTooltip content="Settings">
      <button class="icon-btn" onclick={() => (ui.settingsOpen = true)}>
        <Settings size={20} />
      </button>
    </AppTooltip>
  </div>

  {#if ui.leftPanel}
    <div class="sidebar-panel">
      {#if ui.leftPanel === 'files'}
        <div class="panel-header">
          <span class="panel-title">Explorer</span>
          <div class="panel-actions">
            <AppTooltip content="New File">
              <button class="panel-action" onclick={() => fileTreeRef?.newFileAtRoot()}>
                <FilePlus size={15} />
              </button>
            </AppTooltip>
            <AppTooltip content="New Folder">
              <button class="panel-action" onclick={() => fileTreeRef?.newFolderAtRoot()}>
                <FolderPlus size={15} />
              </button>
            </AppTooltip>
            <AppTooltip content="Open Folder">
              <button class="panel-action" onclick={openFolder}>
                <FolderOpen size={15} />
              </button>
            </AppTooltip>
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
          <p>Git integration coming soon.</p>
        </div>

      {:else if ui.leftPanel === 'taxonomy'}
        <TaxonomyPanel />
      {/if}
    </div>
  {/if}
</aside>

<style>
  .left-sidebar {
    display: flex;
    height: 100%;
    background: var(--cr-bg-base);
    border-right: 1px solid var(--cr-border);
  }

  .icon-strip {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 44px;
    padding: 6px 0;
    gap: 2px;
    background: var(--cr-bg-input);
    border-right: 1px solid var(--cr-border);
  }

  .icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border: none;
    border-radius: var(--cr-radius);
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
  }

  .icon-btn:hover {
    color: var(--cr-text);
    background: var(--cr-hover);
  }

  .icon-btn.active {
    color: var(--cr-accent);
    background: var(--cr-active);
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
    color: var(--cr-text-muted);
    border-bottom: 1px solid var(--cr-border);
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
    border-radius: var(--cr-radius-sm);
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
  }

  .panel-action:hover {
    color: var(--cr-text);
    background: var(--cr-hover);
  }

  .git-placeholder {
    padding: 12px;
    font-size: 12px;
    color: var(--cr-text-muted);
    overflow-y: auto;
    flex: 1;
  }

  .git-placeholder p {
    margin: 0;
  }
</style>
