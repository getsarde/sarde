<script>
  import { FileText, BookOpen, Newspaper } from 'lucide-svelte'
  import { RadioGroup } from 'bits-ui'

  let { template = 'empty', description = '', author = '', onTemplateChange, onDescriptionChange, onAuthorChange } = $props()

  const templates = [
    { id: 'empty', name: 'Empty', desc: 'Blank slate with a single welcome page', icon: FileText },
    { id: 'blog', name: 'Blog', desc: 'Posts collection with RSS feed enabled', icon: Newspaper },
    { id: 'docs', name: 'Docs', desc: 'Documentation site with sidebar navigation', icon: BookOpen },
  ]
</script>

<div class="section">
  <span class="section-label">Template</span>
  <RadioGroup.Root value={template} onValueChange={(v) => onTemplateChange(v)} class="template-grid">
    {#each templates as t}
      {@const Icon = t.icon}
      <RadioGroup.Item value={t.id} class="template-card">
        <Icon size={20} />
        <span class="template-name">{t.name}</span>
        <span class="template-desc">{t.desc}</span>
      </RadioGroup.Item>
    {/each}
  </RadioGroup.Root>
</div>

<div class="field">
  <label class="field-label" for="proj-desc">Description <span class="optional">(optional)</span></label>
  <textarea
    id="proj-desc"
    class="field-input"
    rows="2"
    placeholder="A brief description of your site"
    value={description}
    oninput={(e) => onDescriptionChange(e.target.value)}
  ></textarea>
</div>

<div class="field">
  <label class="field-label" for="proj-author">Author <span class="optional">(optional)</span></label>
  <input
    id="proj-author"
    type="text"
    class="field-input"
    placeholder="Your name"
    value={author}
    oninput={(e) => onAuthorChange(e.target.value)}
  />
</div>

<style>
  .section { margin-bottom: 20px; }

  .section-label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: var(--cr-text-muted);
    margin-bottom: 8px;
  }

  :global(.template-grid) {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
  }

  :global(.template-card) {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 14px 8px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-lg);
    background: var(--cr-bg-input);
    color: var(--cr-text-muted);
    cursor: pointer;
    text-align: center;
    font-family: inherit;
    transition: border-color 0.15s, color 0.15s, background 0.15s;
  }

  :global(.template-card:hover) {
    border-color: var(--cr-text-muted);
    color: var(--cr-text);
    background: var(--cr-bg-elevated);
  }

  :global(.template-card[data-state="checked"]) {
    border-color: var(--cr-accent);
    color: var(--cr-accent);
    background: var(--cr-accent-bg);
    box-shadow: 0 0 0 1px var(--cr-accent);
  }

  .template-name {
    font-size: 13px;
    font-weight: 600;
  }

  .template-desc {
    font-size: 10px;
    line-height: 1.3;
    opacity: 0.7;
  }

  .field { margin-bottom: 16px; }

  .field-label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: var(--cr-text-muted);
    margin-bottom: 6px;
  }

  .optional {
    font-weight: 400;
    font-size: 11px;
    color: var(--cr-text-dim);
    font-style: italic;
  }

  .field-input {
    width: 100%;
    padding: 10px 12px;
    font-size: 14px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: var(--cr-bg-input);
    color: var(--cr-text);
    outline: none;
    box-sizing: border-box;
    font-family: inherit;
    resize: none;
  }

  .field-input:focus { border-color: var(--cr-accent); }
</style>
