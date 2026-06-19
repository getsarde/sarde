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
  class="sd-switch"
  {id}
  {checked}
  {onCheckedChange}
  {disabled}
  aria-label={ariaLabel}
>
  <Switch.Thumb class="sd-switch-thumb" />
</Switch.Root>

