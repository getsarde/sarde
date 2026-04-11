<script>
  import yaml from 'js-yaml'
  import { doc, tabs } from '../stores/app.svelte.js'
  import { fetchSchema } from '../api.js'

  // Parse frontmatter from doc.content reactively
  let parsed = $derived.by(() => {
    if (!doc.content) return { fm: null, bodyStart: -1 }
    const match = doc.content.match(/^---\r?\n([\s\S]*?)\r?\n---/)
    if (!match) return { fm: null, bodyStart: -1 }
    try {
      const fm = yaml.load(match[1])
      if (!fm || typeof fm !== 'object') return { fm: null, bodyStart: -1 }
      return { fm, bodyStart: match[0].length }
    } catch {
      return { fm: null, bodyStart: -1 }
    }
  })

  // Derive collection from file path (first segment: "blog/my-post.md" → "blog")
  let collection = $derived.by(() => {
    if (!doc.filePath) return ''
    const parts = doc.filePath.replace(/\\/g, '/').split('/')
    return parts.length > 1 ? parts[0] : ''
  })

  // Fetch schema when collection changes
  let schema = $state(null)
  let lastFetchedCol = ''

  $effect(() => {
    const col = collection
    if (!col) { schema = null; lastFetchedCol = ''; return }
    if (col === lastFetchedCol) return
    lastFetchedCol = col
    fetchSchema(col).then(resp => {
      schema = resp?.data?.fields ?? null
    }).catch(() => { schema = null })
  })

  // Field ordering: title, draft, date first, then alphabetical
  const PRIORITY = { title: 0, draft: 1, date: 2 }

  let sortedKeys = $derived.by(() => {
    if (!parsed.fm) return []
    return Object.keys(parsed.fm).sort((a, b) => {
      const pa = PRIORITY[a] ?? 999
      const pb = PRIORITY[b] ?? 999
      if (pa !== pb) return pa - pb
      return a.localeCompare(b)
    })
  })

  // Type inference when no schema
  function inferType(key, value) {
    if (typeof value === 'boolean') return 'bool'
    if (typeof value === 'number') return Number.isInteger(value) ? 'int' : 'float'
    if (Array.isArray(value)) return 'list'
    if (typeof value === 'string') {
      if (/^\d{4}-\d{2}-\d{2}(T|\s)/.test(value)) return 'date'
      if (value.length > 80) return 'textarea'
      return 'string'
    }
    return 'string'
  }

  function getFieldType(key) {
    if (schema?.[key]?.type) return schema[key].type
    return inferType(key, parsed.fm[key])
  }

  function getLabel(key) {
    return schema?.[key]?.label || key
  }

  function getOptions(key) {
    return schema?.[key]?.options ?? []
  }

  function getMin(key) {
    return schema?.[key]?.min ?? undefined
  }

  function getMax(key) {
    return schema?.[key]?.max ?? undefined
  }

  // Date helpers
  function toDateInput(val) {
    if (!val || typeof val !== 'string') return ''
    return val.slice(0, 10)
  }

  function toRFC3339(dateStr) {
    if (!dateStr) return ''
    return dateStr + 'T00:00:00Z'
  }

  // Write: update a field and sync to doc.content + editor
  function updateField(key, value) {
    if (!parsed.fm) return
    const newFm = { ...parsed.fm, [key]: value }
    const newYaml = yaml.dump(newFm, { lineWidth: -1, noRefs: true, quotingType: '"', forceQuotes: false })
    const body = parsed.bodyStart >= 0 ? doc.content.slice(parsed.bodyStart) : ''
    doc.content = `---\n${newYaml}---${body}`
    doc.dirty = true
    doc.externalUpdate++

    // Sync to active tab cache
    const activeTab = tabs.items.find(t => t.id === tabs.activeId)
    if (activeTab) {
      activeTab.cachedContent = doc.content
      activeTab.dirty = true
    }
  }

  // Tag input: add on Enter/comma
  function handleTagKeydown(e, key, currentTags) {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      const val = e.target.value.trim().replace(/,$/, '')
      if (val && !currentTags.includes(val)) {
        updateField(key, [...currentTags, val])
      }
      e.target.value = ''
    }
  }

  function removeTag(key, currentTags, index) {
    updateField(key, currentTags.filter((_, i) => i !== index))
  }
</script>

