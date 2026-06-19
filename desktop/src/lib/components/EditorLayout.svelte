<script>
  import { ui, preview, tabs, doc, mdPreview } from '../stores/app.svelte.js'
  import { renderMarkdown } from '../api.js'
  import { contentPathToUrl } from '../utils/url-mapping.js'
  import { open as openShell } from '@tauri-apps/plugin-shell'

  const PREVIEW_MODES = ['editor', 'split', 'preview']

  function cyclePreviewMode() {
    const idx = PREVIEW_MODES.indexOf(ui.previewMode)
    ui.previewMode = PREVIEW_MODES[(idx + 1) % PREVIEW_MODES.length]
  }

  function onGlobalKeydown(e) {
    const ctrl = e.ctrlKey || e.metaKey

    if (ctrl && e.key === 'p' && !e.shiftKey) {
      e.preventDefault()
      ui.commandPaletteOpen = true
    } else if (ctrl && e.key === 's') {
      e.preventDefault()
      window.dispatchEvent(new CustomEvent('sarde:save'))
    } else if (ctrl && e.key === '\\') {
      e.preventDefault()
      ui.leftPanel = ui.leftPanel ? null : 'files'
    } else if (ctrl && e.key === ',') {
      e.preventDefault()
      ui.settingsOpen = !ui.settingsOpen
    } else if (ctrl && e.shiftKey && e.key === 'V') {
      e.preventDefault()
      if (preview.port > 0) {
        const pageUrl = contentPathToUrl(doc.contentPath)
        openShell(`http://localhost:${preview.port}${pageUrl}`)
      }
    } else if (ctrl && e.shiftKey && e.key === 'M') {
      e.preventDefault()
      cyclePreviewMode()
    } else if (ctrl && e.key === '/') {
      e.preventDefault()
      ui.shortcutsOpen = !ui.shortcutsOpen
    }
  }

  // Debounced markdown rendering when preview is visible
  let renderVersion = 0

  $effect(() => {
    const content = doc.content
    const mode = ui.previewMode

    if (mode === 'editor') return

    const version = ++renderVersion
    const timer = setTimeout(async () => {
      if (!content) {
        mdPreview.html = ''
        return
      }
      mdPreview.rendering = true
      mdPreview.error = null
      try {
        const result = await renderMarkdown(content)
        if (version !== renderVersion) return
        mdPreview.html = result?.html ?? ''
      } catch (e) {
        if (version !== renderVersion) return
        mdPreview.error = String(e)
      } finally {
        if (version === renderVersion) mdPreview.rendering = false
      }
    }, 300)

    return () => clearTimeout(timer)
  })
  import MarkdownPreview from './MarkdownPreview.svelte'
  import LeftSidebar from './LeftSidebar.svelte'
  import RightSidebar from './RightSidebar.svelte'
  import EditorToolbar from './EditorToolbar.svelte'
  import CodeEditor from './CodeEditor.svelte'
  import EmptyEditor from './EmptyEditor.svelte'
  import StatusBar from './StatusBar.svelte'
  import BuildLog from './BuildLog.svelte'
  import CommandPalette from './CommandPalette.svelte'
  import SettingsModal from './SettingsModal.svelte'
  import DeployModal from './DeployModal.svelte'
  import ImportModal from './ImportModal.svelte'
  import CreateContentDialog from './CreateContentDialog.svelte'
  import ShortcutsDialog from './ShortcutsDialog.svelte'
  import ToastContainer from './ToastContainer.svelte'

  let { projectName = '', onClose } = $props()
  let editorRef = $state(null)
</script>

<svelte:window onkeydown={onGlobalKeydown} />

<div class="editor-layout" role="application" aria-label="Sarde Editor">
  <LeftSidebar />

  <main class="main-area" aria-label="Editor">
    <EditorToolbar editor={editorRef} />

    <div class="editor-content">
      {#if ui.previewMode !== 'preview'}
        <div class="editor-pane" class:split={ui.previewMode === 'split'}>
          {#if tabs.items.length > 0}
            <CodeEditor bind:this={editorRef} />
          {:else}
            <EmptyEditor />
          {/if}
        </div>
      {/if}

      {#if ui.previewMode !== 'editor'}
        <div class="preview-pane" class:split={ui.previewMode === 'split'}>
          <MarkdownPreview />
        </div>
      {/if}
    </div>

    <BuildLog />
    <StatusBar />
  </main>

  <RightSidebar />

  <CommandPalette />
  <SettingsModal />
  <DeployModal />
  <ImportModal />
  <CreateContentDialog />
  <ShortcutsDialog />
  <ToastContainer />
</div>

<style>
  .editor-layout {
    display: flex;
    width: 100%;
    height: 100vh;
    overflow: hidden;
  }

  .main-area {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .editor-content {
    flex: 1;
    display: flex;
    overflow: hidden;
  }

  .editor-pane {
    flex: 1;
    min-width: 0;
    overflow: hidden;
  }

  .editor-pane.split {
    flex: 0 0 50%;
    max-width: 50%;
  }

  .preview-pane {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    display: flex;
    border-left: 1px solid var(--sd-border);
  }

  .preview-pane.split {
    flex: 0 0 50%;
    max-width: 50%;
  }
</style>
