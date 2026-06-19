<script>
  import { Dialog } from 'bits-ui'
  import './dialog.css'

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
    <Dialog.Overlay class="sd-dialog-overlay" />
    <Dialog.Content
      class="sd-dialog-content"
      aria-label={ariaLabel}
      style="--dialog-width: {width}"
      {onEscapeKeydown}
      {onInteractOutside}
    >
      {@render children()}
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>

