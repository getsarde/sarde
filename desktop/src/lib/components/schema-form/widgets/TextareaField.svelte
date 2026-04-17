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
    onchange={(e) => onchange(e.target.value)}
  ></textarea>
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
    resize: vertical;
    min-height: 48px;
  }

  .field-input:focus {
    border-color: var(--color-accent, #89b4fa);
  }
</style>
