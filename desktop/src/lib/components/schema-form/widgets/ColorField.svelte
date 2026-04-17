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
    border: 1px solid var(--color-border, #313244);
    border-radius: 4px;
    padding: 1px;
    background: var(--color-surface-alt, #181825);
    cursor: pointer;
  }

  .color-text {
    flex: 1;
    box-sizing: border-box;
    padding: 5px 8px;
    font-size: 12px;
    color: var(--color-text, #cdd6f4);
    background: var(--color-surface-alt, #181825);
    border: 1px solid var(--color-border, #313244);
    border-radius: 4px;
    outline: none;
    font-family: var(--font-mono, monospace);
  }

  .color-text:focus {
    border-color: var(--color-accent, #89b4fa);
  }
</style>
