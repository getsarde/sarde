<script>
  import { ui, tabs, preview, doc, addToast, closeTabById, requestCloseTab, warnings } from '../stores/app.svelte.js'
  import { build as apiBuild, startPreview, stopPreview } from '../api.js'
  import { contentPathToUrl } from '../utils/url-mapping.js'
  import { Dialog, Command } from 'bits-ui'
  import { Search } from 'lucide-svelte'
  import { open as openShell } from '@tauri-apps/plugin-shell'

  const commands = [
    { id: 'new-file', label: 'New File', category: 'File', shortcut: 'Ctrl+N' },
    { id: 'new-blog-post', label: 'New Blog Post', category: 'File' },
    { id: 'new-doc-page', label: 'New Doc Page', category: 'File' },
    { id: 'open-file', label: 'Open File', category: 'File', shortcut: 'Ctrl+O' },
    { id: 'save', label: 'Save', category: 'File', shortcut: 'Ctrl+S' },
    { id: 'save-all', label: 'Save All', category: 'File', shortcut: 'Ctrl+Shift+S' },
    { id: 'close-tab', label: 'Close Tab', category: 'File', shortcut: 'Ctrl+W' },
    { id: 'open-browser-preview', label: 'Open Preview in Browser', category: 'View', shortcut: 'Ctrl+Shift+V' },
    { id: 'toggle-sidebar-left', label: 'Toggle Left Sidebar', category: 'View', shortcut: 'Ctrl+\\' },
    { id: 'toggle-sidebar-right', label: 'Toggle Right Sidebar', category: 'View' },
    { id: 'open-settings', label: 'Open Settings', category: 'Preferences', shortcut: 'Ctrl+,' },
    { id: 'build-site', label: 'Build Site', category: 'Build', shortcut: 'Ctrl+Shift+B' },
    { id: 'preview-site', label: 'Preview Site', category: 'Build' },
    { id: 'stop-preview', label: 'Stop Preview Server', category: 'Build' },
    { id: 'deploy', label: 'Deploy Site', category: 'Deploy' },
    { id: 'import-obsidian', label: 'Import Obsidian Vault', category: 'Import' },
    { id: 'find-in-files', label: 'Search in Files', category: 'Search', shortcut: 'Ctrl+Shift+F' },
    { id: 'show-shortcuts', label: 'Keyboard Shortcuts', category: 'Help', shortcut: 'Ctrl+/' },
    { id: 'show-onboarding', label: 'Welcome Tour', category: 'Help' },
  ]

  // Hide context-dependent commands
  let visibleCommands = $derived(
    commands.filter(c => {
      if (c.id === 'stop-preview') return preview.port > 0
      return true
    })
  )

  function close() {
    ui.commandPaletteOpen = false
  }

  function execute(cmd) {
    close()
    switch (cmd.id) {
      case 'new-file':
        window.dispatchEvent(new CustomEvent('sarde:new-file'))
        break
      case 'open-file':
        window.dispatchEvent(new CustomEvent('sarde:open-file'))
        break
      case 'save':
        window.dispatchEvent(new CustomEvent('sarde:save'))
        break
      case 'save-all':
        window.dispatchEvent(new CustomEvent('sarde:save'))
        window.dispatchEvent(new CustomEvent('sarde:save-all'))
        break
      case 'close-tab':
        if (tabs.activeId) requestCloseTab(tabs.activeId)
        break
      case 'new-blog-post':
        ui.createContentType = 'blog'
        break
      case 'new-doc-page':
        ui.createContentType = 'docs'
        break
      case 'open-browser-preview': {
        if (preview.port > 0) {
          const pageUrl = contentPathToUrl(doc.contentPath)
          openShell(`http://localhost:${preview.port}${pageUrl}`)
        } else {
          addToast('warning', 'Preview server not running')
        }
        break
      }
      case 'toggle-sidebar-left':
        ui.leftPanel = ui.leftPanel ? null : 'files'
        break
      case 'toggle-sidebar-right':
        ui.rightPanel = ui.rightPanel ? null : 'toc'
        break
      case 'open-settings':
        ui.settingsOpen = true
        break
      case 'build-site':
        addToast('info', 'Build started...')
        apiBuild().then(resp => {
          const w = resp?.warnings ?? []
          warnings.items = w
          const wCount = w.length
          if (wCount > 0) {
            addToast('warning', `Build complete with ${wCount} warning${wCount === 1 ? '' : 's'}`)
          } else {
            addToast('success', 'Build complete')
          }
        }).catch(e => {
          addToast('error', `Build failed: ${e?.message ?? e}`)
        })
        break
      case 'preview-site':
        if (preview.port > 0) {
          const previewPageUrl = contentPathToUrl(doc.contentPath)
          openShell(`http://localhost:${preview.port}${previewPageUrl}`)
        } else {
          addToast('info', 'Starting preview server...')
          startPreview().catch(e => addToast('error', `Preview failed: ${e}`))
        }
        break
      case 'stop-preview':
        if (preview.port > 0) {
          stopPreview().catch(e => addToast('error', `Failed to stop: ${e}`))
        } else {
          addToast('info', 'Preview server is not running')
        }
        break
      case 'deploy':
        ui.deployOpen = true
        break
      case 'import-obsidian':
        ui.importOpen = true
        break
      case 'find-in-files':
        ui.leftPanel = 'search'
        break
      case 'show-shortcuts':
        ui.shortcutsOpen = true
        break
      case 'show-onboarding':
        window.dispatchEvent(new CustomEvent('sarde:show-onboarding'))
        break
      default:
        console.log('Command executed:', cmd.id)
    }
  }

  function onGlobalKeydown(e) {
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'P') {
      e.preventDefault()
      ui.commandPaletteOpen = !ui.commandPaletteOpen
    }
  }
