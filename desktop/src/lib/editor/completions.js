/**
 * CodeMirror 6 completion sources for Coderoo's markdown editor.
 *
 * Provides:
 *  1. Slash commands  — triggered when the line starts with "/"
 *  2. Block picker    — triggered when the line starts with ":::"
 *  3. Fenced code lang— triggered after opening fence "```"
 *  4. Frontmatter     — triggered for keys inside "---" blocks
 */

import { autocompletion, snippetCompletion } from '@codemirror/autocomplete'

// ---------------------------------------------------------------------------
// Data
// ---------------------------------------------------------------------------

/** Basic markdown shortcuts exposed as /commands */
const BASIC_COMMANDS = [
  { label: '/heading1',   displayLabel: 'Heading 1',       detail: 'H1',     template: '# ${title}',                                             info: 'Top-level heading' },
  { label: '/heading2',   displayLabel: 'Heading 2',       detail: 'H2',     template: '## ${title}',                                            info: 'Second-level heading' },
  { label: '/heading3',   displayLabel: 'Heading 3',       detail: 'H3',     template: '### ${title}',                                           info: 'Third-level heading' },
  { label: '/bold',       displayLabel: 'Bold',            detail: 'Inline', template: '**${text}**',                                            info: 'Bold text' },
  { label: '/italic',     displayLabel: 'Italic',          detail: 'Inline', template: '_${text}_',                                              info: 'Italic text' },
  { label: '/link',       displayLabel: 'Link',            detail: 'Inline', template: '[${label}](${url})',                                      info: 'Hyperlink' },
  { label: '/image',      displayLabel: 'Image',           detail: 'Inline', template: '![${alt}](${url})',                                       info: 'Inline image' },
  { label: '/blockquote', displayLabel: 'Blockquote',      detail: 'Block',  template: '> ${text}',                                              info: 'Quoted text' },
  { label: '/hr',         displayLabel: 'Horizontal Rule', detail: 'Block',  template: '---',                                                    info: 'Horizontal divider' },
  { label: '/table',      displayLabel: 'Table',           detail: 'Block',  template: '| ${Col 1} | Col 2 |\n|---------|-------|\n| Cell    | Cell  |', info: 'Markdown table' },
  { label: '/ul',         displayLabel: 'Unordered List',  detail: 'Block',  template: '- ${item}',                                              info: 'Bulleted list' },
  { label: '/ol',         displayLabel: 'Ordered List',    detail: 'Block',  template: '1. ${item}',                                             info: 'Numbered list' },
  { label: '/code',       displayLabel: 'Inline Code',     detail: 'Inline', template: '`${code}`',                                              info: 'Inline code span' },
  { label: '/codeblock',  displayLabel: 'Code Block',      detail: 'Block',  template: '```${lang}\n${code}\n```',                               info: 'Fenced code block' },
  { label: '/frontmatter',displayLabel: 'Frontmatter',     detail: 'Meta',   template: '---\ntitle: ${Title}\ndescription: ${Description}\n---', info: 'YAML frontmatter header' },
]

/**
 * Coderoo custom block extensions.
 */
