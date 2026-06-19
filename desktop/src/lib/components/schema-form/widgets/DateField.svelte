<script>
  import FieldWrapper from './FieldWrapper.svelte'
  import { toDateInput, toRFC3339 } from '../schema-utils.js'

  let { name, label = name, value = '', required = false, error = '', disabled = false, onchange } = $props()
</script>

<FieldWrapper {name} {label} {required} {error}>
  <input
    class="field-input"
    id={name}
    type="date"
    value={toDateInput(value)}
    {disabled}
    onchange={(e) => onchange(toRFC3339(e.target.value))}
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
