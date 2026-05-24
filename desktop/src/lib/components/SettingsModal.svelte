<script>
  import { ui, project, siteConfig, isConfigDirty, loadSiteConfig, saveSiteConfig } from '../stores/app.svelte.js'
  import { getCurrentTheme, setTheme, THEMES } from '../stores/theme.svelte.js'
  import { getCollections, createCollection, deleteCollection } from '../api.js'
  import { THEME_PRESETS } from './theme-presets.js'
  import { X, Loader, Code, Plus, Trash2, Monitor } from 'lucide-svelte'
  import YamlEditor from './YamlEditor.svelte'
  import NavigationBuilder from './NavigationBuilder.svelte'
  import AppDialog from './primitives/AppDialog.svelte'
  import AppButton from './primitives/AppButton.svelte'
  import { RadioGroup, Tabs, Toggle, AlertDialog } from 'bits-ui'
  import jsYaml from 'js-yaml'

  const sections = ['general', 'appearance', 'editor', 'navigation', 'collections', 'build', 'deploy', 'about']

  // Raw YAML toggle
  let yamlMode = $state(false)
  let yamlError = $state('')

  let rawYaml = $derived(cfg ? jsYaml.dump(cfg) : '')

  function handleYamlChange(text) {
    try {
      const parsed = jsYaml.load(text)
      if (parsed && typeof parsed === 'object') {
        siteConfig.data = parsed
        yamlError = ''
        saveSiteConfig()
      }
    } catch (e) {
      yamlError = e.message
    }
  }

  function handleYamlError(msg) {
    yamlError = msg
  }

  // Collections tab state
  let collections = $state([])
  let collectionsLoading = $state(false)
  let newCollectionName = $state('')
  let collectionCreating = $state(false)


  async function loadCollections() {
    collectionsLoading = true
    try {
      collections = await getCollections()
    } catch (e) {
      collections = []
    } finally {
      collectionsLoading = false
    }
  }

  async function handleCreateCollection() {
    const name = newCollectionName.trim().toLowerCase().replace(/\s+/g, '-')
    if (!name) return
    collectionCreating = true
    try {
      await createCollection(name)
      newCollectionName = ''
      await loadCollections()
    } catch (e) {
      // Could show error, but keep simple
    } finally {
      collectionCreating = false
    }
  }

  async function handleDeleteCollection(name) {
    try {
      await deleteCollection(name)
      await loadCollections()
    } catch (e) {
      // Could show error
    }
  }

  // Load collections when tab is selected
  $effect(() => {
    if (ui.settingsSection === 'collections' && project.contentPath) {
      loadCollections()
    }
  })

  // Reload config whenever the modal opens or the project changes
  $effect(() => {
    if (ui.settingsOpen && project.contentPath) {
      loadSiteConfig()
    }
    return () => clearTimeout(saveTimer)
  })

  // Unsaved changes warning
  let showDirtyBar = $state(false)

  function close() {
    if (isConfigDirty()) {
      showDirtyBar = true
      return
    }
    ui.settingsOpen = false
  }

  function handleEscapeKeydown(e) {
    if (isConfigDirty()) {
      e.preventDefault()
      showDirtyBar = true
    }
  }

  function handleInteractOutside(e) {
    if (isConfigDirty()) {
      e.preventDefault()
      showDirtyBar = true
    }
  }

  async function saveAndClose() {
    await saveSiteConfig()
    showDirtyBar = false
    ui.settingsOpen = false
  }

  function discardAndClose() {
    loadSiteConfig() // reload from disk
    showDirtyBar = false
    ui.settingsOpen = false
  }

  function capitalize(str) {
    return str.charAt(0).toUpperCase() + str.slice(1)
  }

  // Debounced save for text inputs (saves 800ms after last change)
  let saveTimer = null
  function scheduleSave() {
    clearTimeout(saveTimer)
    saveTimer = setTimeout(saveSiteConfig, 800)
  }

  // Immediate save for toggles and selects
  function immediateSave() {
    clearTimeout(saveTimer)
    saveSiteConfig()
  }

  // ---- Editor prefs (localStorage only — not in sarde.yaml) ----
  let editorFontSize = $state(parseInt(localStorage.getItem('sarde-font-size') || '14'))
  let editorLineNumbers = $state(localStorage.getItem('sarde-line-numbers') !== 'false')
  let editorWordWrap = $state(localStorage.getItem('sarde-word-wrap') === 'true')
  let editorAutoSave = $state(localStorage.getItem('sarde-auto-save') !== 'false')

  function setFontSize(val) {
    editorFontSize = Math.max(10, Math.min(24, val))
    localStorage.setItem('sarde-font-size', String(editorFontSize))
    window.dispatchEvent(new CustomEvent('sarde:pref-changed', { detail: { key: 'font-size', value: editorFontSize } }))
  }

  function toggleLineNumbers(val) {
    editorLineNumbers = val
    localStorage.setItem('sarde-line-numbers', String(val))
    window.dispatchEvent(new CustomEvent('sarde:pref-changed', { detail: { key: 'line-numbers', value: val } }))
  }

  function toggleWordWrap(val) {
    editorWordWrap = val
    localStorage.setItem('sarde-word-wrap', String(val))
    window.dispatchEvent(new CustomEvent('sarde:pref-changed', { detail: { key: 'word-wrap', value: val } }))
  }

  function toggleAutoSave(val) {
    editorAutoSave = val
    localStorage.setItem('sarde-auto-save', String(val))
  }

  // Shorthand: cfg = siteConfig.data (null while loading)
  let cfg = $derived(siteConfig.data)

  // URL validation
  function isValidUrl(val) {
    if (!val) return true
    try { new URL(val); return true } catch { return false }
  }

  let urlInvalid = $derived(cfg?.site?.url ? !isValidUrl(cfg.site.url) : false)
  let editUrlInvalid = $derived(cfg?.site?.edit_url ? !isValidUrl(cfg.site.edit_url) : false)

  // Settings search
  let searchQuery = $state('')

  const sectionKeywords = {
    general: ['title', 'description', 'url', 'language', 'heading', '404', 'delimiter', 'edit'],
    appearance: ['theme', 'color', 'primary', 'accent', 'code', 'preset'],
    editor: ['font', 'size', 'line', 'wrap', 'auto', 'save'],
    navigation: ['sidebar', 'footer', 'header', 'search', 'pagination', 'badge', 'credits'],
    collections: ['collection', 'content', 'directory'],
    build: ['output', 'sitemap', 'feed', 'rss', 'minify', 'katex', 'mermaid', 'cdn', 'link', 'llms', 'verbose'],
    deploy: ['provider', 'branch', 'netlify', 'github', 'cloudflare', 'vercel', 'site id'],
    about: ['version', 'about'],
  }

  let filteredSections = $derived(
    searchQuery.trim()
      ? sections.filter(s => {
          const q = searchQuery.toLowerCase()
          return s.includes(q) || (sectionKeywords[s] ?? []).some(k => k.includes(q))
        })
      : sections
  )
