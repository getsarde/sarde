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
      toast: 'sd-toast',
      info: 'sd-toast-info',
      success: 'sd-toast-success',
      warning: 'sd-toast-warning',
      error: 'sd-toast-error',
    },
  }}
/>

