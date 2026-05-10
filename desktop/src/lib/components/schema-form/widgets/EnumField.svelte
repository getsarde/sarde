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
    color: var(--cr-text);
    background: var(--cr-bg-input);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
    outline: none;
    font-family: inherit;
    cursor: pointer;
  }

  .field-input:focus {
    border-color: var(--cr-accent);
  }
</style>