{#if !doc.filePath}
  <p class="empty-msg">Open a file to edit properties.</p>
{:else if !parsed.fm}
  <p class="empty-msg">No frontmatter found.</p>
{:else}
  <div class="properties-form">
    {#each sortedKeys as key (key)}
      {@const fieldType = getFieldType(key)}
      {@const label = getLabel(key)}
      {@const value = parsed.fm[key]}

      <div class="field-group">
        <label class="field-label">{label}</label>

        {#if fieldType === 'bool' || fieldType === 'toggle'}
          <button
            class="toggle"
            class:active={!!value}
            role="switch"
            aria-checked={!!value}
            onclick={() => updateField(key, !value)}
          >
            <span class="toggle-knob"></span>
          </button>

        {:else if fieldType === 'date'}
          <input
            class="field-input"
            type="date"
            value={toDateInput(value)}
            onchange={(e) => updateField(key, toRFC3339(e.target.value))}
          />

        {:else if fieldType === 'enum' || fieldType === 'select'}
          <select
            class="field-input"
            value={value ?? ''}
            onchange={(e) => updateField(key, e.target.value)}
          >
            {#each getOptions(key) as opt}
              <option value={opt}>{opt}</option>
            {/each}
          </select>

        {:else if fieldType === 'list' || fieldType === 'tags'}
          <div class="tag-input">
            {#if Array.isArray(value)}
              {#each value as tag, i}
                <span class="tag">
                  {tag}
                  <button class="tag-remove" onclick={() => removeTag(key, value, i)}>&times;</button>
                </span>
              {/each}
            {/if}
            <input
              class="tag-add"
              type="text"
              placeholder="Add..."
              onkeydown={(e) => handleTagKeydown(e, key, Array.isArray(value) ? value : [])}
            />
          </div>

        {:else if fieldType === 'textarea'}
          <textarea
            class="field-input field-textarea"
            rows="3"
            value={value ?? ''}
            onchange={(e) => updateField(key, e.target.value)}
          ></textarea>

        {:else if fieldType === 'int'}
          <input
            class="field-input"
            type="number"
            step="1"
            min={getMin(key)}
            max={getMax(key)}
            value={value ?? ''}
            onchange={(e) => updateField(key, parseInt(e.target.value) || 0)}
          />

        {:else if fieldType === 'float' || fieldType === 'number'}
          <input
            class="field-input"
            type="number"
            step="any"
            min={getMin(key)}
            max={getMax(key)}
            value={value ?? ''}
            onchange={(e) => updateField(key, parseFloat(e.target.value) || 0)}
          />

        {:else}
          <input
            class="field-input"
            type="text"
            value={value ?? ''}
            onchange={(e) => updateField(key, e.target.value)}
          />
        {/if}
      </div>
    {/each}
  </div>
{/if}

<style>
  .empty-msg {
    padding: 12px;
    margin: 0;
    font-size: 12px;
    color: var(--color-text-muted, #6c7086);
  }

  .properties-form {
    padding: 4px 12px;
  }

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

  .field-input {
    width: 100%;
    box-sizing: border-box;
    padding: 5px 8px;
    font-size: 12px;
    color: var(--color-text, #cdd6f4);
    background: var(--color-surface-alt, #181825);
    border: 1px solid var(--color-border, #313244);
    border-radius: 4px;
    outline: none;
    font-family: inherit;
  }

  .field-input:focus {
    border-color: var(--color-accent, #89b4fa);
  }

  .field-textarea {
    resize: vertical;
    min-height: 48px;
  }

  select.field-input {
    cursor: pointer;
  }

  /* Toggle switch */
  .toggle {
    position: relative;
    width: 36px;
    height: 20px;
    border-radius: 10px;
    border: none;
    background: var(--color-border, #313244);
    cursor: pointer;
    padding: 0;
    transition: background 0.2s;
  }

  .toggle.active {
    background: var(--color-accent, #89b4fa);
  }

  .toggle-knob {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: #fff;
    transition: transform 0.2s;
    pointer-events: none;
  }

  .toggle.active .toggle-knob {
    transform: translateX(16px);
  }

  /* Tag input */
  .tag-input {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    padding: 4px 6px;
    background: var(--color-surface-alt, #181825);
    border: 1px solid var(--color-border, #313244);
    border-radius: 4px;
    min-height: 28px;
    align-items: center;
  }

  .tag-input:focus-within {
    border-color: var(--color-accent, #89b4fa);
  }

  .tag {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    padding: 1px 6px;
    font-size: 11px;
    color: var(--color-text, #cdd6f4);
    background: var(--color-hover, rgba(255, 255, 255, 0.08));
    border-radius: 3px;
    white-space: nowrap;
  }

  .tag-remove {
    border: none;
    background: none;
    color: var(--color-text-muted, #6c7086);
    cursor: pointer;
    padding: 0 1px;
    font-size: 13px;
    line-height: 1;
  }

  .tag-remove:hover {
    color: var(--color-text, #cdd6f4);
  }

  .tag-add {
    flex: 1;
    min-width: 50px;
    border: none;
    background: transparent;
    color: var(--color-text, #cdd6f4);
    font-size: 11px;
    outline: none;
    padding: 2px 0;
  }

  .tag-add::placeholder {
    color: var(--color-text-muted, #6c7086);
  }
</style>
