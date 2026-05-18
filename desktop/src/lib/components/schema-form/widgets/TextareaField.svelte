<script>
  import FieldWrapper from './FieldWrapper.svelte'

  let { name, label = name, value = '', required = false, error = '', disabled = false, onchange, maxLength } = $props()
</script>

<FieldWrapper {name} {label} {required} {error}>
  <textarea
    class="field-input"
    id={name}
    rows="3"
    {value}
    {disabled}
    maxlength={maxLength}
    oninput={(e) => onchange(e.target.value)}
  ></textarea>
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
    resize: vertical;
    min-height: 48px;
  }

  .field-input:focus {
    border-color: var(--cr-accent);
  }
</style>
