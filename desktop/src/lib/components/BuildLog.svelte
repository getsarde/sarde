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
    border-top: 1px solid var(--cr-border);
    background: var(--cr-bg-surface);
    flex-shrink: 0;
  }

  .log-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 4px 10px;
    border-bottom: 1px solid var(--cr-border);
    flex-shrink: 0;
  }

  .log-title {
    font-size: 11px;
    font-weight: 600;
    color: var(--cr-text-muted);
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
    border-radius: var(--cr-radius-sm);
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
  }

  .log-action:hover {
    background: var(--cr-bg-elevated);
    color: var(--cr-text);
  }

  .log-list {
    flex: 1;
    overflow-y: auto;
    padding: 4px 0;
    font-family: var(--cr-font-mono);
    font-size: 12px;
    line-height: 1.6;
  }

  .log-empty {
    padding: 16px;
    text-align: center;
    font-size: 12px;
    color: var(--cr-text-muted);
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
    color: var(--cr-text-muted);
    flex-shrink: 0;
  }

  .log-text {
    color: var(--cr-text);
    word-break: break-all;
  }

  .log-entry.error .log-text {
    color: var(--cr-danger);
  }

  .log-entry.success .log-text {
    color: var(--cr-success);
  }

  .log-entry.warning .log-text {
    color: var(--cr-warning);
  }
</style>
