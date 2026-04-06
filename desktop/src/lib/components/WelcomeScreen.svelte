<script>
  import { FolderOpen, Plus, Clock, Trash2, AlertCircle } from 'lucide-svelte'

  let { onOpen, onCreate } = $props()

  let recentProjects = $state(loadRecents())

  function loadRecents() {
    try {
      return JSON.parse(localStorage.getItem('coderoo-recent-projects') || '[]')
    } catch { return [] }
  }

  function openRecent(project) {
    onOpen(project.path)
  }

  function removeRecent(e, idx) {
    e.stopPropagation()
    recentProjects.splice(idx, 1)
    localStorage.setItem('coderoo-recent-projects', JSON.stringify(recentProjects))
    recentProjects = [...recentProjects]
  }

  function formatDate(iso) {
    if (!iso) return ''
    const d = new Date(iso)
    const now = new Date()
    const diff = now - d
    if (diff < 86400000) return 'Today'
    if (diff < 172800000) return 'Yesterday'
    return d.toLocaleDateString()
  }
</script>

<div class="welcome">
  <div class="welcome-header">
    <div class="logo">C</div>
    <h1>Coderoo</h1>
    <p class="tagline">Build beautiful documentation sites</p>
  </div>

  <div class="actions">
    <button class="action-card" onclick={() => onOpen()}>
      <FolderOpen size={24} />
      <span class="action-label">Open Project</span>
      <span class="action-desc">Open an existing project folder</span>
    </button>
    <button class="action-card" onclick={onCreate}>
      <Plus size={24} />
      <span class="action-label">Create New</span>
      <span class="action-desc">Start a new site from scratch</span>
    </button>
  </div>

  {#if recentProjects.length > 0}
    <div class="recents">
      <h3 class="recents-title"><Clock size={14} /> Recent Projects</h3>
      <ul class="recents-list">
        {#each recentProjects as project, i}
          <li class="recent-row">
            <button class="recent-item" onclick={() => openRecent(project)}>
              <div class="recent-info">
                <span class="recent-name">{project.name}</span>
                <span class="recent-path">{project.path}</span>
              </div>
              <span class="recent-date">{formatDate(project.lastOpened)}</span>
            </button>
            <button class="recent-remove" title="Remove from list" onclick={(e) => removeRecent(e, i)}>
              <Trash2 size={13} />
            </button>
          </li>
        {/each}
      </ul>
    </div>
  {/if}
</div>

<style>
  .welcome {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 32px;
    padding: 40px;
  }

  .welcome-header {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
  }

  .logo {
    width: 72px;
    height: 72px;
    border-radius: 18px;
    background: var(--color-accent, #89b4fa);
    color: var(--color-surface, #1e1e2e);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 36px;
    font-weight: 800;
    margin-bottom: 4px;
  }

  h1 {
    margin: 0;
    font-size: 28px;
    font-weight: 700;
    color: var(--color-text, #cdd6f4);
  }

  .tagline {
    margin: 0;
    font-size: 14px;
    color: var(--color-text-muted, #6c7086);
  }

  .actions {
    display: flex;
    gap: 16px;
  }

  .action-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    width: 180px;
    padding: 24px 16px;
    border: 1px solid var(--color-border, #313244);
    border-radius: 12px;
    background: var(--color-surface-alt, #181825);
    color: var(--color-text, #cdd6f4);
    cursor: pointer;
    transition: border-color 0.15s, background 0.15s, transform 0.1s;
  }

  .action-card:hover {
    border-color: var(--color-accent, #89b4fa);
    background: rgba(137, 180, 250, 0.06);
    transform: translateY(-2px);
  }

  .action-card :global(svg) {
    color: var(--color-accent, #89b4fa);
  }

  .action-label {
    font-size: 15px;
    font-weight: 600;
  }

  .action-desc {
    font-size: 12px;
    color: var(--color-text-muted, #6c7086);
    text-align: center;
  }

  .recents {
    width: 100%;
    max-width: 480px;
  }

  .recents-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-text-muted, #6c7086);
    margin: 0 0 8px;
  }

  .recents-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .recent-row {
    display: flex;
    align-items: center;
    border-radius: 8px;
    transition: background 0.15s;
  }

  .recent-row:hover {
    background: var(--color-hover, rgba(255, 255, 255, 0.06));
  }

  .recent-item {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;
    min-width: 0;
    padding: 10px 12px;
    border: none;
    border-radius: 8px;
    background: transparent;
    color: var(--color-text, #cdd6f4);
    cursor: pointer;
    text-align: left;
  }

  .recent-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .recent-name {
    font-size: 14px;
    font-weight: 500;
  }

  .recent-path {
    font-size: 11px;
    color: var(--color-text-muted, #6c7086);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .recent-date {
    font-size: 11px;
    color: var(--color-text-muted, #6c7086);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .recent-remove {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--color-text-muted, #6c7086);
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s;
    flex-shrink: 0;
  }

  .recent-row:hover .recent-remove {
    opacity: 1;
  }

  .recent-remove:hover {
    color: var(--color-danger, #f38ba8);
    background: rgba(243, 139, 168, 0.1);
  }
</style>
