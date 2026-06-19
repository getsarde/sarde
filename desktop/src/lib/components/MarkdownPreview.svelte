<script>
  import { mdPreview } from '../stores/app.svelte.js'
  import { Loader, Monitor, Tablet, Smartphone, Maximize } from 'lucide-svelte'

  const VIEWPORTS = [
    { id: 'auto', label: 'Auto', icon: Maximize, width: null },
    { id: 'desktop', label: 'Desktop', icon: Monitor, width: 1280 },
    { id: 'tablet', label: 'Tablet', icon: Tablet, width: 768 },
    { id: 'mobile', label: 'Mobile', icon: Smartphone, width: 375 },
  ]

  let activeViewport = $state('auto')
  let viewportWidth = $derived(VIEWPORTS.find(v => v.id === activeViewport)?.width)
</script>

<div class="preview-wrapper">
  <div class="viewport-bar">
    {#each VIEWPORTS as vp}
      {@const Icon = vp.icon}
      <button
        class="viewport-btn"
        class:active={activeViewport === vp.id}
        onclick={() => activeViewport = vp.id}
        title="{vp.label}{vp.width ? ` (${vp.width}px)` : ''}"
      >
        <Icon size={13} />
      </button>
    {/each}
    {#if viewportWidth}
      <span class="viewport-label">{viewportWidth}px</span>
    {/if}
  </div>

  <div class="preview-frame" style={viewportWidth ? `max-width: ${viewportWidth}px; margin: 0 auto;` : ''}>
    <div class="preview-container">
      {#if mdPreview.rendering && !mdPreview.html}
        <div class="preview-loading">
          <Loader size={20} />
          <span>Rendering...</span>
        </div>
      {:else if mdPreview.error}
        <div class="preview-error">
          <p>Render error: {mdPreview.error}</p>
        </div>
      {:else if mdPreview.html}
        <div class="prose">
          {@html mdPreview.html}
        </div>
      {:else}
        <div class="preview-empty">
          <p>Start typing to see preview</p>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .preview-wrapper {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
    background: var(--sd-bg-base);
  }

  .viewport-bar {
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 4px 8px;
    border-bottom: 1px solid var(--sd-border);
    background: var(--sd-bg-surface);
    flex-shrink: 0;
  }

  .viewport-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 22px;
    border: none;
    border-radius: var(--sd-radius-sm);
    background: transparent;
    color: var(--sd-text-muted);
    cursor: pointer;
  }

  .viewport-btn:hover {
    color: var(--sd-text);
    background: var(--sd-hover);
  }

  .viewport-btn.active {
    color: var(--sd-accent);
    background: var(--sd-active);
  }

  .viewport-label {
    font-size: 10px;
    color: var(--sd-text-muted);
    margin-left: 4px;
    font-family: var(--sd-font-mono);
  }

  .preview-frame {
    flex: 1;
    overflow: hidden;
    transition: max-width 0.2s ease;
  }

  .preview-container {
    height: 100%;
    overflow-y: auto;
    padding: 24px 32px;
    color: var(--sd-text);
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    font-size: 15px;
    line-height: 1.7;
  }

  .preview-loading,
  .preview-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    height: 100%;
    color: var(--sd-text-muted);
    font-size: 13px;
  }

  .preview-error {
    padding: 16px;
    color: var(--sd-danger);
    font-size: 13px;
  }

  /* Prose styles for rendered markdown */
  .prose :global(h1) {
    font-size: 2em;
    font-weight: 700;
    margin: 0 0 0.5em;
    padding-bottom: 0.3em;
    border-bottom: 1px solid var(--sd-border);
    color: var(--sd-text);
  }

  .prose :global(h2) {
    font-size: 1.5em;
    font-weight: 600;
    margin: 1.5em 0 0.5em;
    padding-bottom: 0.2em;
    border-bottom: 1px solid var(--sd-border);
    color: var(--sd-text);
  }

  .prose :global(h3) {
    font-size: 1.25em;
    font-weight: 600;
    margin: 1.2em 0 0.4em;
    color: var(--sd-text);
  }

  .prose :global(h4),
  .prose :global(h5),
  .prose :global(h6) {
    font-size: 1.1em;
    font-weight: 600;
    margin: 1em 0 0.3em;
    color: var(--sd-text);
  }

  .prose :global(p) {
    margin: 0 0 1em;
  }

  .prose :global(a) {
    color: var(--sd-accent);
    text-decoration: none;
  }

  .prose :global(a:hover) {
    text-decoration: underline;
  }

  .prose :global(strong) {
    font-weight: 600;
    color: var(--sd-text);
  }

  .prose :global(code) {
    font-family: var(--sd-font-mono);
    font-size: 0.875em;
    padding: 2px 6px;
    border-radius: var(--sd-radius-sm);
    background: var(--sd-bg-elevated);
    color: var(--sd-danger);
  }

  .prose :global(pre) {
    margin: 0 0 1em;
    padding: 16px;
    border-radius: var(--sd-radius);
    background: var(--sd-bg-elevated);
    border: 1px solid var(--sd-border);
    overflow-x: auto;
    font-size: 13px;
    line-height: 1.5;
  }

  .prose :global(pre code) {
    padding: 0;
    background: none;
    color: var(--sd-text);
    font-size: inherit;
  }

  .prose :global(blockquote) {
    margin: 0 0 1em;
    padding: 4px 16px;
    border-left: 3px solid var(--sd-accent);
    color: var(--sd-text-muted);
    background: rgba(137, 180, 250, 0.04);
    border-radius: 0 var(--sd-radius-sm) var(--sd-radius-sm) 0;
  }

  .prose :global(ul),
  .prose :global(ol) {
    margin: 0 0 1em;
    padding-left: 1.5em;
  }

  .prose :global(li) {
    margin: 0.3em 0;
  }

  .prose :global(hr) {
    margin: 2em 0;
    border: none;
    border-top: 1px solid var(--sd-border);
  }

  .prose :global(table) {
    width: 100%;
    margin: 0 0 1em;
    border-collapse: collapse;
    font-size: 14px;
  }

  .prose :global(th),
  .prose :global(td) {
    padding: 8px 12px;
    border: 1px solid var(--sd-border);
    text-align: left;
  }

  .prose :global(th) {
    background: var(--sd-bg-elevated);
    font-weight: 600;
  }

  .prose :global(img) {
    max-width: 100%;
    border-radius: var(--sd-radius);
  }

  .prose :global(mark) {
    background: rgba(249, 226, 175, 0.2);
    color: var(--sd-warning);
    padding: 1px 4px;
    border-radius: 2px;
  }

  .prose :global(kbd) {
    font-family: var(--sd-font-mono);
    font-size: 0.85em;
    padding: 2px 6px;
    border: 1px solid var(--sd-border);
    border-radius: var(--sd-radius-sm);
    background: var(--sd-bg-elevated);
    box-shadow: 0 1px 0 var(--sd-border);
  }

  .prose :global(details) {
    margin: 0 0 1em;
    border: 1px solid var(--sd-border);
    border-radius: var(--sd-radius);
    padding: 8px 12px;
  }

  .prose :global(summary) {
    cursor: pointer;
    font-weight: 600;
  }

  .prose :global(.footnotes) {
    margin-top: 2em;
    padding-top: 1em;
    border-top: 1px solid var(--sd-border);
    font-size: 0.875em;
    color: var(--sd-text-muted);
  }
</style>
