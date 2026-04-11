<script>
  import { ui, project, siteConfig, loadSiteConfig, saveSiteConfig } from '../stores/app.svelte.js'
  import { X, Loader } from 'lucide-svelte'

  const sections = ['general', 'appearance', 'editor', 'navigation', 'build', 'deploy', 'about']

  // Reload config whenever the modal opens or the project changes
  $effect(() => {
    if (ui.settingsOpen && project.contentPath) {
      loadSiteConfig()
    }
  })

  function close() {
    ui.settingsOpen = false
  }

  function onKeydown(e) {
    if (e.key === 'Escape') close()
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

  // ---- Editor prefs (localStorage only — not in site.yaml) ----
  let editorFontSize = $state(parseInt(localStorage.getItem('coderoo-font-size') || '14'))
  let editorLineNumbers = $state(localStorage.getItem('coderoo-line-numbers') !== 'false')
  let editorWordWrap = $state(localStorage.getItem('coderoo-word-wrap') === 'true')
  let editorAutoSave = $state(localStorage.getItem('coderoo-auto-save') !== 'false')

  function setFontSize(val) {
    editorFontSize = Math.max(10, Math.min(24, val))
    localStorage.setItem('coderoo-font-size', String(editorFontSize))
    window.dispatchEvent(new CustomEvent('coderoo:pref-changed', { detail: { key: 'font-size', value: editorFontSize } }))
  }

  function toggleLineNumbers(val) {
    editorLineNumbers = val
    localStorage.setItem('coderoo-line-numbers', String(val))
    window.dispatchEvent(new CustomEvent('coderoo:pref-changed', { detail: { key: 'line-numbers', value: val } }))
  }

  function toggleWordWrap(val) {
    editorWordWrap = val
    localStorage.setItem('coderoo-word-wrap', String(val))
    window.dispatchEvent(new CustomEvent('coderoo:pref-changed', { detail: { key: 'word-wrap', value: val } }))
  }

  function toggleAutoSave(val) {
    editorAutoSave = val
    localStorage.setItem('coderoo-auto-save', String(val))
  }

  // Shorthand: cfg = siteConfig.data (null while loading)
  let cfg = $derived(siteConfig.data)
</script>

<svelte:window onkeydown={onKeydown} />

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="modal-overlay" onclick={close} onkeydown={(e) => e.key === 'Escape' && close()} role="dialog" aria-modal="true" aria-label="Settings" tabindex="-1">
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions, a11y_click_events_have_key_events -->
  <div class="modal-content" onclick={(e) => e.stopPropagation()}>
    <div class="modal-header">
      <h2>Settings</h2>
      <div class="modal-header-right">
        {#if siteConfig.saving}
          <span class="save-indicator"><Loader size={14} /> Saving…</span>
        {/if}
        <button class="modal-close" onclick={close} title="Close">
          <X size={18} />
        </button>
      </div>
    </div>

    <div class="modal-body">
      <nav class="settings-nav">
        {#each sections as section}
          <button
            class="settings-link"
            class:active={ui.settingsSection === section}
            onclick={() => (ui.settingsSection = section)}
          >
            {capitalize(section)}
          </button>
        {/each}
      </nav>

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
              placeholder="https://example.com"
              value={cfg.site?.url ?? ''}
              oninput={(e) => { cfg.site ??= {}; cfg.site.url = e.target.value; scheduleSave() }}
            />
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
              placeholder="https://github.com/user/repo/edit/main/content"
              value={cfg.site?.edit_url ?? ''}
              oninput={(e) => { cfg.site ??= {}; cfg.site.edit_url = e.target.value; scheduleSave() }}
            />
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
            <div class="theme-grid" role="radiogroup" aria-labelledby="theme-label">
              {#each ['default', 'academic', 'minimal', 'docs', 'clean'] as theme}
                <button
                  class="theme-card"
                  class:selected={cfg.theme?.name === theme}
                  onclick={() => { cfg.theme ??= {}; cfg.theme.name = theme; immediateSave() }}
                >
                  <div class="theme-preview theme-{theme}"></div>
                  <span class="theme-name">{capitalize(theme)}</span>
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
          <div class="field">
            <label class="field-check">
              <input type="checkbox"
                checked={cfg.sidebar?.auto_generate ?? true}
                onchange={(e) => { cfg.sidebar ??= {}; cfg.sidebar.auto_generate = e.target.checked; immediateSave() }}
              />
              <span>Auto-generate sidebar from file structure</span>
            </label>
          </div>
          <div class="field">
            <label class="field-check">
              <input type="checkbox"
                checked={cfg.sidebar?.collapsed ?? false}
                onchange={(e) => { cfg.sidebar ??= {}; cfg.sidebar.collapsed = e.target.checked; immediateSave() }}
              />
              <span>Collapse sidebar by default</span>
            </label>
          </div>
          <div class="field">
            <label class="field-check">
              <input type="checkbox"
                checked={cfg.sidebar?.badges ?? true}
                onchange={(e) => { cfg.sidebar ??= {}; cfg.sidebar.badges = e.target.checked; immediateSave() }}
              />
              <span>Show badges in sidebar</span>
            </label>
          </div>
          <div class="field">
            <label class="field-check">
              <input type="checkbox"
                checked={cfg.sidebar?.pagination ?? true}
                onchange={(e) => { cfg.sidebar ??= {}; cfg.sidebar.pagination = e.target.checked; immediateSave() }}
              />
              <span>Show previous/next page links</span>
            </label>
          </div>
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
              <span>Show "Built with Coderoo" in footer</span>
            </label>
          </div>

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
            <p class="about-name">Coderoo Desktop</p>
            <p class="about-version">v0.1.0</p>
            <p class="about-detail">Built with Tauri v2 + Svelte 5</p>
            <p class="about-detail">Go CLI for SSG build &amp; preview</p>
            <p class="about-status about-online">Rust backend active</p>
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
  }

  .modal-content {
    width: 720px;
    max-width: 90vw;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    background: var(--color-surface, #1e1e2e);
    border: 1px solid var(--color-border, #313244);
    border-radius: 12px;
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.4);
    overflow: hidden;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid var(--color-border, #313244);
  }

  .modal-header h2 {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    color: var(--color-text, #cdd6f4);
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
    color: var(--color-text-muted, #6c7086);
  }

  .modal-close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--color-text-muted, #6c7086);
    cursor: pointer;
  }

  .modal-close:hover {
    background: var(--color-hover, rgba(255, 255, 255, 0.06));
    color: var(--color-text, #cdd6f4);
  }

  .modal-body {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  .settings-nav {
    display: flex;
    flex-direction: column;
    width: 160px;
    padding: 8px;
    gap: 2px;
    border-right: 1px solid var(--color-border, #313244);
    overflow-y: auto;
    flex-shrink: 0;
  }

  .settings-link {
    padding: 8px 12px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--color-text-muted, #6c7086);
    font-size: 13px;
    text-align: left;
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
  }

  .settings-link:hover {
    color: var(--color-text, #cdd6f4);
    background: var(--color-hover, rgba(255, 255, 255, 0.06));
  }

  .settings-link.active {
    color: var(--color-text, #cdd6f4);
    background: var(--color-active, rgba(137, 180, 250, 0.1));
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
    color: var(--color-text, #cdd6f4);
  }

  .section-desc {
    font-size: 12px;
    color: var(--color-text-muted, #6c7086);
    margin: -8px 0 16px;
    line-height: 1.5;
  }

  .loading-state {
    display: flex;
    align-items: center;
    gap: 10px;
    color: var(--color-text-muted, #6c7086);
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
    color: var(--color-text-muted, #6c7086);
    margin-bottom: 6px;
  }

  .field-input,
  .field-textarea,
  .field-select {
    width: 100%;
    padding: 8px 10px;
    font-size: 13px;
    border: 1px solid var(--color-border, #313244);
    border-radius: 6px;
    background: var(--color-input, #11111b);
    color: var(--color-text, #cdd6f4);
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
    color: var(--color-text-muted, #6c7086);
    margin-top: 4px;
  }

  .field-input:focus,
  .field-textarea:focus,
  .field-select:focus {
    border-color: var(--color-accent, #89b4fa);
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
    color: var(--color-text, #cdd6f4);
    cursor: pointer;
  }

  .field-check input[type='checkbox'] {
    width: 16px;
    height: 16px;
    accent-color: var(--color-accent, #89b4fa);
    cursor: pointer;
  }

  .range-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .field-range {
    flex: 1;
    accent-color: var(--color-accent, #89b4fa);
  }

  .range-value {
    font-size: 12px;
    color: var(--color-text-muted, #6c7086);
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
    border: 1px solid var(--color-border, #313244);
    border-radius: 6px;
    background: var(--color-input, #11111b);
    cursor: pointer;
    flex-shrink: 0;
  }

  .color-text {
    flex: 1;
  }

  /* Theme grid */
  .theme-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 10px;
    margin-top: 4px;
  }

  .theme-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 10px;
    border: 1px solid var(--color-border, #313244);
    border-radius: 8px;
    background: var(--color-surface-alt, #181825);
    cursor: pointer;
    transition: border-color 0.15s;
  }

  .theme-card:hover,
  .theme-card.selected {
    border-color: var(--color-accent, #89b4fa);
  }

  .theme-card.selected {
    background: var(--color-active, rgba(137, 180, 250, 0.08));
  }

  .theme-preview {
    width: 100%;
    height: 48px;
    border-radius: 4px;
  }

  .theme-default  { background: linear-gradient(135deg, #1e1e2e, #313244); }
  .theme-academic { background: linear-gradient(135deg, #2b2d42, #8d99ae); }
  .theme-minimal  { background: linear-gradient(135deg, #fafafa, #e0e0e0); }
  .theme-docs     { background: linear-gradient(135deg, #1a1b26, #414868); }
  .theme-clean    { background: linear-gradient(135deg, #f8f9fa, #dee2e6); }

  .theme-name {
    font-size: 11px;
    color: var(--color-text-muted, #6c7086);
  }

  /* About */
  .about-block { line-height: 1.6; }
  .about-block p { margin: 0 0 4px; }
  .about-name    { font-size: 16px; font-weight: 600; color: var(--color-text, #cdd6f4); }
  .about-version { font-size: 13px; color: var(--color-accent, #89b4fa); }
  .about-detail  { font-size: 13px; color: var(--color-text-muted, #6c7086); }

  .about-status {
    margin-top: 12px !important;
    font-size: 12px;
    padding: 6px 10px;
    border-radius: 6px;
    display: inline-block;
  }

  .about-online  { background: rgba(166, 227, 161, 0.1); color: #a6e3a1; }
  .about-offline { background: rgba(243, 139, 168, 0.1); color: #f38ba8; }
</style>