</script>

<svelte:window onkeydown={onGlobalKeydown} />

<Dialog.Root
  open={ui.commandPaletteOpen}
  onOpenChange={(v) => (ui.commandPaletteOpen = v)}
>
  <Dialog.Portal>
    <Dialog.Overlay class="palette-overlay" />
    <Dialog.Content class="palette-container" aria-label="Command palette">
      <Command.Root label="Command palette" shouldFilter={true} loop={true}>
        <div class="palette-input-row">
          <Search size={16} class="palette-search-icon" />
          <Command.Input class="palette-input" placeholder="Type a command..." />
        </div>

        <Command.List class="palette-list">
          <Command.Empty class="palette-empty">No matching commands</Command.Empty>
          {#each visibleCommands as cmd (cmd.id)}
            <Command.Item
              value={cmd.id}
              keywords={[cmd.label, cmd.category]}
              onSelect={() => execute(cmd)}
              class="palette-item"
            >
              <span class="palette-category">{cmd.category}</span>
              <span class="palette-label">{cmd.label}</span>
              {#if cmd.shortcut}
                <span class="palette-shortcut">{cmd.shortcut}</span>
              {/if}
            </Command.Item>
          {/each}
        </Command.List>
      </Command.Root>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>

<style>
  :global(.palette-overlay) {
    position: fixed;
    inset: 0;
    z-index: 200;
    background: rgba(0, 0, 0, 0.4);
  }

  :global(.palette-container) {
    position: fixed;
    top: 20vh;
    left: 50%;
    transform: translateX(-50%);
    z-index: 200;
    width: 500px;
    max-width: 90vw;
    max-height: 380px;
    display: flex;
    flex-direction: column;
    background: var(--cr-bg-base);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-lg);
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
    overflow: hidden;
    outline: none;
  }

  .palette-input-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 14px;
    border-bottom: 1px solid var(--cr-border);
  }

  :global(.palette-search-icon) {
    flex-shrink: 0;
    color: var(--cr-text-muted);
  }

  :global(.palette-input) {
    flex: 1;
    border: none;
    background: transparent;
    font-size: 14px;
    color: var(--cr-text);
    outline: none;
    font-family: inherit;
  }

  :global(.palette-input)::placeholder {
    color: var(--cr-text-muted);
  }

  :global(.palette-list) {
    flex: 1;
    overflow-y: auto;
    padding: 4px;
  }

  :global(.palette-empty) {
    padding: 20px;
    text-align: center;
    font-size: 13px;
    color: var(--cr-text-muted);
  }

  :global(.palette-item) {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 8px 10px;
    border: none;
    border-radius: var(--cr-radius);
    background: transparent;
    color: var(--cr-text);
    font-size: 13px;
    text-align: left;
    cursor: pointer;
  }

  :global(.palette-item[data-selected]) {
    background: var(--cr-active);
  }

  .palette-category {
    font-size: 11px;
    color: var(--cr-text-muted);
    min-width: 70px;
  }

  .palette-label {
    flex: 1;
  }

  .palette-shortcut {
    font-size: 11px;
    color: var(--cr-text-muted);
    padding: 2px 6px;
    border-radius: var(--cr-radius-sm);
    background: var(--cr-bg-input);
    border: 1px solid var(--cr-border);
    font-family: inherit;
  }
</style>
