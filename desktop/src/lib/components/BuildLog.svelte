<script>
  import { buildLog, clearBuildLog, toggleBuildLog } from '../stores/app.svelte.js'
  import { X, Trash2 } from 'lucide-svelte'

  let listEl = $state(null)

  // Auto-scroll to bottom when new entries arrive
  $effect(() => {
    if (listEl && buildLog.entries.length > 0) {
      listEl.scrollTop = listEl.scrollHeight
    }
  })

  function formatTime(ts) {
    const d = new Date(ts)
    return d.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })
  }
</script>

{#if buildLog.visible}
  <div class="build-log">
    <div class="log-header">
      <span class="log-title">Build Output</span>
      <div class="log-actions">
        <button class="log-action" onclick={clearBuildLog} title="Clear log"><Trash2 size={13} /></button>
        <button class="log-action" onclick={toggleBuildLog} title="Close"><X size={14} /></button>
      </div>
    </div>
    <div class="log-list" bind:this={listEl}>
      {#if buildLog.entries.length === 0}
        <div class="log-empty">No build output yet. Start the preview server to see logs here.</div>
      {:else}
        {#each buildLog.entries as entry}
          <div class="log-entry {entry.type}">
            <span class="log-time">{formatTime(entry.timestamp)}</span>
            <span class="log-text">{entry.text}</span>
          </div>
        {/each}
      {/if}
    </div>
  </div>
{/if}

<style>
  .build-log {
    display: flex;
    flex-direction: column;
    height: 160px;
    border-top: 1px solid var(--border, #2e2e3e);
    background: var(--bg-surface, #1e1e2e);
    flex-shrink: 0;
  }

  .log-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 4px 10px;
    border-bottom: 1px solid var(--border, #2e2e3e);
    flex-shrink: 0;
  }

  .log-title {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-muted, #888);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .log-actions {
    display: flex;
    gap: 4px;
  }

  .log-action {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--text-muted, #888);
    cursor: pointer;
  }

  .log-action:hover {
    background: var(--bg-elevated, #2a2a3a);
    color: var(--text-primary, #e0e0e0);
  }

  .log-list {
    flex: 1;
    overflow-y: auto;
    padding: 4px 0;
    font-family: var(--font-mono, monospace);
    font-size: 12px;
    line-height: 1.6;
  }

  .log-empty {
    padding: 16px;
    text-align: center;
    font-size: 12px;
    color: var(--text-muted, #555);
    font-family: inherit;
  }

  .log-entry {
    display: flex;
    gap: 8px;
    padding: 1px 10px;
  }

  .log-entry:hover {
    background: rgba(255, 255, 255, 0.02);
  }

  .log-time {
    color: var(--text-muted, #555);
    flex-shrink: 0;
  }

  .log-text {
    color: var(--text-primary, #cdd6f4);
    word-break: break-all;
  }

  .log-entry.error .log-text {
    color: #f38ba8;
  }

  .log-entry.success .log-text {
    color: #a6e3a1;
  }

  .log-entry.warning .log-text {
    color: #f9e2af;
  }
</style>
