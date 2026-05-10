<script>
  import { toasts, removeToast } from '../stores/app.svelte.js'
</script>

<div class="toast-container" aria-live="polite">
  {#each toasts.items as toast (toast.id)}
    <div class="toast toast-{toast.type}">
      <div class="toast-icon">
        {#if toast.type === 'success'}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20 6 9 17l-5-5"/>
          </svg>
        {:else if toast.type === 'error'}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/><path d="m15 9-6 6"/><path d="m9 9 6 6"/>
          </svg>
        {:else if toast.type === 'warning'}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"/>
            <path d="M12 9v4"/><path d="M12 17h.01"/>
          </svg>
        {:else}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>
          </svg>
        {/if}
      </div>
      <span class="toast-message">{toast.message}</span>
      <button class="toast-close" onclick={() => removeToast(toast.id)} title="Dismiss">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M18 6 6 18"/><path d="m6 6 12 12"/>
        </svg>
      </button>
    </div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    bottom: 16px;
    right: 16px;
    z-index: 300;
    display: flex;
    flex-direction: column-reverse;
    gap: 8px;
    pointer-events: none;
    max-width: 380px;
  }

  .toast {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 10px 14px;
    border-radius: var(--cr-radius);
    background: var(--cr-bg-base);
    border: 1px solid var(--cr-border);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
    pointer-events: auto;
    animation: toast-in 0.25s ease-out;
    font-size: 13px;
    color: var(--cr-text);
  }

  @keyframes toast-in {
    from {
      opacity: 0;
      transform: translateY(12px) scale(0.96);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  .toast-icon {
    flex-shrink: 0;
    margin-top: 1px;
  }

  .toast-info .toast-icon {
    color: var(--cr-info);
  }

  .toast-success .toast-icon {
    color: var(--cr-success);
  }

  .toast-warning .toast-icon {
    color: var(--cr-warning);
  }

  .toast-error .toast-icon {
    color: var(--cr-danger);
  }

  .toast-info {
    border-left: 3px solid var(--cr-info);
  }

  .toast-success {
    border-left: 3px solid var(--cr-success);
  }

  .toast-warning {
    border-left: 3px solid var(--cr-warning);
  }

  .toast-error {
    border-left: 3px solid var(--cr-danger);
  }

  .toast-message {
    flex: 1;
    line-height: 1.4;
  }

  .toast-close {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border: none;
    border-radius: var(--cr-radius-sm);
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
    margin: -2px -4px 0 0;
  }

  .toast-close:hover {
    background: var(--cr-hover);
    color: var(--cr-text);
  }
</style>
