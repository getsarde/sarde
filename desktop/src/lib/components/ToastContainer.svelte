<script>
  import { Toaster } from 'svelte-sonner'
  import './toast.css'
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

