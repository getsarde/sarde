<script>
  import FieldWrapper from './FieldWrapper.svelte'

  let { name, label = name, value = [], required = false, error = '', disabled = false, onchange } = $props()

  let tags = $derived(Array.isArray(value) ? value : [])

  function handleKeydown(e) {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      const val = e.target.value.trim().replace(/,$/, '')
      if (val && !tags.includes(val)) {
        onchange([...tags, val])
      }
      e.target.value = ''
    }
  }

  function removeTag(index) {
    onchange(tags.filter((_, i) => i !== index))
  }
</script>

<FieldWrapper {name} {label} {required} {error}>
  <div class="tag-input">
    {#each tags as tag, i}
      <span class="tag">
        {tag}
        {#if !disabled}
          <button class="tag-remove" onclick={() => removeTag(i)}>&times;</button>
        {/if}
      </span>
    {/each}
    {#if !disabled}
      <input
        class="tag-add"
        type="text"
        placeholder="Add..."
        onkeydown={handleKeydown}
      />
    {/if}
  </div>
</FieldWrapper>

<style>
  .tag-input {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    padding: 4px 6px;
    background: var(--cr-bg-input);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
    min-height: 28px;
    align-items: center;
  }

  .tag-input:focus-within {
    border-color: var(--cr-accent);
  }

  .tag {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    padding: 1px 6px;
    font-size: 11px;
    color: var(--cr-text);
    background: var(--cr-hover);
    border-radius: 3px;
    white-space: nowrap;
  }

  .tag-remove {
    border: none;
    background: none;
    color: var(--cr-text-muted);
    cursor: pointer;
    padding: 0 1px;
    font-size: 13px;
    line-height: 1;
  }

  .tag-remove:hover {
    color: var(--cr-text);
  }

  .tag-add {
    flex: 1;
    min-width: 50px;
    border: none;
    background: transparent;
    color: var(--cr-text);
    font-size: 11px;
    outline: none;
    padding: 2px 0;
  }

  .tag-add::placeholder {
    color: var(--cr-text-muted);
  }
</style>
