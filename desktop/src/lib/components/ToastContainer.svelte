<script>
  import { Toaster } from 'svelte-sonner'
</script>

<Toaster
  position="bottom-right"
  offset={16}
  gap={8}
  toastOptions={{
    unstyled: true,
    classes: {
      toast: 'cr-toast',
      info: 'cr-toast-info',
      success: 'cr-toast-success',
      warning: 'cr-toast-warning',
      error: 'cr-toast-error',
    },
  }}
/>

<style>
  :global([data-sonner-toaster]) {
    z-index: 300 !important;
    font-family: var(--cr-font-ui) !important;
  }

  :global(.cr-toast) {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 10px 14px;
    border-radius: var(--cr-radius);
    background: var(--cr-bg-base);
    border: 1px solid var(--cr-border);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
    font-size: 13px;
    color: var(--cr-text);
    max-width: 380px;
  }

  :global(.cr-toast-info) {
    border-left: 3px solid var(--cr-info);
  }

  :global(.cr-toast-success) {
    border-left: 3px solid var(--cr-success);
  }

  :global(.cr-toast-warning) {
    border-left: 3px solid var(--cr-warning);
  }

  :global(.cr-toast-error) {
    border-left: 3px solid var(--cr-danger);
  }
</style>
