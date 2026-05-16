<script>
  import { onMount } from 'svelte'
  import { Compartment, EditorState, StateEffect, StateField } from '@codemirror/state'
  import {
    EditorView, Decoration, keymap, lineNumbers, highlightActiveLine, drawSelection,
  } from '@codemirror/view'
  import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
  import { markdown } from '@codemirror/lang-markdown'
  import { foldGutter, foldKeymap, bracketMatching, indentOnInput } from '@codemirror/language'
  import { closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
  import { searchKeymap, highlightSelectionMatches } from '@codemirror/search'
  import { coderooCompletions } from '../editor/completions.js'
  import { doc, tabs, addToast } from '../stores/app.svelte.js'
  import { saveContent } from '../api.js'
  import yaml from 'js-yaml'

  let editorContainer = $state(null)
  let view = $state(null)
  let currentFilePath = $state('')
  let saveTimer
  let flashTimer
  let applyingExternal = false

  // Font size — persisted across sessions (Ctrl+= / Ctrl+-)
  let fontSize = $state(parseInt(localStorage.getItem('coderoo-font-size') || '14'))
  const fontComp = new Compartment()

  // Transient line highlight for search navigation
  const flashLineEffect = StateEffect.define()
  const flashClearEffect = StateEffect.define()
  const flashLineDeco = Decoration.line({ class: 'cm-flash-line' })
  const flashLineField = StateField.define({
    create() { return Decoration.none },
    update(deco, tr) {
      deco = deco.map(tr.changes)
      for (const e of tr.effects) {
        if (e.is(flashLineEffect)) deco = Decoration.set([flashLineDeco.range(e.value)])
        else if (e.is(flashClearEffect)) deco = Decoration.none
      }
      return deco
    },
    provide: f => EditorView.decorations.from(f),
  })

  function fontTheme(size) {
    return EditorView.theme({
      '.cm-content': { fontSize: `${size}px` },
      '.cm-gutters': { fontSize: `${size}px` },
    })
  }

  function adjustFontSize(delta) {
    fontSize = Math.max(10, Math.min(24, fontSize + delta))
    localStorage.setItem('coderoo-font-size', String(fontSize))
    view?.dispatch({ effects: fontComp.reconfigure(fontTheme(fontSize)) })
  }

  // Inline helpers used by keymaps
  function wrapAt(v, before, after) {
    const { from, to } = v.state.selection.main
    const selected = v.state.sliceDoc(from, to)
    v.dispatch({ changes: { from, to, insert: before + selected + after } })
    v.focus()
    return true
  }

  function insertLink(v) {
    const { from, to } = v.state.selection.main
    const selected = v.state.sliceDoc(from, to)
    v.dispatch({ changes: { from, to, insert: selected ? `[${selected}](url)` : '[](url)' } })
    v.focus()
    return true
  }

  // When a new file is opened (doc.filePath changes), replace editor contents
  $effect(() => {
    if (view && doc.filePath !== currentFilePath) {
      currentFilePath = doc.filePath
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: doc.content },
      })
    }
  })

  // When PropertiesPanel updates doc.content, sync it into the CodeMirror view
  $effect(() => {
    const _tick = doc.externalUpdate
    if (_tick > 0 && view && doc.filePath === currentFilePath) {
      applyingExternal = true
      const oldSel = view.state.selection.main
      const newLen = doc.content.length
      const anchor = Math.min(oldSel.anchor, newLen)
      const head = Math.min(oldSel.head, newLen)
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: doc.content },
        selection: { anchor, head },
      })
      applyingExternal = false
      scheduleAutoSave()
    }
  })

  // Scroll to target line when set (e.g. from search results)
  $effect(() => {
    const line = doc.targetLine
    if (line > 0 && view) {
      const clamped = Math.min(line, view.state.doc.lines)
      const pos = view.state.doc.line(clamped).from
      view.dispatch({
        selection: { anchor: pos },
        effects: [
          EditorView.scrollIntoView(pos, { y: 'center' }),
          flashLineEffect.of(pos),
        ],
      })
      doc.targetLine = 0
      clearTimeout(flashTimer)
      flashTimer = setTimeout(() => {
        view?.dispatch({ effects: flashClearEffect.of(null) })
      }, 1200)
    }
  })

  // Insert text at cursor when set (e.g. from media panel)
  $effect(() => {
    const text = doc.insertText
    if (text && view) {
      const cursor = view.state.selection.main.head
      view.dispatch({
        changes: { from: cursor, insert: text },
        selection: { anchor: cursor + text.length },
      })
      doc.insertText = ''
      doc.dirty = true
      scheduleAutoSave()
    }
  })

  // Auto-save: write to disk 5s after last edit
  async function saveFile() {
    if (!doc.contentPath || !doc.dirty) return
    doc.pendingSave++
    try {
      const { frontmatter, body } = splitMarkdown(doc.content)
      await saveContent(doc.contentPath, frontmatter, body)
      doc.dirty = false
      const activeTab = tabs.items.find(t => t.id === tabs.activeId)
      if (activeTab) {
        activeTab.dirty = false
        activeTab.cachedContent = doc.content
      }
    } catch (e) {
      console.error('Failed to save file:', e)
      addToast('error', `Save failed: ${e}`)
    } finally {
      setTimeout(() => { if (doc.pendingSave > 0) doc.pendingSave-- }, 1500)
    }
  }

  function splitMarkdown(content) {
    const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?/)
    if (!match) return { frontmatter: {}, body: content }
    const parsed = yaml.load(match[1]) ?? {}
    if (typeof parsed !== 'object' || Array.isArray(parsed)) {
      throw new Error('Frontmatter must be a YAML object')
    }
    return { frontmatter: parsed, body: content.slice(match[0].length) }
  }

  function scheduleAutoSave() {
    clearTimeout(saveTimer)
    saveTimer = setTimeout(saveFile, 5000)
  }

  onMount(() => {
    window.addEventListener('coderoo:save', saveFile)

    const state = EditorState.create({
      doc: doc.content,
      extensions: [
        lineNumbers(),
        foldGutter(),
        highlightActiveLine(),
        drawSelection(),
        indentOnInput(),
        bracketMatching(),
        closeBrackets(),
        coderooCompletions(),
        history(),
        markdown(),
        highlightSelectionMatches(),
        fontComp.of(fontTheme(fontSize)),
        flashLineField,
        keymap.of([
          { key: 'Mod-s', run: () => { saveFile(); return true } },
          { key: 'Mod-b', run: (v) => wrapAt(v, '**', '**') },
          { key: 'Mod-i', run: (v) => wrapAt(v, '_', '_') },
          { key: 'Mod-k', run: (v) => insertLink(v) },
          { key: 'Mod-=', run: () => { adjustFontSize(1); return true } },
          { key: 'Mod--', run: () => { adjustFontSize(-1); return true } },
          ...closeBracketsKeymap,
          ...defaultKeymap,
          ...historyKeymap,
          ...searchKeymap,
          ...foldKeymap,
        ]),
        EditorView.updateListener.of(update => {
          if (update.docChanged && !applyingExternal) {
            doc.content = update.state.doc.toString()
            doc.dirty = true
            // Sync to active tab cache so switching tabs preserves unsaved edits
            const activeTab = tabs.items.find(t => t.id === tabs.activeId)
            if (activeTab) {
              activeTab.dirty = true
              activeTab.cachedContent = doc.content
            }
            const words = doc.content.split(/\s+/).filter(w => w).length
            doc.wordCount = words
            doc.readingTime = Math.max(1, Math.ceil(words / 250))
            scheduleAutoSave()
          }
          if (update.selectionSet) {
            const pos = update.state.selection.main.head
            const line = update.state.doc.lineAt(pos)
            doc.cursorLine = line.number
            doc.cursorCol = pos - line.from + 1
          }
        }),
        EditorView.theme({
          '&': { height: '100%' },
          '.cm-content': {
            fontFamily: 'var(--cr-font-mono)',
            padding: '16px',
          },
          '.cm-gutters': {
            background: 'var(--cr-bg-surface)',
            color: 'var(--cr-text-muted)',
            border: 'none',
          },
          '.cm-activeLineGutter': { background: 'var(--cr-bg-elevated)' },
          '.cm-activeLine': { background: 'rgba(99, 102, 241, 0.08)' },
          '.cm-cursor': { borderLeftColor: 'var(--cr-accent)' },
          '.cm-selectionBackground': { background: 'rgba(99, 102, 241, 0.2) !important' },
          '.cm-foldPlaceholder': {
            background: 'var(--cr-bg-elevated)',
            color: 'var(--cr-text-muted)',
            border: 'none',
            borderRadius: '3px',
            padding: '0 4px',
          },
          '.cm-matchingBracket': { background: 'rgba(99, 102, 241, 0.25)', outline: 'none' },
          // Autocomplete dropdown
          '.cm-tooltip.cm-tooltip-autocomplete': {
            background: 'var(--cr-bg-surface)',
            border: '1px solid var(--cr-border)',
            borderRadius: 'var(--cr-radius)',
            boxShadow: '0 8px 32px rgba(0,0,0,0.5)',
            overflow: 'hidden',
          },
          '.cm-tooltip-autocomplete > ul': {
            fontFamily: 'var(--cr-font-ui)',
            maxHeight: '280px',
          },
          '.cm-tooltip-autocomplete > ul > li': {
            padding: '5px 12px',
            lineHeight: '1.5',
          },
          '.cm-tooltip-autocomplete > ul > li[aria-selected]': {
            background: 'var(--cr-active)',
            color: 'var(--cr-text)',
          },
          '.cm-completionLabel': {
            color: 'var(--cr-text)',
            fontSize: '13px',
          },
          '.cm-completionDetail': {
            color: 'var(--cr-text-muted)',
            fontSize: '11px',
            marginLeft: '8px',
          },
          '.cm-completionInfo': {
            background: 'var(--cr-bg-elevated)',
            border: '1px solid var(--cr-border)',
            borderRadius: 'var(--cr-radius)',
            padding: '6px 10px',
            fontSize: '12px',
            color: 'var(--cr-text-muted)',
            maxWidth: '260px',
          },
        }, { dark: true }),
      ],
    })

    view = new EditorView({ state, parent: editorContainer })

    return () => {
      clearTimeout(saveTimer)
      clearTimeout(flashTimer)
      window.removeEventListener('coderoo:save', saveFile)
      view?.destroy()
    }
  })

  /** Return the underlying EditorView instance. */
  export function getView() { return view }

  /** Insert text at the current cursor position. */
  export function insertText(text) {
    if (!view) return
    const { from, to } = view.state.selection.main
    view.dispatch({ changes: { from, to, insert: text } })
    view.focus()
  }

  /** Wrap the current selection with before/after strings. */
  export function wrapSelection(before, after) {
    if (!view) return
    wrapAt(view, before, after)
  }
</script>

<div class="code-editor" bind:this={editorContainer}></div>

<style>
  .code-editor {
    flex: 1;
    overflow: hidden;
  }
  .code-editor :global(.cm-editor) {
    height: 100%;
  }
  .code-editor :global(.cm-scroller) {
    overflow: auto;
  }
  .code-editor :global(.cm-flash-line) {
    background-color: rgba(250, 204, 21, 0.28);
    animation: cm-flash-fade 1.2s ease-out forwards;
  }
  @keyframes cm-flash-fade {
    0%   { background-color: rgba(250, 204, 21, 0.45); }
    100% { background-color: rgba(250, 204, 21, 0); }
  }
</style>
