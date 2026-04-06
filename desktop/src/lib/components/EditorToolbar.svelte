<script>
  import { open as openShell } from '@tauri-apps/plugin-shell'
  import { doc, sidecar, addToast } from '../stores/app.svelte.js'
  import {
    Bold, Italic, Heading, Link, Image, Code, Braces, List, Table, ExternalLink,
  } from 'lucide-svelte'

  let { editor = null } = $props()

  function previewInBrowser() {
    if (!sidecar.previewUrl) {
      addToast('warning', 'Preview server is not running')
      return
    }
    openShell(sidecar.previewUrl)
  }

  let previewUrl = $derived(
    sidecar.previewUrl ? sidecar.previewUrl.replace(/^https?:\/\//, '') : null
  )
</script>

<div class="editor-toolbar">
  <div class="tool-group">
    <button class="tool-btn" onclick={() => editor?.wrapSelection('**', '**')} title="Bold (Ctrl+B)"><Bold size={15} /></button>
    <button class="tool-btn" onclick={() => editor?.wrapSelection('_', '_')}   title="Italic (Ctrl+I)"><Italic size={15} /></button>
    <button class="tool-btn" onclick={() => editor?.insertText('\n## ')}       title="Heading"><Heading size={15} /></button>
    <div class="tool-sep"></div>
    <button class="tool-btn" onclick={() => editor?.wrapSelection('[', '](url)')}    title="Link (Ctrl+K)"><Link size={15} /></button>
    <button class="tool-btn" onclick={() => editor?.insertText('\n![alt](url)\n')}   title="Image"><Image size={15} /></button>
    <div class="tool-sep"></div>
    <button class="tool-btn" onclick={() => editor?.wrapSelection('`', '`')}         title="Inline code"><Code size={15} /></button>
    <button class="tool-btn" onclick={() => editor?.insertText('\n```\n\n```\n')}    title="Code block"><Braces size={15} /></button>
    <div class="tool-sep"></div>
    <button class="tool-btn" onclick={() => editor?.insertText('\n- ')}              title="Bullet list"><List size={15} /></button>
    <button class="tool-btn" onclick={() => editor?.insertText('\n| Col 1 | Col 2 | Col 3 |\n| ----- | ----- | ----- |\n| Cell  | Cell  | Cell  |\n')} title="Table"><Table size={15} /></button>
  </div>

  <div class="preview-area">
    <span class="server-indicator" class:live={!!sidecar.previewUrl} title={sidecar.previewUrl ? 'Preview server running' : 'Preview server offline'}></span>
    {#if sidecar.previewUrl && previewUrl}
      <span class="server-url">{previewUrl}</span>
    {:else}
      <span class="server-url muted">offline</span>
    {/if}
    <button
      class="preview-btn"
      class:disabled={!sidecar.previewUrl}
      onclick={previewInBrowser}
      title="Open preview in browser (Ctrl+Shift+V)"
      disabled={!sidecar.previewUrl}
    >
      <ExternalLink size={13} />
      Open
    </button>
  </div>
</div>

<style>
  .editor-toolbar {
    display: flex;
    align-items: center;
    height: 36px;
    padding: 0 8px;
    background: var(--bg-surface, #1e1e2e);
    border-bottom: 1px solid var(--border, #2e2e3e);
    gap: 4px;
    user-select: none;
    overflow: hidden;
  }

  .tool-group {
    display: flex;
    align-items: center;
    gap: 2px;
  }

  .tool-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 28px;
    height: 26px;
    padding: 0 6px;
    border: none;
    background: transparent;
    color: var(--text-muted, #888);
    font-size: 13px;
    font-weight: 600;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.12s, color 0.12s;
  }

  .tool-btn:hover {
    background: var(--bg-elevated, #2a2a3a);
    color: var(--text-primary, #e0e0e0);
  }

  .tool-btn:active {
    background: var(--bg-base, #141420);
  }

  .tool-sep {
    width: 1px;
    height: 16px;
    background: var(--border, #2e2e3e);
    margin: 0 2px;
    flex-shrink: 0;
  }

  .preview-area {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-left: auto;
    padding: 0 4px 0 8px;
    border-left: 1px solid var(--border, #2e2e3e);
    flex-shrink: 0;
  }

  .server-indicator {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--text-muted, #555);
    flex-shrink: 0;
    transition: background 0.3s;
  }

  .server-indicator.live {
    background: #22c55e;
    box-shadow: 0 0 4px #22c55e88;
  }

  .server-url {
    font-size: 11px;
    color: var(--text-primary, #cdd6f4);
    font-family: var(--font-mono, monospace);
    white-space: nowrap;
  }

  .server-url.muted {
    color: var(--text-muted, #555);
  }

  .preview-btn {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 3px 10px;
    border: none;
    border-radius: 4px;
    background: var(--accent, #6366f1);
    color: #fff;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.12s;
    white-space: nowrap;
  }

  .preview-btn:hover:not(:disabled) {
    background: var(--accent-hover, #5558e6);
  }

  .preview-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
</style>