</script>

<AppDialog
  open={ui.settingsOpen}
  onOpenChange={(v) => { if (!v) close(); }}
  ariaLabel="Settings"
  width="720px"
  onEscapeKeydown={handleEscapeKeydown}
  onInteractOutside={handleInteractOutside}
>
  <div class="modal-header">
    <h2>Settings</h2>
    <div class="modal-header-right">
      {#if siteConfig.saving}
        <span class="save-indicator"><Loader size={14} /> Saving…</span>
      {/if}
      <Toggle.Root
        pressed={yamlMode}
        onPressedChange={(v) => { yamlMode = v; yamlError = '' }}
        class="yaml-toggle"
        title={yamlMode ? 'Switch to form view' : 'Edit raw YAML'}
      >
        <Code size={16} />
      </Toggle.Root>
      <AppButton variant="ghost" size="icon" onclick={close}>
        <X size={18} />
      </AppButton>
    </div>
  </div>

  <div class="modal-body">
    {#if yamlMode}
      <div class="yaml-mode">
        {#if cfg}
          <YamlEditor value={rawYaml} onchange={handleYamlChange} onerror={handleYamlError} />
          {#if yamlError}
            <div class="yaml-error">{yamlError}</div>
          {/if}
        {:else}
          <div class="loading-state"><Loader size={20} /><span>Loading config…</span></div>
        {/if}
      </div>
    {:else}
    <Tabs.Root value={ui.settingsSection} onValueChange={(v) => (ui.settingsSection = v)} orientation="vertical" class="settings-tabs-root">
    <Tabs.List class="settings-nav">
      <input
        type="text"
        class="settings-search"
        placeholder="Search settings…"
        bind:value={searchQuery}
      />
      {#each filteredSections as section}
        <Tabs.Trigger value={section} class="settings-link">
          {capitalize(section)}
        </Tabs.Trigger>
      {/each}
    </Tabs.List>

    <div class="settings-content">
      {#if !cfg && ui.settingsSection !== 'editor' && ui.settingsSection !== 'about'}
        <div class="loading-state">
          <Loader size={20} />
          <span>Loading settings…</span>
        </div>

      {:else if ui.settingsSection === 'general'}
        <h3>General</h3>
        <div class="field">
          <label class="field-label" for="site-title">Site Title</label>
          <input id="site-title" type="text" class="field-input"
            placeholder="My Site"
            value={cfg.site?.title ?? ''}
            oninput={(e) => { cfg.site ??= {}; cfg.site.title = e.target.value; scheduleSave() }}
          />
        </div>
        <div class="field">
          <label class="field-label" for="site-desc">Description</label>
          <textarea id="site-desc" class="field-textarea" rows="3"
            placeholder="Site description"
            value={cfg.site?.description ?? ''}
            oninput={(e) => { cfg.site ??= {}; cfg.site.description = e.target.value; scheduleSave() }}
          ></textarea>
        </div>
        <div class="field">
          <label class="field-label" for="site-url">URL</label>
          <input id="site-url" type="url" class="field-input"
            class:field-invalid={urlInvalid}
            placeholder="https://example.com"
            value={cfg.site?.url ?? ''}
            oninput={(e) => { cfg.site ??= {}; cfg.site.url = e.target.value; scheduleSave() }}
          />
          {#if urlInvalid}
            <span class="field-hint field-hint-error">Please enter a valid URL (e.g. https://example.com)</span>
          {/if}
        </div>
        <div class="field">
          <label class="field-label" for="site-lang">Language</label>
          <input id="site-lang" type="text" class="field-input"
            placeholder="en"
            value={cfg.site?.language ?? 'en'}
            oninput={(e) => { cfg.site ??= {}; cfg.site.language = e.target.value; scheduleSave() }}
          />
        </div>
        <div class="field">
          <label class="field-label" for="edit-url">Edit URL</label>
          <input id="edit-url" type="url" class="field-input"
            class:field-invalid={editUrlInvalid}
            placeholder="https://github.com/user/repo/edit/main/content"
            value={cfg.site?.edit_url ?? ''}
            oninput={(e) => { cfg.site ??= {}; cfg.site.edit_url = e.target.value; scheduleSave() }}
          />
          {#if editUrlInvalid}
            <span class="field-hint field-hint-error">Please enter a valid URL</span>
          {/if}
        </div>
        <div class="field">
          <label class="field-label" for="title-delim">Title Delimiter</label>
          <input id="title-delim" type="text" class="field-input field-input-short"
            placeholder="|"
            value={cfg.site?.title_delimiter ?? '|'}
            oninput={(e) => { cfg.site ??= {}; cfg.site.title_delimiter = e.target.value; scheduleSave() }}
          />
          <span class="field-hint">Separator between page title and site title in browser tab</span>
        </div>
        <div class="field">
          <label class="field-label" for="custom-404">Custom 404 Page</label>
          <input id="custom-404" type="text" class="field-input"
            placeholder="404.md"
            value={cfg.site?.custom_404 ?? ''}
            oninput={(e) => { cfg.site ??= {}; cfg.site.custom_404 = e.target.value; scheduleSave() }}
          />
        </div>
        <div class="field">
          <label class="field-check">
            <input type="checkbox"
              checked={cfg.site?.heading_links ?? true}
              onchange={(e) => { cfg.site ??= {}; cfg.site.heading_links = e.target.checked; immediateSave() }}
            />
            <span>Heading anchor links</span>
          </label>
        </div>

      {:else if ui.settingsSection === 'appearance'}
        <h3>Appearance</h3>
        <div class="field">
          <span class="field-label" id="theme-label">Theme</span>
          <RadioGroup.Root value={cfg.theme?.name ?? 'default'} onValueChange={(v) => { cfg.theme ??= {}; cfg.theme.name = v; immediateSave() }} class="theme-grid" aria-labelledby="theme-label">
            {#each ['default', 'academic', 'minimal', 'docs', 'clean'] as theme}
              <RadioGroup.Item value={theme} class="theme-card">
                <div class="theme-preview theme-{theme}"></div>
                <span class="theme-name">{capitalize(theme)}</span>
              </RadioGroup.Item>
            {/each}
          </RadioGroup.Root>
        </div>
        <div class="field">
          <span class="field-label">Color Preset</span>
          <div class="preset-row">
            {#each THEME_PRESETS as preset}
              <button
                class="preset-swatch"
                class:active={cfg.theme?.preset === preset.id}
                title={preset.name}
                onclick={() => {
                  cfg.theme ??= {};
                  cfg.theme.preset = preset.id;
                  cfg.theme.primary_color = preset.primary;
                  cfg.theme.accent_color = preset.accent;
                  immediateSave();
                }}
              >
                <span class="swatch-half" style="background:{preset.primary}"></span>
                <span class="swatch-half" style="background:{preset.accent}"></span>
              </button>
            {/each}
          </div>
        </div>
        <div class="field">
          <label class="field-label" for="primary-color">Primary Color</label>
          <div class="color-row">
            <input id="primary-color" type="color" class="field-color"
              value={cfg.theme?.primary_color || '#6366f1'}
              oninput={(e) => { cfg.theme ??= {}; cfg.theme.primary_color = e.target.value; scheduleSave() }}
            />
            <input type="text" class="field-input color-text"
              value={cfg.theme?.primary_color || ''}
              placeholder="#6366f1"
              oninput={(e) => { cfg.theme ??= {}; cfg.theme.primary_color = e.target.value; scheduleSave() }}
            />
          </div>
        </div>
        <div class="field">
          <label class="field-label" for="accent-color">Accent Color</label>
          <div class="color-row">
            <input id="accent-color" type="color" class="field-color"
              value={cfg.theme?.accent_color || '#89b4fa'}
              oninput={(e) => { cfg.theme ??= {}; cfg.theme.accent_color = e.target.value; scheduleSave() }}
            />
            <input type="text" class="field-input color-text"
              value={cfg.theme?.accent_color || ''}
              placeholder="#89b4fa"
              oninput={(e) => { cfg.theme ??= {}; cfg.theme.accent_color = e.target.value; scheduleSave() }}
            />
          </div>
        </div>
        <div class="field">
          <label class="field-label" for="code-light">Code Theme (Light)</label>
          <input id="code-light" type="text" class="field-input"
            placeholder="github"
            value={cfg.theme?.code_light ?? ''}
            oninput={(e) => { cfg.theme ??= {}; cfg.theme.code_light = e.target.value; scheduleSave() }}
          />
        </div>
        <div class="field">
          <label class="field-label" for="code-dark">Code Theme (Dark)</label>
          <input id="code-dark" type="text" class="field-input"
            placeholder="catppuccin-mocha"
            value={cfg.theme?.code_dark ?? ''}
            oninput={(e) => { cfg.theme ??= {}; cfg.theme.code_dark = e.target.value; scheduleSave() }}
          />
        </div>

      {:else if ui.settingsSection === 'editor'}
        <h3>Editor</h3>
        <p class="section-desc">These preferences are local to this device and are not saved to config.yaml.</p>

        <div class="field">
          <span class="field-label">Interface Theme</span>
          <RadioGroup.Root value={getCurrentTheme()} onValueChange={(v) => setTheme(v)} class="interface-theme-grid">
            {#each THEMES as theme}
              <RadioGroup.Item value={theme.id} class="interface-theme-card">
                {#if theme.id === 'system'}
                  <Monitor size={14} />
                {/if}
                <span>{theme.name}</span>
              </RadioGroup.Item>
            {/each}
          </RadioGroup.Root>
        </div>

        <div class="field">
          <label class="field-label" for="font-size">Font Size</label>
          <div class="range-row">
            <input id="font-size" type="range" min="10" max="24"
              value={editorFontSize}
              class="field-range"
              oninput={(e) => setFontSize(parseInt(e.target.value))}
            />
            <span class="range-value">{editorFontSize}px</span>
          </div>
        </div>
        <div class="field">
          <label class="field-check">
            <input type="checkbox"
              checked={editorLineNumbers}
              onchange={(e) => toggleLineNumbers(e.target.checked)}
            />
            <span>Line Numbers</span>
          </label>
        </div>
        <div class="field">
          <label class="field-check">
            <input type="checkbox"
              checked={editorWordWrap}
              onchange={(e) => toggleWordWrap(e.target.checked)}
            />
            <span>Word Wrap</span>
          </label>
        </div>
        <div class="field">
          <label class="field-check">
            <input type="checkbox"
              checked={editorAutoSave}
              onchange={(e) => toggleAutoSave(e.target.checked)}
            />
            <span>Auto-save (5s delay)</span>
          </label>
        </div>

      {:else if ui.settingsSection === 'navigation'}
        <h3>Navigation</h3>
        <p class="section-desc">Configure site navigation structure. Drag items to reorder, or add custom links and groups.</p>

        <NavigationBuilder />

        <h4 class="nav-sub-heading">Sidebar Options</h4>
        <div class="field-group">
          <div class="field">
            <label class="field-check">
              <input type="checkbox"
                checked={cfg.sidebar?.auto_generate ?? true}
                onchange={(e) => { cfg.sidebar ??= {}; cfg.sidebar.auto_generate = e.target.checked; immediateSave() }}
              />
              <span>Auto-generate sidebar</span>
            </label>
          </div>
          <div class="field">
            <label class="field-check">
              <input type="checkbox"
                checked={cfg.sidebar?.collapsed ?? false}
                onchange={(e) => { cfg.sidebar ??= {}; cfg.sidebar.collapsed = e.target.checked; immediateSave() }}
              />
              <span>Collapse by default</span>
            </label>
          </div>
          <div class="field">
            <label class="field-check">
              <input type="checkbox"
                checked={cfg.sidebar?.badges ?? true}
                onchange={(e) => { cfg.sidebar ??= {}; cfg.sidebar.badges = e.target.checked; immediateSave() }}
              />
              <span>Show badges</span>
            </label>
          </div>
          <div class="field">
            <label class="field-check">
              <input type="checkbox"
                checked={cfg.sidebar?.pagination ?? true}
                onchange={(e) => { cfg.sidebar ??= {}; cfg.sidebar.pagination = e.target.checked; immediateSave() }}
              />
              <span>Previous/next links</span>
            </label>
          </div>
        </div>

        <h4 class="nav-sub-heading">Header & Footer</h4>
        <div class="field-group">
          <div class="field">
            <label class="field-check">
              <input type="checkbox"
                checked={cfg.header?.search ?? true}
                onchange={(e) => { cfg.header ??= {}; cfg.header.search = e.target.checked; immediateSave() }}
              />
              <span>Header search bar</span>
            </label>
          </div>
          <div class="field">
            <label class="field-check">
              <input type="checkbox"
                checked={cfg.header?.theme_toggle ?? true}
                onchange={(e) => { cfg.header ??= {}; cfg.header.theme_toggle = e.target.checked; immediateSave() }}
              />
              <span>Header theme toggle</span>
            </label>
          </div>
        </div>
        <div class="field">
          <label class="field-label" for="footer-text">Footer Text</label>
          <input id="footer-text" type="text" class="field-input"
            placeholder="© 2025 My Site"
            value={cfg.footer?.text ?? ''}
            oninput={(e) => { cfg.footer ??= {}; cfg.footer.text = e.target.value; scheduleSave() }}
          />
        </div>
        <div class="field">
          <label class="field-check">
            <input type="checkbox"
              checked={cfg.footer?.credits ?? false}
              onchange={(e) => { cfg.footer ??= {}; cfg.footer.credits = e.target.checked; immediateSave() }}
            />
            <span>Show "Built with Sarde" in footer</span>
          </label>
        </div>

      {:else if ui.settingsSection === 'collections'}
        <h3>Collections</h3>
        <p class="section-desc">Content collections are subdirectories of content/. Each directory is a collection.</p>

        {#if collectionsLoading}
          <div class="loading-state"><Loader size={16} /><span>Loading…</span></div>
        {:else}
          <div class="collection-list">
            {#each collections as col}
              <div class="collection-card">
                <div class="collection-info">
                  <span class="collection-name">{col.title}</span>
                  <span class="collection-meta">{col.name}/ — {col.pageCount} page{col.pageCount !== 1 ? 's' : ''}</span>
                </div>
                <AlertDialog.Root>
                  <AlertDialog.Trigger class="collection-delete" title="Delete collection">
                    <Trash2 size={14} />
                  </AlertDialog.Trigger>
                  <AlertDialog.Portal>
                    <AlertDialog.Overlay class="cr-alert-overlay" />
                    <AlertDialog.Content class="cr-alert-content">
                      <AlertDialog.Title class="cr-alert-title">Delete Collection</AlertDialog.Title>
                      <AlertDialog.Description class="cr-alert-desc">Are you sure you want to delete "{col.name}"?</AlertDialog.Description>
                      <div class="cr-alert-actions">
                        <AlertDialog.Cancel class="cr-btn cr-btn-secondary cr-btn-sm">Cancel</AlertDialog.Cancel>
                        <AlertDialog.Action class="cr-btn cr-btn-danger cr-btn-sm" onclick={() => handleDeleteCollection(col.name)}>Delete</AlertDialog.Action>
                      </div>
                    </AlertDialog.Content>
                  </AlertDialog.Portal>
                </AlertDialog.Root>
              </div>
            {/each}

            {#if collections.length === 0}
              <p class="empty-msg">No collections found. Create one below.</p>
            {/if}
          </div>

          <div class="add-collection-row">
            <input
              type="text"
              class="field-input"
              placeholder="Collection name (e.g. tutorials)"
              bind:value={newCollectionName}
              onkeydown={(e) => e.key === 'Enter' && handleCreateCollection()}
            />
            <button class="add-btn" onclick={handleCreateCollection} disabled={!newCollectionName.trim() || collectionCreating}>
              <Plus size={14} /> Create
            </button>
          </div>
        {/if}

      {:else if ui.settingsSection === 'build'}
        <h3>Build</h3>
        <div class="field">
          <label class="field-label" for="output-dir">Output Directory</label>
          <input id="output-dir" type="text" class="field-input"
            placeholder="dist"
            value={cfg.build?.output ?? 'dist'}
            oninput={(e) => { cfg.build ??= {}; cfg.build.output = e.target.value; scheduleSave() }}
          />
        </div>
        <div class="field">
          <label class="field-label" for="base-path">Base Path</label>
          <input id="base-path" type="text" class="field-input"
            placeholder="/"
            value={cfg.build?.base_path ?? ''}
            oninput={(e) => { cfg.build ??= {}; cfg.build.base_path = e.target.value; scheduleSave() }}
          />
        </div>
        <div class="field-group">
          {#each [
            ['sitemap',    'Generate Sitemap'],
            ['feed',       'Generate RSS Feed'],
            ['search',     'Build Search Index'],
            ['minify',     'Minify HTML'],
            ['clean',      'Clean Output on Build'],
            ['link_check', 'Check Links'],
            ['llms',       'Generate llms.txt'],
            ['katex',      'Enable KaTeX (Math)'],
            ['mermaid',    'Enable Mermaid (Diagrams)'],
            ['cdn',        'Load KaTeX/Mermaid from CDN'],
          ] as [key, label]}
            <div class="field">
              <label class="field-check">
                <input type="checkbox"
                  checked={cfg.build?.[key] ?? false}
                  onchange={(e) => { cfg.build ??= {}; cfg.build[key] = e.target.checked; immediateSave() }}
                />
                <span>{label}</span>
              </label>
            </div>
          {/each}
        </div>
        <h4>Output</h4>
        <div class="field">
          <label class="field-check">
            <input type="checkbox"
              checked={localStorage.getItem('sarde-verbose-build') === 'true'}
              onchange={(e) => { localStorage.setItem('sarde-verbose-build', e.target.checked ? 'true' : 'false') }}
            />
            <span>Verbose build output</span>
          </label>
          <p class="field-hint">Show per-phase timing and detailed stats when building</p>
        </div>

      {:else if ui.settingsSection === 'deploy'}
        <h3>Deploy</h3>
        <div class="field">
          <label class="field-label" for="deploy-provider">Provider</label>
          <select id="deploy-provider" class="field-select"
            value={cfg.deploy?.provider ?? ''}
            onchange={(e) => { cfg.deploy ??= {}; cfg.deploy.provider = e.target.value; immediateSave() }}
          >
            <option value="">None</option>
            <option value="github">GitHub Pages</option>
            <option value="netlify">Netlify</option>
            <option value="cloudflare">Cloudflare Pages</option>
            <option value="vercel">Vercel</option>
            <option value="custom">Custom</option>
          </select>
        </div>
        <div class="field">
          <label class="field-label" for="deploy-branch">Branch</label>
          <input id="deploy-branch" type="text" class="field-input"
            placeholder="main"
            value={cfg.deploy?.branch ?? 'main'}
            oninput={(e) => { cfg.deploy ??= {}; cfg.deploy.branch = e.target.value; scheduleSave() }}
          />
        </div>
        <div class="field">
          <label class="field-label" for="deploy-siteid">Site ID</label>
          <input id="deploy-siteid" type="text" class="field-input"
            placeholder="your-site-id"
            value={cfg.deploy?.site_id ?? ''}
            oninput={(e) => { cfg.deploy ??= {}; cfg.deploy.site_id = e.target.value; scheduleSave() }}
          />
        </div>

      {:else if ui.settingsSection === 'about'}
        <h3>About</h3>
        <div class="about-block">
          <p class="about-name">Sarde Desktop</p>
          <p class="about-version">v0.1.0</p>
          <p class="about-detail">Built with Tauri v2 + Svelte 5</p>
          <p class="about-detail">Go CLI for SSG build &amp; preview</p>
          <p class="about-status about-online">Rust backend active</p>
        </div>
      {/if}
    </div>
    </Tabs.Root>
    {/if}
  </div>

  {#if showDirtyBar}
    <div class="dirty-bar">
      <span>You have unsaved changes</span>
      <div class="dirty-actions">
        <button class="dirty-btn cancel" onclick={() => showDirtyBar = false}>Cancel</button>
        <button class="dirty-btn discard" onclick={discardAndClose}>Discard</button>
        <button class="dirty-btn save" onclick={saveAndClose}>Save & Close</button>
      </div>
    </div>
  {/if}
</AppDialog>

<style>
  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid var(--cr-border);
  }

  .modal-header h2 {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    color: var(--cr-text);
  }

  .modal-header-right {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .save-indicator {
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: 12px;
    color: var(--cr-text-muted);
  }

  .modal-body {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  :global(.settings-tabs-root) {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  :global(.settings-nav) {
    display: flex;
    flex-direction: column;
    width: 160px;
    padding: 8px;
    gap: 2px;
    border-right: 1px solid var(--cr-border);
    overflow-y: auto;
    flex-shrink: 0;
  }

  :global(.settings-link) {
    padding: 8px 12px;
    border: none;
    border-radius: var(--cr-radius);
    background: transparent;
    color: var(--cr-text-muted);
    font-size: 13px;
    text-align: left;
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
  }

  :global(.settings-link:hover) {
    color: var(--cr-text);
    background: var(--cr-hover);
  }

  :global(.settings-link[data-state="active"]) {
    color: var(--cr-text);
    background: var(--cr-active);
    font-weight: 500;
  }

  .settings-content {
    flex: 1;
    padding: 20px 24px;
    overflow-y: auto;
  }

  .settings-content h3 {
    margin: 0 0 16px;
    font-size: 15px;
    font-weight: 600;
    color: var(--cr-text);
  }

  .section-desc {
    font-size: 12px;
    color: var(--cr-text-muted);
    margin: -8px 0 16px;
    line-height: 1.5;
  }

  .nav-sub-heading {
    margin: 20px 0 10px;
    font-size: 12px;
    font-weight: 600;
    color: var(--cr-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .loading-state {
    display: flex;
    align-items: center;
    gap: 10px;
    color: var(--cr-text-muted);
    font-size: 13px;
    padding: 20px 0;
  }

  .field {
    margin-bottom: 14px;
  }

  .field-group {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 4px 16px;
  }

  .field-label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: var(--cr-text-muted);
    margin-bottom: 6px;
  }

  .field-input,
  .field-textarea,
  .field-select {
    width: 100%;
    padding: 8px 10px;
    font-size: 13px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: var(--cr-bg-input);
    color: var(--cr-text);
    outline: none;
    box-sizing: border-box;
    font-family: inherit;
  }

  .field-input-short {
    max-width: 100px;
  }

  .field-hint {
    display: block;
    font-size: 11px;
    color: var(--cr-text-muted);
    margin-top: 4px;
  }

  .field-input:focus,
  .field-textarea:focus,
  .field-select:focus {
    border-color: var(--cr-accent);
  }

  .field-textarea {
    resize: vertical;
  }

  .field-select {
    cursor: pointer;
  }

  .field-check {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: var(--cr-text);
    cursor: pointer;
  }

  .field-check input[type='checkbox'] {
    width: 16px;
    height: 16px;
    accent-color: var(--cr-accent);
    cursor: pointer;
  }

  .range-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .field-range {
    flex: 1;
    accent-color: var(--cr-accent);
  }

  .range-value {
    font-size: 12px;
    color: var(--cr-text-muted);
    min-width: 36px;
    text-align: right;
  }

  /* Color picker */
  .color-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .field-color {
    width: 36px;
    height: 36px;
    padding: 2px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: var(--cr-bg-input);
    cursor: pointer;
    flex-shrink: 0;
  }

  .color-text {
    flex: 1;
  }

  /* Theme grid */
  :global(.theme-grid) {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 10px;
    margin-top: 4px;
  }

  :global(.theme-card) {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 10px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: var(--cr-bg-input);
    cursor: pointer;
    transition: border-color 0.15s;
  }

  :global(.theme-card:hover),
  :global(.theme-card[data-state="checked"]) {
    border-color: var(--cr-accent);
  }

  :global(.theme-card[data-state="checked"]) {
    background: var(--cr-accent-bg);
  }

  .theme-preview {
    width: 100%;
    height: 48px;
    border-radius: var(--cr-radius-sm);
  }

  .theme-default  { background: linear-gradient(135deg, #1e1e2e, #313244); }
  .theme-academic { background: linear-gradient(135deg, #2b2d42, #8d99ae); }
  .theme-minimal  { background: linear-gradient(135deg, #fafafa, #e0e0e0); }
  .theme-docs     { background: linear-gradient(135deg, #1a1b26, #414868); }
  .theme-clean    { background: linear-gradient(135deg, #f8f9fa, #dee2e6); }

  .theme-name {
    font-size: 11px;
    color: var(--cr-text-muted);
  }

  /* About */
  .about-block { line-height: 1.6; }
  .about-block p { margin: 0 0 4px; }
  .about-name    { font-size: 16px; font-weight: 600; color: var(--cr-text); }
  .about-version { font-size: 13px; color: var(--cr-accent); }
  .about-detail  { font-size: 13px; color: var(--cr-text-muted); }

  .about-status {
    margin-top: 12px !important;
    font-size: 12px;
    padding: 6px 10px;
    border-radius: var(--cr-radius);
    display: inline-block;
  }

  .about-online  { background: var(--cr-success-bg); color: var(--cr-success); }
  .about-offline { background: var(--cr-danger-bg); color: var(--cr-danger); }

  /* YAML toggle button */
  :global(.yaml-toggle) {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
    transition: all 0.15s;
  }

  :global(.yaml-toggle:hover) {
    color: var(--cr-text);
    border-color: var(--cr-text-muted);
  }

  :global(.yaml-toggle[data-state="on"]) {
    color: var(--cr-accent);
    border-color: var(--cr-accent);
    background: var(--cr-active);
  }

  .yaml-mode {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .yaml-error {
    padding: 8px 12px;
    font-size: 12px;
    color: var(--cr-danger);
    background: var(--cr-danger-bg);
    border-top: 1px solid var(--cr-border);
  }

  /* Color presets */
  .preset-row {
    display: flex;
    gap: 8px;
    margin-top: 4px;
  }

  .preset-swatch {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    border: 2px solid var(--cr-border);
    overflow: hidden;
    cursor: pointer;
    display: flex;
    padding: 0;
    background: none;
    transition: border-color 0.15s, transform 0.1s;
  }

  .preset-swatch:hover {
    border-color: var(--cr-text-muted);
    transform: scale(1.1);
  }

  .preset-swatch.active {
    border-color: var(--cr-accent);
    box-shadow: 0 0 0 2px var(--cr-accent-bg);
  }

  .swatch-half {
    flex: 1;
  }

  /* Collections tab */
  .collection-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-bottom: 14px;
  }

  .collection-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: var(--cr-bg-input);
  }

  .collection-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .collection-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--cr-text);
  }

  .collection-meta {
    font-size: 11px;
    color: var(--cr-text-muted);
  }

  :global(.collection-delete) {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border: none;
    border-radius: var(--cr-radius);
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
  }

  :global(.collection-delete:hover) {
    color: var(--cr-danger);
    background: var(--cr-danger-bg);
  }

  /* AlertDialog for delete confirmation */
  :global(.cr-alert-overlay) {
    position: fixed;
    inset: 0;
    z-index: 150;
    background: var(--cr-bg-overlay);
  }

  :global(.cr-alert-content) {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 151;
    max-width: 340px;
    width: 90%;
    padding: 20px;
    background: var(--cr-bg-base);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-lg);
    box-shadow: var(--cr-shadow-lg);
  }

  :global(.cr-alert-title) {
    margin: 0 0 6px;
    font-size: 15px;
    font-weight: 600;
    color: var(--cr-text);
  }

  :global(.cr-alert-desc) {
    margin: 0 0 16px;
    font-size: 13px;
    color: var(--cr-text-muted);
    line-height: 1.4;
  }

  :global(.cr-alert-actions) {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .empty-msg {
    font-size: 13px;
    color: var(--cr-text-muted);
    text-align: center;
    padding: 16px 0;
  }

  .add-collection-row {
    display: flex;
    gap: 8px;
  }

  .add-btn {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 8px 14px;
    border: 1px solid var(--cr-accent);
    border-radius: var(--cr-radius);
    background: var(--cr-active);
    color: var(--cr-accent);
    font-size: 12px;
    font-weight: 500;
    font-family: inherit;
    cursor: pointer;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .add-btn:hover:not(:disabled) {
    background: var(--cr-accent-bg);
  }

  .add-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  /* Settings search */
  .settings-search {
    width: 100%;
    padding: 7px 10px;
    font-size: 12px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: var(--cr-bg-input);
    color: var(--cr-text);
    outline: none;
    box-sizing: border-box;
    font-family: inherit;
    margin-bottom: 6px;
  }

  .settings-search:focus {
    border-color: var(--cr-accent);
  }

  .settings-search::placeholder {
    color: var(--cr-text-muted);
  }

  /* URL validation */
  .field-invalid {
    border-color: var(--cr-danger) !important;
  }

  .field-hint-error {
    color: var(--cr-danger) !important;
  }

  /* Unsaved changes bar */
  .dirty-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 16px;
    border-top: 1px solid var(--cr-border);
    background: rgba(249, 226, 175, 0.06);
    font-size: 13px;
    color: var(--cr-warning);
  }

  .dirty-actions {
    display: flex;
    gap: 8px;
  }

  .dirty-btn {
    padding: 5px 14px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    font-size: 12px;
    font-family: inherit;
    cursor: pointer;
    background: transparent;
  }

  .dirty-btn.cancel {
    color: var(--cr-text-muted);
  }

  .dirty-btn.cancel:hover {
    color: var(--cr-text);
  }

  .dirty-btn.discard {
    color: var(--cr-danger);
    border-color: var(--cr-danger);
  }

  .dirty-btn.discard:hover {
    background: var(--cr-danger-bg);
  }

  .dirty-btn.save {
    color: var(--cr-bg-base);
    background: var(--cr-accent);
    border-color: var(--cr-accent);
    font-weight: 600;
  }

  .dirty-btn.save:hover {
    background: var(--cr-accent-hover);
    border-color: var(--cr-accent-hover);
  }

  /* Interface theme picker */
  :global(.interface-theme-grid) {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 6px;
    margin-top: 4px;
  }

  :global(.interface-theme-card) {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 5px;
    padding: 8px 10px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: var(--cr-bg-input);
    color: var(--cr-text-muted);
    font-size: 12px;
    font-family: inherit;
    cursor: pointer;
    transition: border-color 0.15s, color 0.15s, background 0.15s;
  }

  :global(.interface-theme-card:hover) {
    color: var(--cr-text);
    border-color: var(--cr-text-muted);
  }

  :global(.interface-theme-card[data-state="checked"]) {
    color: var(--cr-accent);
    border-color: var(--cr-accent);
    background: var(--cr-active);
  }
</style>
