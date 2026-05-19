<script>
  import { Switch } from 'bits-ui'
  import './switch.css'

  let {
    id = '',
    checked = false,
    onCheckedChange,
    disabled = false,
    ariaLabel = '',
  } = $props()
</script>

<Switch.Root
  class="cr-switch"
  {id}
  {checked}
  {onCheckedChange}
  {disabled}
  aria-label={ariaLabel}
>
  <Switch.Thumb class="cr-switch-thumb" />
</Switch.Root>

