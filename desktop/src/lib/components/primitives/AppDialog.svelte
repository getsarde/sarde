<script>
  import { Dialog } from 'bits-ui'

  let {
    open = false,
    onOpenChange,
    ariaLabel = '',
    width = '400px',
    onEscapeKeydown,
    onInteractOutside,
    children,
  } = $props()
</script>

<Dialog.Root {open} {onOpenChange}>
  <Dialog.Portal>
    <Dialog.Overlay class="cr-dialog-overlay" />
    <Dialog.Content
      class="cr-dialog-content"
      aria-label={ariaLabel}
      style="--dialog-width: {width}"
      {onEscapeKeydown}
      {onInteractOutside}
    >
      {@render children()}
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>

<style>
  :global(.cr-dialog-overlay) {
    position: fixed;
    inset: 0;
    z-index: 100;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
  }

  :global(.cr-dialog-content) {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 100;
    width: var(--dialog-width);
    max-width: 90vw;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    background: var(--cr-bg-base);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-lg);
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.4);
    overflow: hidden;
    outline: none;
  }
</style>
