<script>
  import { getCurrentWindow } from '@tauri-apps/api/window'
  import { ui } from '../stores/app.svelte.js'
  import { PanelLeft, PanelRight } from 'lucide-svelte'
  import TabBar from './TabBar.svelte'

  let { showTabs = false } = $props()

  const appWindow = getCurrentWindow()
  let isMaximized = $state(false)

  $effect(() => {
    appWindow.isMaximized().then(v => { isMaximized = v })
    const promise = appWindow.onResized(() => {
      appWindow.isMaximized().then(v => { isMaximized = v })
    })
    return () => { promise.then(unlisten => unlisten()) }
  })

  function minimize() { appWindow.minimize() }
  function toggleMaximize() { appWindow.toggleMaximize() }
  function close() { appWindow.close() }

  function toggleLeft() {
    ui.leftPanel = ui.leftPanel ? null : 'files'
  }

  function toggleRight() {
    ui.rightPanel = ui.rightPanel ? null : 'toc'
  }

  function isInteractive(e) {
    return e.target.closest('.win-controls') || e.target.closest('.tab') || e.target.closest('button')
  }

  function onMouseDown(e) {
    if (e.button !== 0 || isInteractive(e)) return
    appWindow.startDragging()
  }

  function onDblClick(e) {
    if (isInteractive(e)) return
    toggleMaximize()
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="titlebar" onmousedown={onMouseDown} ondblclick={onDblClick}>
  <div class="titlebar-left">
    {#if showTabs}
      <button class="titlebar-btn" onclick={toggleLeft} title="Toggle left sidebar">
        <PanelLeft size={16} />
      </button>
    {/if}
  </div>

  <div class="titlebar-center">
    {#if showTabs}
      <TabBar embedded={true} />
    {:else}
      <span class="app-title">Coderoo</span>
    {/if}
  </div>

  <div class="titlebar-right">
    {#if showTabs}
      <button class="titlebar-btn" onclick={toggleRight} title="Toggle right sidebar">
        <PanelRight size={16} />
      </button>
    {/if}
    <div class="win-controls">
      <button class="wc-btn" onclick={minimize} title="Minimize">
        <svg width="10" height="10" viewBox="0 0 10 10">
          <line x1="0" y1="5" x2="10" y2="5" stroke="currentColor" stroke-width="1" />
        </svg>
      </button>
      <button class="wc-btn" onclick={toggleMaximize} title={isMaximized ? 'Restore' : 'Maximize'}>
        {#if isMaximized}
          <svg width="10" height="10" viewBox="0 0 10 10">
            <polyline points="2,3 2,0 10,0 10,7 7,7" fill="none" stroke="currentColor" stroke-width="1" />
            <rect x="0" y="3" width="7" height="7" fill="none" stroke="currentColor" stroke-width="1" />
          </svg>
        {:else}
          <svg width="10" height="10" viewBox="0 0 10 10">
            <rect x="0" y="0" width="10" height="10" fill="none" stroke="currentColor" stroke-width="1" />
          </svg>
        {/if}
      </button>
      <button class="wc-btn close-btn" onclick={close} title="Close">
        <svg width="10" height="10" viewBox="0 0 10 10">
          <line x1="0" y1="0" x2="10" y2="10" stroke="currentColor" stroke-width="1" />
          <line x1="10" y1="0" x2="0" y2="10" stroke="currentColor" stroke-width="1" />
        </svg>
      </button>
    </div>
  </div>
</div>

<style>
  .titlebar {
    display: flex;
    align-items: center;
    height: var(--cr-titlebar-height, 38px);
    background: var(--cr-bg-surface);
    border-bottom: 1px solid var(--cr-border);
    user-select: none;
    flex-shrink: 0;
  }

  .titlebar-left {
    width: var(--cr-icon-strip-width, 48px);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    height: 100%;
  }

  .titlebar-center {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    overflow: hidden;
    height: 100%;
  }

  .titlebar-right {
    display: flex;
    align-items: center;
    flex-shrink: 0;
    height: 100%;
  }

  .app-title {
    flex: 1;
    text-align: center;
    font-size: 13px;
    color: var(--cr-text-muted);
    pointer-events: none;
  }

  .titlebar-btn {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    color: var(--cr-text-muted);
    border-radius: var(--cr-radius-sm);
    cursor: pointer;
    transition: background 0.1s, color 0.1s;
  }

  .titlebar-btn:hover {
    background: var(--cr-hover);
    color: var(--cr-text);
  }

  .win-controls {
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }

  .wc-btn {
    width: 46px;
    height: var(--cr-titlebar-height, 38px);
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
    transition: background 0.1s, color 0.1s;
  }

  .wc-btn:hover {
    background: var(--cr-hover);
    color: var(--cr-text);
  }

  .close-btn:hover {
    background: #e81123;
    color: white;
  }
</style>
