<script>
  import { open as openShell } from '@tauri-apps/plugin-shell'
  import { doc, preview, ui, tabs, addToast } from '../stores/app.svelte.js'
  import { startPreview, stopPreview } from '../api.js'
  import { contentPathToUrl } from '../utils/url-mapping.js'
  import {
    Bold, Italic, Heading, Link, Image, Code, Braces, List, Table, ExternalLink, Play, Square, Eye, Save, MonitorSmartphone, Tablet, Monitor,
  } from 'lucide-svelte'
  import AppTooltip from './primitives/AppTooltip.svelte'

  let { editor = null } = $props()

  let hasFile = $derived(tabs.items.length > 0)
  let previewRunning = $derived(preview.port > 0)
  let pageUrl = $derived(contentPathToUrl(doc.contentPath))
  let previewFullUrl = $derived(previewRunning ? `http://localhost:${preview.port}${pageUrl}` : '')
  let previewUrl = $derived(previewRunning ? `localhost:${preview.port}${pageUrl}` : null)

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
    <AppTooltip content="Save (Ctrl+S)">
      <button class="tool-btn" disabled={!hasFile || !doc.dirty} onclick={() => window.dispatchEvent(new CustomEvent('sarde:save'))}><Save size={15} /></button>
    </AppTooltip>
    <div class="tool-sep"></div>
    <AppTooltip content="Bold (Ctrl+B)">
      <button class="tool-btn" disabled={!hasFile} onclick={() => editor?.wrapSelection('**', '**')}><Bold size={15} /></button>
    </AppTooltip>
    <AppTooltip content="Italic (Ctrl+I)">
      <button class="tool-btn" disabled={!hasFile} onclick={() => editor?.wrapSelection('_', '_')}><Italic size={15} /></button>
    </AppTooltip>
    <AppTooltip content="Heading">
      <button class="tool-btn" disabled={!hasFile} onclick={() => editor?.insertText('\n## ')}><Heading size={15} /></button>
    </AppTooltip>
    <div class="tool-sep"></div>
    <AppTooltip content="Link (Ctrl+K)">
      <button class="tool-btn" disabled={!hasFile} onclick={() => editor?.wrapSelection('[', '](url)')}><Link size={15} /></button>
    </AppTooltip>
    <AppTooltip content="Insert image">
      <button class="tool-btn" disabled={!hasFile} onclick={() => { ui.rightPanel = 'assets' }}><Image size={15} /></button>
    </AppTooltip>
    <div class="tool-sep"></div>
    <AppTooltip content="Inline code">
      <button class="tool-btn" disabled={!hasFile} onclick={() => editor?.wrapSelection('`', '`')}><Code size={15} /></button>
    </AppTooltip>
    <AppTooltip content="Code block">
      <button class="tool-btn" disabled={!hasFile} onclick={() => {
        const v = editor?.getView()
        if (!v) return
        const cursor = v.state.selection.main.head
        const text = '\n```\n\n```\n'
        v.dispatch({ changes: { from: cursor, insert: text }, selection: { anchor: cursor + 4 } })
        v.focus()
      }}><Braces size={15} /></button>
    </AppTooltip>
    <div class="tool-sep"></div>
    <AppTooltip content="Bullet list">
      <button class="tool-btn" disabled={!hasFile} onclick={() => editor?.insertText('\n- ')}><List size={15} /></button>
    </AppTooltip>
    <AppTooltip content="Table">
      <button class="tool-btn" disabled={!hasFile} onclick={() => editor?.insertText('\n| Col 1 | Col 2 | Col 3 |\n| ----- | ----- | ----- |\n| Cell  | Cell  | Cell  |\n')}><Table size={15} /></button>
    </AppTooltip>
    {#if false}
    <div class="tool-sep"></div>
    <div class="mode-group">
      <AppTooltip content="Editor only">
        <button class="mode-btn" class:active={ui.previewMode === 'editor'} onclick={() => ui.previewMode = 'editor'}><Code size={13} /></button>
      </AppTooltip>
      <AppTooltip content="Split view">
        <button class="mode-btn" class:active={ui.previewMode === 'split'} onclick={() => ui.previewMode = 'split'}><Tablet size={13} /></button>
      </AppTooltip>
      <AppTooltip content="Preview only">
        <button class="mode-btn" class:active={ui.previewMode === 'preview'} onclick={() => ui.previewMode = 'preview'}><Eye size={13} /></button>
      </AppTooltip>
    </div>
    {/if}
  </div>

  <div class="preview-area">
    <span class="server-indicator" class:live={previewRunning}></span>
    {#if previewRunning && previewUrl}
      <span class="server-url">{previewUrl}</span>
      <AppTooltip content="Open preview in browser (Ctrl+Shift+V)">
        <button
          class="preview-btn"
          onclick={previewInBrowser}
        >
          <ExternalLink size={13} />
          Open
        </button>
      </AppTooltip>
      <AppTooltip content="Stop preview server">
        <button
          class="preview-btn stop"
          onclick={stopServer}
          disabled={stopping}
        >
          <Square size={11} />
          Stop
        </button>
      </AppTooltip>
    {:else}
      <span class="server-url muted">offline</span>
      <AppTooltip content="Start preview server">
        <button
          class="preview-btn start"
          onclick={startServer}
          disabled={starting}
        >
          <Play size={13} />
          {starting ? 'Starting...' : 'Start'}
        </button>
      </AppTooltip>
    {/if}
  </div>
</div>

<style>
  .editor-toolbar {
    display: flex;
    align-items: center;
    height: 36px;
    padding: 0 8px;
    background: var(--sd-bg-surface);
    border-bottom: 1px solid var(--sd-border);
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
    color: var(--sd-text-muted);
    font-size: 13px;
    font-weight: 600;
    border-radius: var(--sd-radius-sm);
    cursor: pointer;
    transition: background 0.12s, color 0.12s;
  }

  .tool-btn:hover {
    background: var(--sd-bg-elevated);
    color: var(--sd-text);
  }

  .tool-btn:active {
    background: var(--sd-bg-base);
  }

  .tool-btn:disabled {
    opacity: 0.3;
    pointer-events: none;
  }

  :global(.tool-btn[data-state="on"]) {
    color: var(--sd-accent);
    background: var(--sd-active);
  }

  .mode-group {
    display: flex;
    gap: 1px;
    background: var(--sd-border);
    border-radius: var(--sd-radius-sm);
    overflow: hidden;
  }

  .mode-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 24px;
    border: none;
    background: var(--sd-bg-surface);
    color: var(--sd-text-muted);
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
  }

  .mode-btn:hover { background: var(--sd-bg-elevated); color: var(--sd-text); }
  .mode-btn.active { background: var(--sd-active); color: var(--sd-accent); }

  .tool-sep {
    width: 1px;
    height: 16px;
    background: var(--sd-border);
    margin: 0 2px;
    flex-shrink: 0;
  }

  .preview-area {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-left: auto;
    padding: 0 4px 0 8px;
    border-left: 1px solid var(--sd-border);
    flex-shrink: 0;
  }

  .server-indicator {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--sd-text-muted);
    flex-shrink: 0;
    transition: background 0.3s;
  }

  .server-indicator.live {
    background: var(--sd-success);
    box-shadow: 0 0 4px color-mix(in srgb, var(--sd-success) 50%, transparent);
  }

  .server-url {
    font-size: 11px;
    color: var(--sd-text);
    font-family: var(--sd-font-mono);
    white-space: nowrap;
  }

  .server-url.muted {
    color: var(--sd-text-muted);
  }

  .preview-btn {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 3px 10px;
    border: none;
    border-radius: var(--sd-radius-sm);
    background: var(--sd-accent);
    color: #fff;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.12s;
    white-space: nowrap;
  }

  .preview-btn:hover:not(:disabled) {
    background: var(--sd-accent-hover);
  }

  .preview-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .preview-btn.start {
    background: var(--sd-bg-elevated);
    color: var(--sd-text);
    border: 1px solid var(--sd-border);
  }

  .preview-btn.start:hover:not(:disabled) {
    background: var(--sd-accent);
    color: #fff;
    border-color: transparent;
  }

  .preview-btn.stop {
    background: transparent;
    color: var(--sd-text-muted);
    border: 1px solid var(--sd-border);
  }

  .preview-btn.stop:hover:not(:disabled) {
    color: var(--sd-danger);
    border-color: var(--sd-danger);
    background: var(--sd-danger-bg);
  }
</style>
