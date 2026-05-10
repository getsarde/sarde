<script>
  import yaml from 'js-yaml'
  import { doc, tabs, ui } from '../stores/app.svelte.js'
  import { fetchSchema } from '../api.js'
  import SchemaForm from './schema-form/SchemaForm.svelte'
  import YamlEditor from './YamlEditor.svelte'

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

  // Raw YAML text (between --- delimiters, without delimiters)
  let rawYaml = $derived.by(() => {
    if (!doc.content) return ''
    const match = doc.content.match(/^---\r?\n([\s\S]*?)\r?\n---/)
    return match ? match[1] : ''
  })

  let yamlError = $state(null)

  // Derive collection from file path (first segment: "blog/my-post.md" → "blog")
  let collection = $derived.by(() => {
    if (!doc.contentPath) return ''
    const parts = doc.contentPath.replace(/\\/g, '/').split('/')
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
      schema = resp?.fields ?? null
    }).catch(() => { schema = null })
  })

  // Write from form: update a single field
  function updateField(key, value) {
    if (!parsed.fm) return
    const newFm = { ...parsed.fm, [key]: value }
    const newYaml = yaml.dump(newFm, { lineWidth: -1, noRefs: true, quotingType: '"', forceQuotes: false })
    const body = parsed.bodyStart >= 0 ? doc.content.slice(parsed.bodyStart) : ''
    doc.content = `---\n${newYaml}---${body}`
    doc.dirty = true
    doc.externalUpdate++

    const activeTab = tabs.items.find(t => t.id === tabs.activeId)
    if (activeTab) {
      activeTab.cachedContent = doc.content
      activeTab.dirty = true
    }
  }

  // Write from YAML editor: replace entire frontmatter
  function handleYamlChange(newYaml) {
    const body = parsed.bodyStart >= 0 ? doc.content.slice(parsed.bodyStart) : ''
    doc.content = `---\n${newYaml}\n---${body}`
    doc.dirty = true
    doc.externalUpdate++

    const activeTab = tabs.items.find(t => t.id === tabs.activeId)
    if (activeTab) {
      activeTab.cachedContent = doc.content
      activeTab.dirty = true
    }
  }
</script>

{#if !doc.filePath}
  <p class="empty-msg">Open a file to edit properties.</p>
{:else if ui.propertiesMode === 'yaml'}
  <div class="yaml-editor-wrap">
    <YamlEditor
      value={rawYaml}
      onchange={handleYamlChange}
      onerror={(e) => { yamlError = e }}
    />
    {#if yamlError}
      <div class="yaml-error">{yamlError}</div>
    {/if}
  </div>
{:else if !parsed.fm}
  <p class="empty-msg">No frontmatter found.</p>
{:else}
  <div class="properties-form">
    <SchemaForm
      fields={schema ?? {}}
      values={parsed.fm}
      onchange={updateField}
    />
  </div>
{/if}

<style>
  .empty-msg {
    padding: 12px;
    margin: 0;
    font-size: 12px;
    color: var(--cr-text-muted);
  }

  .properties-form {
    padding: 4px 12px;
  }

  .yaml-editor-wrap {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 4px;
  }

  .yaml-error {
    padding: 6px 8px;
    font-size: 11px;
    color: var(--cr-danger);
    background: rgba(243, 139, 168, 0.1);
    border-top: 1px solid var(--cr-border);
    white-space: pre-wrap;
    word-break: break-word;
  }
</style>
