<script>
  import FieldWrapper from './FieldWrapper.svelte'

  let { name, label = name, value = '#000000', required = false, error = '', disabled = false, onchange } = $props()
</script>

<FieldWrapper {name} {label} {required} {error}>
  <div class="color-row">
    <input
      class="color-picker"
      type="color"
      value={value || '#000000'}
      {disabled}
      oninput={(e) => onchange(e.target.value)}
    />
    <input
      class="color-text"
      type="text"
      value={value ?? ''}
      {disabled}
      onchange={(e) => onchange(e.target.value)}
    />
  </div>
</FieldWrapper>

<style>
  .color-row {
    display: flex;
    gap: 6px;
    align-items: center;
  }

  .color-picker {
    width: 28px;
    height: 28px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
    padding: 1px;
    background: var(--cr-bg-input);
    cursor: pointer;
  }

  .color-text {
    flex: 1;
    box-sizing: border-box;
    padding: 5px 8px;
    font-size: 12px;
    color: var(--cr-text);
    background: var(--cr-bg-input);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
    outline: none;
    font-family: var(--cr-font-mono);
  }

  .color-text:focus {
    border-color: var(--cr-accent);
  }
</style>
