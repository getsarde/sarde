<script>
  import { open as openShell } from '@tauri-apps/plugin-shell'
  import { doc, preview, ui, addToast } from '../stores/app.svelte.js'
  import { startPreview, stopPreview } from '../api.js'
  import {
    Bold, Italic, Heading, Link, Image, Code, Braces, List, Table, ExternalLink, Play, Square, Eye,
  } from 'lucide-svelte'

  let { editor = null } = $props()

  let previewRunning = $derived(preview.port > 0)
  let previewFullUrl = $derived(previewRunning ? `http://localhost:${preview.port}` : '')
  let previewUrl = $derived(previewRunning ? `localhost:${preview.port}` : null)

  let starting = $state(false)

  function previewInBrowser() {
    if (!previewRunning) {
      addToast('warning', 'Preview server is not running')
      return
    }
    openShell(previewFullUrl)
  }

  async function startServer() {
    if (starting || previewRunning) return
    starting = true
    addToast('info', 'Starting preview server...')
    try {
      await startPreview()
    } catch (e) {
      addToast('error', `Preview failed: ${e}`)
    } finally {
      starting = false
    }
  }

  let stopping = $state(false)

  async function stopServer() {
    if (stopping || !previewRunning) return
    stopping = true
    try {
      await stopPreview()
      addToast('info', 'Preview server stopped')
    } catch (e) {
      addToast('error', `Failed to stop preview: ${e}`)
    } finally {
      stopping = false
    }
  }
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
    <div class="tool-sep"></div>
    <button
      class="tool-btn"
      class:active={ui.previewMode !== 'editor'}
      onclick={() => {
        const modes = ['editor', 'split', 'preview']
        const idx = modes.indexOf(ui.previewMode)
        ui.previewMode = modes[(idx + 1) % modes.length]
      }}
      title="Toggle preview (Ctrl+Shift+M)"
    >
      <Eye size={15} />
    </button>
  </div>

  <div class="preview-area">
    <span class="server-indicator" class:live={previewRunning} title={previewRunning ? 'Preview server running' : 'Preview server offline'}></span>
    {#if previewRunning && previewUrl}
      <span class="server-url">{previewUrl}</span>
      <button
        class="preview-btn"
        onclick={previewInBrowser}
        title="Open preview in browser (Ctrl+Shift+V)"
      >
        <ExternalLink size={13} />
        Open
      </button>
      <button
        class="preview-btn stop"
        onclick={stopServer}
        title="Stop preview server"
        disabled={stopping}
      >
        <Square size={11} />
        Stop
      </button>
    {:else}
      <span class="server-url muted">offline</span>
      <button
        class="preview-btn start"
        onclick={startServer}
        title="Start preview server"
        disabled={starting}
      >
        <Play size={13} />
        {starting ? 'Starting...' : 'Start'}
      </button>
    {/if}
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

  .tool-btn.active {
    color: var(--accent, #89b4fa);
    background: rgba(137, 180, 250, 0.1);
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

  .preview-btn.start {
    background: var(--bg-elevated, #2a2a3a);
    color: var(--text-primary, #cdd6f4);
    border: 1px solid var(--border, #2e2e3e);
  }

  .preview-btn.start:hover:not(:disabled) {
    background: var(--accent, #6366f1);
    color: #fff;
    border-color: transparent;
  }

  .preview-btn.stop {
    background: transparent;
    color: var(--text-muted, #888);
    border: 1px solid var(--border, #2e2e3e);
  }

  .preview-btn.stop:hover:not(:disabled) {
    color: #f38ba8;
    border-color: #f38ba8;
    background: rgba(243, 139, 168, 0.1);
  }
</style>
