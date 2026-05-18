<script>
  import { Button } from 'bits-ui'

  let {
    variant = 'primary',
    size = 'default',
    fullWidth = false,
    disabled = false,
    onclick,
    children,
    ...rest
  } = $props()

  let cls = $derived(`cr-btn cr-btn-${variant} cr-btn-${size}${fullWidth ? ' cr-btn-full' : ''}`)
</script>

<Button.Root
  class={cls}
  {disabled}
  {onclick}
  {...rest}
>
  {@render children()}
</Button.Root>

<style>
  :global(.cr-btn) {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    border: none;
    border-radius: var(--cr-radius);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
    font-family: inherit;
  }

  :global(.cr-btn:hover:not(:disabled)) {
    opacity: 0.9;
  }

  :global(.cr-btn:disabled) {
    opacity: 0.4;
    cursor: not-allowed;
  }

  :global(.cr-btn:focus-visible) {
    outline: 2px solid var(--cr-accent);
    outline-offset: 2px;
  }

  /* Sizes */
  :global(.cr-btn-default) {
    padding: 8px 20px;
  }

  :global(.cr-btn-sm) {
    padding: 5px 14px;
    font-size: 12px;
  }

  :global(.cr-btn-icon) {
    width: 30px;
    height: 30px;
    padding: 0;
  }

  /* Variants */
  :global(.cr-btn-primary) {
    background: var(--cr-accent);
    color: var(--cr-accent-text, #fff);
  }

  :global(.cr-btn-secondary) {
    background: var(--cr-hover);
    color: var(--cr-text);
  }

  :global(.cr-btn-ghost) {
    background: transparent;
    color: var(--cr-text-muted);
  }

  :global(.cr-btn-ghost:hover:not(:disabled)) {
    background: var(--cr-hover);
    color: var(--cr-text);
    opacity: 1;
  }

  :global(.cr-btn-danger) {
    background: transparent;
    color: var(--cr-danger);
    border: 1px solid var(--cr-danger);
  }

  :global(.cr-btn-danger:hover:not(:disabled)) {
    background: rgba(243, 139, 168, 0.1);
    opacity: 1;
  }

  /* Full width */
  :global(.cr-btn-full) {
    width: 100%;
  }
</style>
