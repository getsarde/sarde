<script>
  import { Rocket } from 'lucide-svelte'

  let { projectName, location, fullPath, template, description, author, creating = false, error = '', oncreate } = $props()

  const templateLabels = { empty: 'Empty', blog: 'Blog', docs: 'Docs' }
</script>

<div class="review">
  <div class="review-row">
    <span class="review-label">Name</span>
    <span class="review-value">{projectName}</span>
  </div>
  <div class="review-row">
    <span class="review-label">Location</span>
    <span class="review-value"><code>{fullPath}</code></span>
  </div>
  <div class="review-row">
    <span class="review-label">Template</span>
    <span class="review-value">{templateLabels[template] ?? template}</span>
  </div>
  {#if description}
    <div class="review-row">
      <span class="review-label">Description</span>
      <span class="review-value">{description}</span>
    </div>
  {/if}
  {#if author}
    <div class="review-row">
      <span class="review-label">Author</span>
      <span class="review-value">{author}</span>
    </div>
  {/if}
</div>

{#if error}
  <p class="error-msg">{error}</p>
{/if}

<button class="create-btn" onclick={oncreate} disabled={creating}>
  {#if creating}
    Creating...
  {:else}
    <Rocket size={16} /> Create Project
  {/if}
</button>

<style>
  .review {
    border: 1px solid var(--sd-border);
    border-radius: var(--sd-radius-lg);
    background: var(--sd-bg-input);
    padding: 4px 0;
    margin-bottom: 18px;
  }

  .review-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 14px;
    font-size: 13px;
  }

  .review-row + .review-row {
    border-top: 1px solid var(--sd-border);
  }

  .review-label {
    color: var(--sd-text-muted);
    font-weight: 500;
  }

  .review-value {
    color: var(--sd-text);
    text-align: right;
    max-width: 60%;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .review-value code {
    font-size: 11px;
    color: var(--sd-accent);
    word-break: break-all;
    white-space: normal;
  }

  .error-msg {
    font-size: 13px;
    color: var(--sd-danger);
    margin: 0 0 12px;
    padding: 8px 10px;
    background: var(--sd-danger-bg);
    border-radius: var(--sd-radius);
  }

  .create-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    width: 100%;
    padding: 12px;
    border: none;
    border-radius: var(--sd-radius-lg);
    background: var(--sd-accent);
    color: var(--sd-bg-base);
    font-size: 15px;
    font-weight: 700;
    font-family: inherit;
    cursor: pointer;
    transition: background 0.15s, transform 0.1s;
  }

  .create-btn:hover:not(:disabled) {
    background: var(--sd-accent-hover);
    transform: translateY(-1px);
  }

  .create-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
