<script>
  import FieldWrapper from './FieldWrapper.svelte'
  import AppSwitch from '../../primitives/AppSwitch.svelte'

  let { name, label = name, value = false, required = false, error = '', disabled = false, onchange } = $props()
</script>

<FieldWrapper {name} {label} {required} {error}>
  <AppSwitch
    id={name}
    checked={!!value}
    onCheckedChange={(v) => onchange(v)}
    {disabled}
    ariaLabel={label}
  />
</FieldWrapper>
