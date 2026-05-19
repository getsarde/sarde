<script>
  import { Tooltip } from 'bits-ui'
  import './tooltip.css'

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

