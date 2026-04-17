<script>
  let { name, label, required = false, error = '', children } = $props()
</script>

<div class="field-group">
  <label class="field-label" for={name}>
    {label}
    {#if required}<span class="required">*</span>{/if}
  </label>
  {@render children()}
  {#if error}<span class="field-error">{error}</span>{/if}
</div>

<style>
  .field-group {
    margin-bottom: 12px;
  }

  .field-label {
    display: block;
    font-size: 11px;
    font-weight: 600;
    color: var(--color-text-muted, #6c7086);
    margin-bottom: 4px;
    text-transform: capitalize;
  }

  .required {
    color: var(--color-error, #f38ba8);
    margin-left: 2px;
  }

  .field-error {
    display: block;
    font-size: 11px;
    color: var(--color-error, #f38ba8);
    margin-top: 3px;
  }
</style>