const BLOCK_EXTENSIONS = [
  // Content
  { name: 'aside',         category: 'Content',    info: 'Callout box (note, tip, caution, danger)',     template: ':::aside[${note}]\n${content}\n:::' },
  { name: 'details',       category: 'Content',    info: 'Collapsible details block',                    template: ':::details[${Summary}]\n${content}\n:::' },
  { name: 'spoiler',       category: 'Content',    info: 'Hidden content revealed on click',             template: ':::spoiler\n${hidden content}\n:::' },
  { name: 'annotation',    category: 'Content',    info: 'Annotated content block',                      template: ':::annotation\n${content}\n:::' },
  { name: 'highlight',     category: 'Content',    info: 'Highlighted section',                          template: ':::highlight\n${content}\n:::' },
  // Structure
  { name: 'steps',         category: 'Structure',  info: 'Numbered steps layout',                        template: ':::steps\n## ${Step 1}\n${content}\n\n## Step 2\ncontent\n:::' },
  { name: 'tabs',          category: 'Structure',  info: 'Tabbed content panel',                         template: ':::tabs\n== ${Tab 1}\n${content}\n\n== Tab 2\ncontent\n:::' },
  { name: 'card',          category: 'Structure',  info: 'Single content card',                          template: ':::card{title="${Title}"}\n${content}\n:::' },
  { name: 'card-grid',     category: 'Structure',  info: 'Grid of cards',                                template: ':::card-grid\n:::card{title="${Card 1}"}\n${content}\n:::\n:::card{title="Card 2"}\ncontent\n:::\n:::' },
  { name: 'timeline',      category: 'Structure',  info: 'Chronological timeline',                       template: ':::timeline\n## ${Event title}\n${description}\n\n## Event 2\ndescription\n:::' },
  // Code
  { name: 'code-group',    category: 'Code',       info: 'Tabbed code examples',                         template: ':::code-group\n```${js} [label]\n${code}\n```\n```ts [label]\ncode\n```\n:::' },
  { name: 'code-diff',     category: 'Code',       info: 'Diff-highlighted code block',                  template: ':::code-diff\n```${js}\n- removed line\n+ added line\n```\n:::' },
  { name: 'terminal',      category: 'Code',       info: 'Styled terminal output',                       template: ':::terminal\n```\n$ ${command}\n```\n:::' },
  { name: 'mermaid',       category: 'Code',       info: 'Mermaid diagram',                              template: ':::mermaid\n```\nflowchart LR\n    ${A} --> B\n```\n:::' },
  // Media
  { name: 'figure',        category: 'Media',      info: 'Image with caption',                           template: ':::figure[${Caption}]\n![${alt}](${url})\n:::' },
  { name: 'gallery',       category: 'Media',      info: 'Image gallery grid',                           template: ':::gallery\n![${alt 1}](${url1})\n![alt 2](url2)\n:::' },
  { name: 'image-compare', category: 'Media',      info: 'Before/after image slider',                    template: ':::image-compare\n![${Before}](${before.jpg})\n![After](after.jpg)\n:::' },
  { name: 'video',         category: 'Media',      info: 'Embedded video player',                        template: ':::video{src="${url}"}\n:::' },
  // Navigation / UI
  { name: 'file-tree',     category: 'Navigation', info: 'File tree diagram',                            template: ':::file-tree\n- ${src/}\n  - index.js\n- package.json\n:::' },
  { name: 'link-button',   category: 'Navigation', info: 'Styled link button',                           template: ':::link-button{href="${url}"}\n${Label}\n:::' },
  { name: 'link-card',     category: 'Navigation', info: 'Preview link card',                            template: ':::link-card{href="${url}" title="${Title}"}\n${description}\n:::' },
  { name: 'badge',         category: 'UI',         info: 'Inline badge/tag (success, warning, danger, info)', template: ':::badge{type="${success}"}\n${Label}\n:::' },
  { name: 'kbd',           category: 'UI',         info: 'Keyboard shortcut display',                    template: ':::kbd\n${Ctrl+K}\n:::' },
  // Math
  { name: 'math',          category: 'Math',       info: 'LaTeX math block',                             template: ':::math\n$$\n${E = mc^2}\n$$\n:::' },
]

/** Languages supported by Chroma (Go syntax highlighter) */
const CODE_LANGUAGES = [
  'bash', 'sh', 'shell', 'zsh',
  'javascript', 'js', 'jsx',
  'typescript', 'ts', 'tsx',
  'python', 'py',
  'go',
  'rust',
  'java',
  'kotlin',
  'swift',
  'c', 'cpp',
  'csharp',
  'php',
  'ruby',
  'html',
  'css', 'scss', 'sass', 'less',
  'json', 'yaml', 'toml', 'xml',
  'sql', 'mysql', 'postgresql',
  'markdown',
  'dockerfile',
  'makefile',
  'nginx',
  'vim',
  'lua',
  'perl',
  'r',
  'scala',
  'haskell',
  'elixir',
  'erlang',
  'clojure',
  'dart',
  'svelte',
  'vue',
  'graphql',
  'proto',
  'diff',
  'ini',
  'env',
  'powershell',
  'bat',
  'terraform',
  'zig',
]

