<script>
  import { onMount } from 'svelte'
  import yaml from 'js-yaml'
  import { EditorState } from '@codemirror/state'
  import { EditorView, keymap, drawSelection } from '@codemirror/view'
  import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
  import { closeBrackets } from '@codemirror/autocomplete'
  import { yaml as yamlLang } from '@codemirror/lang-yaml'

  let { value = '', onchange, onerror } = $props()

  let containerEl = $state(null)
  let view = null
  let selfUpdate = false
  let inboundSync = false

  onMount(() => {
    const state = EditorState.create({
      doc: value,
      extensions: [
        yamlLang(),
        history(),
        keymap.of([...defaultKeymap, ...historyKeymap]),
        EditorView.lineWrapping,
        drawSelection(),
        closeBrackets(),
        EditorView.updateListener.of(handleUpdate),
        EditorView.theme({
          '&': { height: '100%', fontSize: '12px' },
          '.cm-content': {
            fontFamily: 'var(--cr-font-mono)',
            padding: '8px',
          },
          '.cm-cursor': { borderLeftColor: 'var(--cr-accent)' },
          '.cm-selectionBackground': { background: 'rgba(99, 102, 241, 0.2) !important' },
          '.cm-scroller': { overflow: 'auto' },
        }, { dark: true }),
      ],
    })

    view = new EditorView({ state, parent: containerEl })

    return () => { view?.destroy(); view = null }
  })

  function handleUpdate(update) {
    if (!update.docChanged) return
    if (inboundSync) return

    selfUpdate = true
    const text = update.state.doc.toString()

    try {
      yaml.load(text)
      onchange?.(text)
      onerror?.(null)
    } catch (e) {
      onerror?.(e.message?.split('\n')[0] ?? 'Invalid YAML')
    }

    // Reset flag after microtask so the inbound $effect skips this change.
    queueMicrotask(() => { selfUpdate = false })
  }

  // Inbound sync: update CM when value prop changes from outside.
  $effect(() => {
    const v = value
    if (!view || selfUpdate) return

    const current = view.state.doc.toString()
    if (v !== current) {
      inboundSync = true
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: v },
      })
      inboundSync = false
    }
  })
</script>

<div class="yaml-editor" bind:this={containerEl}></div>

<style>
  .yaml-editor {
    flex: 1;
    overflow: hidden;
    background: var(--cr-bg-input);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-sm);
  }

  .yaml-editor :global(.cm-editor) {
    height: 100%;
    background: transparent;
  }

  .yaml-editor :global(.cm-focused) {
    outline: none;
  }

  .yaml-editor :global(.cm-scroller) {
    font-size: 12px;
  }
</style>
