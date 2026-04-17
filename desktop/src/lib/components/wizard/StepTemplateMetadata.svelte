<script>
  import { FileText, BookOpen, Newspaper } from 'lucide-svelte'

  let { template = 'empty', description = '', author = '', onTemplateChange, onDescriptionChange, onAuthorChange } = $props()

  const templates = [
    { id: 'empty', name: 'Empty', desc: 'Blank slate with a single welcome page', icon: FileText },
    { id: 'blog', name: 'Blog', desc: 'Posts collection with RSS feed enabled', icon: Newspaper },
    { id: 'docs', name: 'Docs', desc: 'Documentation site with sidebar navigation', icon: BookOpen },
  ]
</script>

<div class="section">
  <label class="section-label">Template</label>
  <div class="template-grid">
    {#each templates as t}
      {@const Icon = t.icon}
      <button
        class="template-card"
        class:active={template === t.id}
        onclick={() => onTemplateChange(t.id)}
      >
        <Icon size={20} />
        <span class="template-name">{t.name}</span>
        <span class="template-desc">{t.desc}</span>
      </button>
    {/each}
  </div>
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
  .section { margin-bottom: 18px; }

  .section-label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: var(--color-text-muted, #6c7086);
    margin-bottom: 8px;
  }

  .template-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
  }

  .template-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 14px 8px;
    border: 1px solid var(--color-border, #313244);
    border-radius: 10px;
    background: var(--color-input, #11111b);
    color: var(--color-text-muted, #6c7086);
    cursor: pointer;
    text-align: center;
    font-family: inherit;
    transition: border-color 0.15s, color 0.15s;
  }

  .template-card:hover {
    border-color: var(--color-text-muted, #6c7086);
    color: var(--color-text, #cdd6f4);
  }

  .template-card.active {
    border-color: var(--color-accent, #89b4fa);
    color: var(--color-accent, #89b4fa);
    background: rgba(137, 180, 250, 0.06);
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

  .field { margin-bottom: 18px; }

  .field-label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: var(--color-text-muted, #6c7086);
    margin-bottom: 6px;
  }

  .optional {
    font-weight: 400;
    opacity: 0.6;
  }

  .field-input {
    width: 100%;
    padding: 10px 12px;
    font-size: 14px;
    border: 1px solid var(--color-border, #313244);
    border-radius: 8px;
    background: var(--color-input, #11111b);
    color: var(--color-text, #cdd6f4);
    outline: none;
    box-sizing: border-box;
    font-family: inherit;
    resize: vertical;
  }

  .field-input:focus { border-color: var(--color-accent, #89b4fa); }
</style>
