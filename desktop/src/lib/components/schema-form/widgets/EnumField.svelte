<script>
  import FieldWrapper from './FieldWrapper.svelte'

  let { name, label = name, value = '', required = false, error = '', disabled = false, onchange, options = [] } = $props()
</script>

<FieldWrapper {name} {label} {required} {error}>
  <select
    class="field-input"
    id={name}
    {value}
    {disabled}
    onchange={(e) => onchange(e.target.value)}
  >
    {#each options as opt}
      <option value={opt}>{opt}</option>
    {/each}
  </select>
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
    cursor: pointer;
  }

  .field-input:focus {
    border-color: var(--color-accent, #89b4fa);
  }
</style>
