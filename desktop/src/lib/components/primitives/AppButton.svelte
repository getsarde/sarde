<script>
  import { Button } from 'bits-ui'
  import './button.css'

  let {
    variant = 'primary',
    size = 'default',
    fullWidth = false,
    disabled = false,
    onclick,
    children,
    ...rest
  } = $props()

  let cls = $derived(`cr-btn cr-btn-${variant} cr-btn-${size}${fullWidth ? ' cr-btn-full' : ''}`)
</script>

<Button.Root
  class={cls}
  {disabled}
  {onclick}
  {...rest}
>
  {@render children()}
</Button.Root>

