<script>
  import FieldWrapper from './FieldWrapper.svelte'

  let { name, label = name, value = '', required = false, error = '', disabled = false, onchange, min, max } = $props()
</script>

<FieldWrapper {name} {label} {required} {error}>
  <input
    class="field-input"
    id={name}
    type="number"
    step="1"
    {min}
    {max}
    {value}
    {disabled}
    onchange={(e) => onchange(parseInt(e.target.value) || 0)}
  />
</FieldWrapper>

<style>
  .field-input {
    width: 100%;
    box-sizing: border-box;
    padding: 5px 8px;
    font-size: 12px;
    color: var(--color-text, #cdd6f4);
    background: var(--color-surface-alt, #181825);
    border: 1px solid var(--color-border, #313244);
    border-radius: 4px;
    outline: none;
    font-family: inherit;
  }

  .field-input:focus {
    border-color: var(--color-accent, #89b4fa);
  }
</style>
