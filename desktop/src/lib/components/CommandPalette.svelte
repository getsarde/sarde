<script>
  import { ui, tabs, preview, addToast, closeTabById, warnings } from '../stores/app.svelte.js'
  import { build as apiBuild, startPreview, stopPreview } from '../api.js'
  import { Search } from 'lucide-svelte'
  import { open as openShell } from '@tauri-apps/plugin-shell'

  let query = $state('')
  let selectedIndex = $state(0)
  let inputEl = $state(null)

  const commands = [
    { id: 'new-file', label: 'New File', category: 'File', shortcut: 'Ctrl+N' },
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
  ]

  // Hide context-dependent commands
  let visibleCommands = $derived(
    commands.filter(c => {
      if (c.id === 'stop-preview') return preview.port > 0
      return true
    })
  )

  let filtered = $derived.by(() => {
    if (!query.trim()) return visibleCommands
    const q = query.toLowerCase()
    return visibleCommands.filter(c =>
      c.label.toLowerCase().includes(q) || c.category.toLowerCase().includes(q)
    )
  })

  $effect(() => {
    if (ui.commandPaletteOpen && inputEl) {
      inputEl.focus()
    }
  })

  // Reset state when palette opens
  $effect(() => {
    if (ui.commandPaletteOpen) {
      query = ''
      selectedIndex = 0
    }
  })

  function close() {
    ui.commandPaletteOpen = false
  }

  function execute(cmd) {
    close()
    switch (cmd.id) {
      case 'new-file':
        window.dispatchEvent(new CustomEvent('coderoo:new-file'))
        break
      case 'open-file':
        window.dispatchEvent(new CustomEvent('coderoo:open-file'))
        break
      case 'save':
        window.dispatchEvent(new CustomEvent('coderoo:save'))
        break
      case 'save-all':
        window.dispatchEvent(new CustomEvent('coderoo:save'))
        addToast('info', 'All files saved')
        break
      case 'close-tab':
        if (tabs.activeId) closeTabById(tabs.activeId)
        break
      case 'open-browser-preview':
        if (preview.port > 0) openShell(`http://localhost:${preview.port}`)
        else addToast('warning', 'Preview server not running')
        break
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
          addToast('error', `Build failed: ${e.message}`)
        })
        break
      case 'preview-site':
        if (preview.port > 0) {
          openShell(`http://localhost:${preview.port}`)
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
      default:
        console.log('Command executed:', cmd.id)
    }
  }

  function onKeydown(e) {
    if (e.key === 'Escape') {
      close()
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      selectedIndex = Math.min(selectedIndex + 1, filtered.length - 1)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      selectedIndex = Math.max(selectedIndex - 1, 0)
    } else if (e.key === 'Enter' && filtered[selectedIndex]) {
      e.preventDefault()
      execute(filtered[selectedIndex])
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

{#if ui.commandPaletteOpen}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="palette-overlay" onclick={close} role="presentation">
    <div class="palette" onclick={(e) => e.stopPropagation()} onkeydown={onKeydown} role="dialog" aria-label="Command palette" tabindex="-1">
      <div class="palette-input-row">
        <Search size={16} class="palette-search-icon" />
        <input
          bind:this={inputEl}
          bind:value={query}
          type="text"
          class="palette-input"
          placeholder="Type a command..."
        />
      </div>

      <div class="palette-list">
        {#if filtered.length === 0}
          <div class="palette-empty">No matching commands</div>
        {:else}
          {#each filtered as cmd, i}
            <button
              class="palette-item"
              class:selected={i === selectedIndex}
              onclick={() => execute(cmd)}
              onmouseenter={() => (selectedIndex = i)}
            >
              <span class="palette-category">{cmd.category}</span>
              <span class="palette-label">{cmd.label}</span>
              {#if cmd.shortcut}
                <span class="palette-shortcut">{cmd.shortcut}</span>
              {/if}
            </button>
          {/each}
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .palette-overlay {
    position: fixed;
    inset: 0;
    z-index: 200;
    display: flex;
    justify-content: center;
    padding-top: 20vh;
    background: rgba(0, 0, 0, 0.4);
  }

  .palette {
    width: 500px;
    max-width: 90vw;
    max-height: 380px;
    display: flex;
    flex-direction: column;
    background: var(--color-surface, #1e1e2e);
    border: 1px solid var(--color-border, #313244);
    border-radius: 10px;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
    overflow: hidden;
    align-self: flex-start;
  }

  .palette-input-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 14px;
    border-bottom: 1px solid var(--color-border, #313244);
  }

  :global(.palette-search-icon) {
    flex-shrink: 0;
    color: var(--color-text-muted, #6c7086);
  }

  .palette-input {
    flex: 1;
    border: none;
    background: transparent;
    font-size: 14px;
    color: var(--color-text, #cdd6f4);
    outline: none;
    font-family: inherit;
  }

  .palette-input::placeholder {
    color: var(--color-text-muted, #6c7086);
  }

  .palette-list {
    flex: 1;
    overflow-y: auto;
    padding: 4px;
  }

  .palette-empty {
    padding: 20px;
    text-align: center;
    font-size: 13px;
    color: var(--color-text-muted, #6c7086);
  }

  .palette-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 8px 10px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--color-text, #cdd6f4);
    font-size: 13px;
    text-align: left;
    cursor: pointer;
  }

  .palette-item:hover,
  .palette-item.selected {
    background: var(--color-active, rgba(137, 180, 250, 0.1));
  }

  .palette-category {
    font-size: 11px;
    color: var(--color-text-muted, #6c7086);
    min-width: 70px;
  }

  .palette-label {
    flex: 1;
  }

  .palette-shortcut {
    font-size: 11px;
    color: var(--color-text-muted, #6c7086);
    padding: 2px 6px;
    border-radius: 4px;
    background: var(--color-surface-alt, #181825);
    border: 1px solid var(--color-border, #313244);
    font-family: inherit;
  }
</style>
