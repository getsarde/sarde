<script>
  import { mdPreview } from '../stores/app.svelte.js'
  import { Loader } from 'lucide-svelte'
</script>

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

<style>
  .preview-container {
    flex: 1;
    overflow-y: auto;
    padding: 24px 32px;
    background: var(--cr-bg-base);
    color: var(--cr-text);
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
    color: var(--cr-text-muted);
    font-size: 13px;
  }

  .preview-error {
    padding: 16px;
    color: var(--cr-danger);
    font-size: 13px;
  }

  /* Prose styles for rendered markdown */
  .prose :global(h1) {
    font-size: 2em;
    font-weight: 700;
    margin: 0 0 0.5em;
    padding-bottom: 0.3em;
    border-bottom: 1px solid var(--cr-border);
    color: var(--cr-text);
  }

  .prose :global(h2) {
    font-size: 1.5em;
    font-weight: 600;
    margin: 1.5em 0 0.5em;
    padding-bottom: 0.2em;
    border-bottom: 1px solid var(--cr-border);
    color: var(--cr-text);
  }

  .prose :global(h3) {
    font-size: 1.25em;
    font-weight: 600;
    margin: 1.2em 0 0.4em;
    color: var(--cr-text);
  }

  .prose :global(h4),
  .prose :global(h5),
  .prose :global(h6) {
    font-size: 1.1em;
    font-weight: 600;
    margin: 1em 0 0.3em;
    color: var(--cr-text);
  }

  .prose :global(p) {
    margin: 0 0 1em;
  }

  .prose :global(a) {
    color: var(--cr-accent);
    text-decoration: none;
  }

  .prose :global(a:hover) {
    text-decoration: underline;
  }

  .prose :global(strong) {
    font-weight: 600;
    color: var(--cr-text);
  }

  .prose :global(code) {
    font-family: var(--cr-font-mono);
    font-size: 0.875em;
    padding: 2px 6px;
    border-radius: var(--cr-radius-sm);
    background: var(--cr-bg-elevated);
    color: var(--cr-danger);
  }

  .prose :global(pre) {
    margin: 0 0 1em;
    padding: 16px;
    border-radius: var(--cr-radius);
    background: var(--cr-bg-elevated);
    border: 1px solid var(--cr-border);
    overflow-x: auto;
    font-size: 13px;
    line-height: 1.5;
  }

  .prose :global(pre code) {
    padding: 0;
    background: none;
    color: var(--cr-text);
    font-size: inherit;
  }

  .prose :global(blockquote) {
    margin: 0 0 1em;
    padding: 4px 16px;
    border-left: 3px solid var(--cr-accent);
    color: var(--cr-text-muted);
    background: rgba(137, 180, 250, 0.04);
    border-radius: 0 var(--cr-radius-sm) var(--cr-radius-sm) 0;
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
    border-top: 1px solid var(--cr-border);
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
    border: 1px solid var(--cr-border);
    text-align: left;
  }

  .prose :global(th) {
    background: var(--cr-bg-elevated);
    font-weight: 600;
  }

  .prose :global(img) {
    max-width: 100%;
    border-radius: var(--cr-radius);
  }

  .prose :global(mark) {
    background: rgba(249, 226, 175, 0.2);
    color: var(--cr-warning);
    padding: 1px 4px;
    border-radius: 2px;
  }

  .prose :global(kbd) {
    font-family: var(--cr-font-mono);
    font-size: 0.85em;
    padding: 2px 6px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
    background: var(--cr-bg-elevated);
    box-shadow: 0 1px 0 var(--cr-border);
  }

  .prose :global(details) {
    margin: 0 0 1em;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    padding: 8px 12px;
  }

  .prose :global(summary) {
    cursor: pointer;
    font-weight: 600;
  }

  .prose :global(.footnotes) {
    margin-top: 2em;
    padding-top: 1em;
    border-top: 1px solid var(--cr-border);
    font-size: 0.875em;
    color: var(--cr-text-muted);
  }
</style>
