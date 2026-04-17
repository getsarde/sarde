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
    border: 1px solid var(--color-border, #313244);
    border-radius: 10px;
    background: var(--color-input, #11111b);
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
    border-top: 1px solid var(--color-border, #313244);
  }

  .review-label {
    color: var(--color-text-muted, #6c7086);
    font-weight: 500;
  }

  .review-value {
    color: var(--color-text, #cdd6f4);
    text-align: right;
    max-width: 60%;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .review-value code {
    font-size: 11px;
    color: var(--color-accent, #89b4fa);
    word-break: break-all;
    white-space: normal;
  }

  .error-msg {
    font-size: 13px;
    color: var(--color-danger, #f38ba8);
    margin: 0 0 12px;
    padding: 8px 10px;
    background: rgba(243, 139, 168, 0.1);
    border-radius: 6px;
  }

  .create-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    width: 100%;
    padding: 12px;
    border: none;
    border-radius: 10px;
    background: var(--color-accent, #89b4fa);
    color: var(--color-surface, #1e1e2e);
    font-size: 15px;
    font-weight: 700;
    font-family: inherit;
    cursor: pointer;
    transition: background 0.15s, transform 0.1s;
  }

  .create-btn:hover:not(:disabled) {
    background: #74c7ec;
    transform: translateY(-1px);
  }

  .create-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
