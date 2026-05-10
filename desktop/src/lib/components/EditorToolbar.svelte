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
    background: var(--cr-bg-surface);
    border-bottom: 1px solid var(--cr-border);
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
    color: var(--cr-text-muted);
    font-size: 13px;
    font-weight: 600;
    border-radius: var(--cr-radius-sm);
    cursor: pointer;
    transition: background 0.12s, color 0.12s;
  }

  .tool-btn:hover {
    background: var(--cr-bg-elevated);
    color: var(--cr-text);
  }

  .tool-btn:active {
    background: var(--cr-bg-base);
  }

  .tool-btn.active {
    color: var(--cr-accent);
    background: var(--cr-active);
  }

  .tool-sep {
    width: 1px;
    height: 16px;
    background: var(--cr-border);
    margin: 0 2px;
    flex-shrink: 0;
  }

  .preview-area {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-left: auto;
    padding: 0 4px 0 8px;
    border-left: 1px solid var(--cr-border);
    flex-shrink: 0;
  }

  .server-indicator {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--cr-text-muted);
    flex-shrink: 0;
    transition: background 0.3s;
  }

  .server-indicator.live {
    background: #22c55e;
    box-shadow: 0 0 4px #22c55e88;
  }

  .server-url {
    font-size: 11px;
    color: var(--cr-text);
    font-family: var(--cr-font-mono);
    white-space: nowrap;
  }

  .server-url.muted {
    color: var(--cr-text-muted);
  }

  .preview-btn {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 3px 10px;
    border: none;
    border-radius: var(--cr-radius-sm);
    background: var(--cr-accent);
    color: #fff;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.12s;
    white-space: nowrap;
  }

  .preview-btn:hover:not(:disabled) {
    background: var(--cr-accent-hover);
  }

  .preview-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .preview-btn.start {
    background: var(--cr-bg-elevated);
    color: var(--cr-text);
    border: 1px solid var(--cr-border);
  }

  .preview-btn.start:hover:not(:disabled) {
    background: var(--cr-accent);
    color: #fff;
    border-color: transparent;
  }

  .preview-btn.stop {
    background: transparent;
    color: var(--cr-text-muted);
    border: 1px solid var(--cr-border);
  }

  .preview-btn.stop:hover:not(:disabled) {
    color: var(--cr-danger);
    border-color: var(--cr-danger);
    background: rgba(243, 139, 168, 0.1);
  }
</style>
