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
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-lg);
    background: var(--cr-bg-input);
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
    border-top: 1px solid var(--cr-border);
  }

  .review-label {
    color: var(--cr-text-muted);
    font-weight: 500;
  }

  .review-value {
    color: var(--cr-text);
    text-align: right;
    max-width: 60%;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .review-value code {
    font-size: 11px;
    color: var(--cr-accent);
    word-break: break-all;
    white-space: normal;
  }

  .error-msg {
    font-size: 13px;
    color: var(--cr-danger);
    margin: 0 0 12px;
    padding: 8px 10px;
    background: rgba(243, 139, 168, 0.1);
    border-radius: var(--cr-radius);
  }

  .create-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    width: 100%;
    padding: 12px;
    border: none;
    border-radius: var(--cr-radius-lg);
    background: var(--cr-accent);
    color: var(--cr-bg-base);
    font-size: 15px;
    font-weight: 700;
    font-family: inherit;
    cursor: pointer;
    transition: background 0.15s, transform 0.1s;
  }

  .create-btn:hover:not(:disabled) {
    background: var(--cr-accent-hover);
    transform: translateY(-1px);
  }

  .create-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
