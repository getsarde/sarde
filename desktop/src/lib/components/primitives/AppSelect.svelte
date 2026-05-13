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
  <Select.Trigger class="cr-select-trigger" {id} {disabled}>
    <span class="cr-select-value">{value || placeholder}</span>
    <ChevronDown size={13} />
  </Select.Trigger>
  <Select.Portal>
    <Select.Content class="cr-select-content" sideOffset={4} position="popper">
      <Select.Viewport>
        {#each options as opt}
          <Select.Item class="cr-select-item" value={opt} label={opt}>
            <span class="cr-select-item-text">{opt}</span>
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
  :global(.cr-select-trigger) {
    display: flex;
    align-items: center;
    justify-content: space-between;
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

  :global(.cr-select-trigger:focus) {
    border-color: var(--cr-accent);
  }

  :global(.cr-select-trigger[data-disabled]) {
    opacity: 0.5;
    cursor: not-allowed;
  }

  :global(.cr-select-value) {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :global(.cr-select-content) {
    background: var(--cr-bg-elevated);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
    box-shadow: var(--cr-shadow-sm);
    z-index: 400;
    max-height: 200px;
    overflow-y: auto;
    min-width: var(--bits-select-trigger-width);
  }

  :global(.cr-select-item) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 5px 8px;
    font-size: 12px;
    color: var(--cr-text);
    cursor: pointer;
    outline: none;
  }

  :global(.cr-select-item[data-highlighted]) {
    background: var(--cr-hover);
  }

  :global(.cr-select-item[data-selected]) {
    color: var(--cr-accent);
  }

  :global(.cr-select-item-text) {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
