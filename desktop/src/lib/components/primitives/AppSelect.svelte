<script>
  import { Select } from 'bits-ui'
  import { ChevronDown, Check } from 'lucide-svelte'

  let {
    id = '',
    value = '',
    onValueChange,
    options = [],
    disabled = false,
    placeholder = 'Select...',
  } = $props()

  function handleChange(v) {
    if (v !== undefined) onValueChange?.(v)
  }
</script>

<Select.Root type="single" {value} onValueChange={handleChange}>
  <Select.Trigger class="sd-select-trigger" {id} {disabled}>
    <span class="sd-select-value">{value || placeholder}</span>
    <ChevronDown size={13} />
  </Select.Trigger>
  <Select.Portal>
    <Select.Content class="sd-select-content" sideOffset={4} position="popper">
      <Select.Viewport>
        {#each options as opt}
          <Select.Item class="sd-select-item" value={opt} label={opt}>
            <span class="sd-select-item-text">{opt}</span>
            {#if opt === value}
              <Check size={12} />
            {/if}
          </Select.Item>
        {/each}
      </Select.Viewport>
    </Select.Content>
  </Select.Portal>
</Select.Root>

<style>
  :global(.sd-select-trigger) {
    display: flex;
    align-items: center;
    justify-content: space-between;
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
    cursor: pointer;
  }

  :global(.sd-select-trigger:focus) {
    border-color: var(--sd-accent);
  }

  :global(.sd-select-trigger[data-disabled]) {
    opacity: 0.5;
    cursor: not-allowed;
  }

  :global(.sd-select-value) {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :global(.sd-select-content) {
    background: var(--sd-bg-elevated);
    border: 1px solid var(--sd-border);
    border-radius: var(--sd-radius-sm);
    box-shadow: var(--sd-shadow-sm);
    z-index: 400;
    max-height: 200px;
    overflow-y: auto;
    min-width: var(--bits-select-trigger-width);
  }

  :global(.sd-select-item) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 5px 8px;
    font-size: 12px;
    color: var(--sd-text);
    cursor: pointer;
    outline: none;
  }

  :global(.sd-select-item[data-highlighted]) {
    background: var(--sd-hover);
  }

  :global(.sd-select-item[data-selected]) {
    color: var(--sd-accent);
  }

  :global(.sd-select-item-text) {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
