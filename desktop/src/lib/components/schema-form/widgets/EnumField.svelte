<script>
  import FieldWrapper from './FieldWrapper.svelte'
  import AppSelect from '../../primitives/AppSelect.svelte'

  let { name, label = name, value = '', required = false, error = '', disabled = false, onchange, options = [] } = $props()
</script>

<FieldWrapper {name} {label} {required} {error}>
  <AppSelect
    id={name}
    {value}
    onValueChange={(v) => onchange(v)}
    {options}
    {disabled}
  />
</FieldWrapper>
