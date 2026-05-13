<script>
  import { Tooltip } from 'bits-ui'

  let {
    content = '',
    delayDuration = 600,
    children,
  } = $props()
</script>

<Tooltip.Provider {delayDuration} skipDelayDuration={300}>
  <Tooltip.Root>
    <Tooltip.Trigger class="cr-tooltip-trigger">
      {@render children()}
    </Tooltip.Trigger>
    <Tooltip.Portal>
      <Tooltip.Content class="cr-tooltip-content" sideOffset={5}>
        {content}
      </Tooltip.Content>
    </Tooltip.Portal>
  </Tooltip.Root>
</Tooltip.Provider>

<style>
  :global(.cr-tooltip-trigger) {
    display: contents;
  }

  :global(.cr-tooltip-content) {
    padding: 5px 9px;
    background: var(--cr-bg-elevated);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
    font-size: 12px;
    color: var(--cr-text);
    box-shadow: var(--cr-shadow-sm);
    z-index: 400;
    pointer-events: none;
    white-space: nowrap;
    font-family: var(--cr-font-ui);
  }
</style>
