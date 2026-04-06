<script>
  import { ui, sidecar } from '../stores/app.svelte.js'
  import { open as openShell } from '@tauri-apps/plugin-shell'

  function onGlobalKeydown(e) {
    const ctrl = e.ctrlKey || e.metaKey

    if (ctrl && e.key === 'p') {
      e.preventDefault() // block browser print dialog
    } else if (ctrl && e.key === 's') {
      e.preventDefault()
      window.dispatchEvent(new CustomEvent('coderoo:save'))
    } else if (ctrl && e.key === '\\') {
      e.preventDefault()
      ui.leftPanel = ui.leftPanel ? null : 'files'
    } else if (ctrl && e.key === ',') {
      e.preventDefault()
      ui.settingsOpen = !ui.settingsOpen
    } else if (ctrl && e.shiftKey && e.key === 'V') {
      e.preventDefault()
      if (sidecar.url) openShell(sidecar.url)
    }
  }
  import { tabs } from '../stores/app.svelte.js'
  import LeftSidebar from './LeftSidebar.svelte'
  import RightSidebar from './RightSidebar.svelte'
  import TabBar from './TabBar.svelte'
  import EditorToolbar from './EditorToolbar.svelte'
  import CodeEditor from './CodeEditor.svelte'
  import EmptyEditor from './EmptyEditor.svelte'
  import StatusBar from './StatusBar.svelte'
  import CommandPalette from './CommandPalette.svelte'
  import SettingsModal from './SettingsModal.svelte'
  import ToastContainer from './ToastContainer.svelte'

  let { projectName = '', onClose } = $props()
  let editorRef = $state(null)
</script>

<svelte:window onkeydown={onGlobalKeydown} />

<div class="editor-layout">
  <LeftSidebar />

  <div class="main-area">
    <TabBar />
    <EditorToolbar editor={editorRef} />

    <div class="editor-content">
      <div class="editor-pane">
        {#if tabs.items.length > 0}
          <CodeEditor bind:this={editorRef} />
        {:else}
          <EmptyEditor />
        {/if}
      </div>
    </div>

    <StatusBar />
  </div>

  <RightSidebar />

  <CommandPalette />
  {#if ui.settingsOpen}
    <SettingsModal />
  {/if}
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
</style>
