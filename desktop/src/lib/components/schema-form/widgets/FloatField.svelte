<script>
  import FieldWrapper from './FieldWrapper.svelte'

  let { name, label = name, value = '', required = false, error = '', disabled = false, onchange, min, max } = $props()
</script>

<FieldWrapper {name} {label} {required} {error}>
  <input
    class="field-input"
    id={name}
    type="number"
    step="any"
    {min}
    {max}
    {value}
    {disabled}
    onchange={(e) => onchange(parseFloat(e.target.value) || 0)}
  />
</FieldWrapper>

<style>
  .field-input {
    width: 100%;
    box-sizing: border-box;
    padding: 5px 8px;
    font-size: 12px;
    color: var(--sd-text);
    background: var(--sd-bg-input);
    border: 1px solid var(--sd-border);
    border-radius: var(--sd-radius-sm);
    outline: none;
    font-family: inherit;
  }

  .field-input:focus {
    border-color: var(--sd-accent);
  }
</style>