/** Common frontmatter keys */
const FRONTMATTER_KEYS = [
  { label: 'title',       detail: 'string',  info: 'Page title',                template: 'title: ${Title}' },
  { label: 'description', detail: 'string',  info: 'Page description',           template: 'description: ${Description}' },
  { label: 'date',        detail: 'date',    info: 'Publication date YYYY-MM-DD',template: 'date: ${2025-01-01}' },
  { label: 'author',      detail: 'string',  info: 'Author name',                template: 'author: ${Name}' },
  { label: 'draft',       detail: 'boolean', info: 'Hide from production build',  template: 'draft: ${true}' },
  { label: 'tags',        detail: 'array',   info: 'Tag list',                   template: 'tags:\n  - ${tag}' },
  { label: 'categories',  detail: 'array',   info: 'Category list',              template: 'categories:\n  - ${category}' },
  { label: 'slug',        detail: 'string',  info: 'URL slug override',           template: 'slug: ${my-page}' },
  { label: 'image',       detail: 'string',  info: 'Cover image URL',             template: 'image: ${/images/cover.jpg}' },
  { label: 'layout',      detail: 'string',  info: 'Layout template',             template: 'layout: ${default}' },
  { label: 'sidebar',     detail: 'boolean', info: 'Show sidebar',               template: 'sidebar: ${true}' },
  { label: 'toc',         detail: 'boolean', info: 'Show table of contents',      template: 'toc: ${true}' },
]

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function isInFrontmatter(state, pos) {
  const text = state.doc.sliceString(0, pos)
  if (!text.startsWith('---\n')) return false
  const fmEnd = text.search(/\n---(\n|$)/)
  if (fmEnd === -1) return true       // no closing delimiter yet — still inside
  return pos < fmEnd                   // stop before the \n that starts ---
}

// ---------------------------------------------------------------------------
// Completion Sources
// ---------------------------------------------------------------------------

/**
 * Slash commands — triggered when the line starts with "/"
 */
export function slashCommandSource(context) {
  if (isInFrontmatter(context.state, context.pos)) return null

  const line = context.state.doc.lineAt(context.pos)
  const lineText = context.state.sliceDoc(line.from, context.pos)

  const match = lineText.match(/^(\s*)(\/\S*)$/)
  if (!match) return null

  const slashStart = line.from + match[1].length

  const options = [
    // Basic markdown
    ...BASIC_COMMANDS.map(cmd =>
      snippetCompletion(cmd.template, {
        label: cmd.label,
        displayLabel: cmd.displayLabel,
        detail: cmd.detail,
        info: cmd.info,
        type: 'keyword',
        boost: 1,
      })
    ),
    // Extension blocks
    ...BLOCK_EXTENSIONS.map(ext =>
      snippetCompletion(ext.template, {
        label: `/${ext.name}`,
        displayLabel: ext.name,
        detail: ext.category,
        info: ext.info,
        type: 'namespace',
      })
    ),
  ]

  return {
    from: slashStart,
    options,
    validFor: /^\/\S*$/,
  }
}

/**
 * Block picker — triggered when the line starts with ":::"
 */
export function blockPickerSource(context) {
  const line = context.state.doc.lineAt(context.pos)
  const lineText = context.state.sliceDoc(line.from, context.pos)

  if (!lineText.match(/^:::([\w-]*)$/)) return null

  const options = BLOCK_EXTENSIONS.map(ext =>
    snippetCompletion(ext.template, {
      label: `:::${ext.name}`,
      displayLabel: ext.name,
      detail: ext.category,
      info: ext.info,
      type: 'namespace',
    })
  )

  return {
    from: line.from,
    options,
    validFor: /^:::\S*$/,
  }
}

/**
 * Fenced code language picker — triggered after opening "```"
 */
export function codeFenceSource(context) {
  const line = context.state.doc.lineAt(context.pos)
  const lineText = context.state.sliceDoc(line.from, context.pos)

  if (!lineText.match(/^```(\w*)$/)) return null

  const options = CODE_LANGUAGES.map(lang => ({ label: lang, type: 'keyword' }))

  return {
    from: line.from + 3,
    options,
    validFor: /^\w*$/,
  }
}

/**
 * Frontmatter key autocomplete — triggered inside "---" YAML blocks
 */
export function frontmatterSource(context) {
  if (!isInFrontmatter(context.state, context.pos)) return null

  const line = context.state.doc.lineAt(context.pos)
  const lineText = context.state.sliceDoc(line.from, context.pos)

  // Only complete at key position (start of line, no colon yet)
  if (!lineText.match(/^[\w-]*$/)) return null

  const options = FRONTMATTER_KEYS.map(k =>
    snippetCompletion(k.template, {
      label: k.label,
      detail: k.detail,
      info: k.info,
      type: 'property',
    })
  )

  return {
    from: line.from,
    options,
    validFor: /^[\w-]*$/,
  }
}

// ---------------------------------------------------------------------------
// Extension bundle
// ---------------------------------------------------------------------------

export function coderooCompletions() {
  return autocompletion({
    override: [
      slashCommandSource,
      blockPickerSource,
      codeFenceSource,
      frontmatterSource,
    ],
    defaultKeymap: true,
    activateOnTyping: true,
    icons: false,
  })
}
