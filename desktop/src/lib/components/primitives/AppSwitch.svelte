<script>
  import { Switch } from 'bits-ui'

  let {
    id = '',
    checked = false,
    onCheckedChange,
    disabled = false,
    ariaLabel = '',
  } = $props()
</script>

<Switch.Root
  class="cr-switch"
  {id}
  {checked}
  {onCheckedChange}
  {disabled}
  aria-label={ariaLabel}
>
  <Switch.Thumb class="cr-switch-thumb" />
</Switch.Root>

<style>
  :global(.cr-switch) {
    position: relative;
    display: inline-flex;
    align-items: center;
    width: 36px;
    height: 20px;
    border-radius: 10px;
    border: none;
    background: var(--cr-border);
    cursor: pointer;
    padding: 0;
    transition: background 0.2s;
  }

  :global(.cr-switch:focus-visible) {
    outline: 2px solid var(--cr-accent);
    outline-offset: 2px;
  }

  :global(.cr-switch[data-disabled]) {
    opacity: 0.5;
    cursor: not-allowed;
  }

  :global(.cr-switch[data-state="checked"]) {
    background: var(--cr-accent);
  }

  :global(.cr-switch-thumb) {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: var(--cr-bg-base);
    transition: transform 0.2s;
    pointer-events: none;
  }

  :global(.cr-switch[data-state="checked"] .cr-switch-thumb) {
    transform: translateX(16px);
  }
</style>
