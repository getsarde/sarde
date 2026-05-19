<script>
  import { ArrowLeft, ArrowRight } from 'lucide-svelte'
  import { projectCreate } from '../api.js'
  import StepNameLocation from './wizard/StepNameLocation.svelte'
  import StepTemplateMetadata from './wizard/StepTemplateMetadata.svelte'
  import StepReview from './wizard/StepReview.svelte'

  let { onCreated, onBack } = $props()

  let step = $state(1)
  let projectName = $state('')
  let location = $state('')
  let template = $state('empty')
  let description = $state('')
  let author = $state('')
  let creating = $state(false)
  let error = $state('')

  const stepLabels = ['Name & Location', 'Template & Details', 'Review']

  let fullPath = $derived(
    location && projectName.trim()
      ? `${location}/${projectName.trim()}`
      : ''
  )

  let canNext = $derived(
    step === 1 ? projectName.trim().length > 0 && location.length > 0
    : step === 2 ? true
    : false
  )

  function next() {
    if (step < 3 && canNext) step++
  }

  async function create() {
    creating = true
    error = ''
    const name = projectName.trim()
    const root = `${location}/${name}`

    try {
      await projectCreate(root, name, template, description, author)
      onCreated(root)
    } catch (e) {
      error = String(e)
    } finally {
      creating = false
    }
  }

  function onKeydown(e) {
    if (e.key === 'Enter' && step < 3 && canNext) next()
    if (e.key === 'Escape') onBack()
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="wizard">
  <div class="wizard-card">
    <h2>Create New Project</h2>

    <!-- Step indicator -->
    <div class="steps">
      {#each stepLabels as label, i}
        {@const num = i + 1}
        <div class="step-item" class:active={step === num} class:done={step > num}>
          <span class="step-dot">{step > num ? '✓' : num}</span>
          <span class="step-label">{label}</span>
        </div>
        {#if i < stepLabels.length - 1}
          <div class="step-line" class:done={step > num}></div>
        {/if}
      {/each}
    </div>

    <!-- Step content -->
    <div class="step-content">
      {#if step === 1}
        <StepNameLocation
          {projectName}
          {location}
          onNameChange={(v) => projectName = v}
          onLocationChange={(v) => location = v}
        />
      {:else if step === 2}
        <StepTemplateMetadata
          {template}
          {description}
          {author}
          onTemplateChange={(v) => template = v}
          onDescriptionChange={(v) => description = v}
          onAuthorChange={(v) => author = v}
        />
      {:else}
        <StepReview
          {projectName}
          {location}
          {fullPath}
          {template}
          {description}
          {author}
          {creating}
          {error}
          oncreate={create}
        />
      {/if}
    </div>

    <!-- Navigation -->
    <div class="nav-row">
      <button class="nav-btn secondary" onclick={onBack}>Cancel</button>
      {#if step > 1}
        <button class="nav-btn secondary" onclick={() => step--}>
          <ArrowLeft size={14} /> Back
        </button>
      {/if}
      {#if step < 3}
        <button class="nav-btn primary" onclick={next} disabled={!canNext}>
          Next <ArrowRight size={14} />
        </button>
      {/if}
    </div>
  </div>
</div>

<style>
  .wizard {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 20px;
    padding: 40px;
  }

  .wizard-card {
    width: 440px;
    max-width: 90vw;
    padding: 32px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-lg);
    background: var(--cr-bg-input);
    box-shadow: var(--cr-shadow-md);
  }

  h2 {
    margin: 0 0 20px;
    font-size: 20px;
    font-weight: 700;
    color: var(--cr-text);
  }

  /* Step indicator */
  .steps {
    display: flex;
    align-items: center;
    gap: 0;
    margin-bottom: 24px;
  }

  .step-item {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  .step-dot {
    width: 22px;
    height: 22px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    font-weight: 700;
    border: 2px solid var(--cr-border);
    color: var(--cr-text-muted);
    background: transparent;
    transition: all 0.2s;
  }

  .step-item.active .step-dot {
    border-color: var(--cr-accent);
    color: var(--cr-accent);
    background: var(--cr-active);
  }

  .step-item.done .step-dot {
    border-color: var(--cr-success);
    color: var(--cr-success);
    background: rgba(166, 227, 161, 0.1);
  }

  .step-label {
    font-size: 11px;
    color: var(--cr-text-muted);
    white-space: nowrap;
  }

  .step-item.active .step-label {
    color: var(--cr-text);
    font-weight: 600;
  }

  .step-item.done .step-label {
    color: var(--cr-success);
  }

  .step-line {
    flex: 1;
    height: 1px;
    background: var(--cr-border);
    margin: 0 8px;
    min-width: 12px;
    transition: background 0.2s;
  }

  .step-line.done {
    background: var(--cr-success);
  }

  /* Step content area */
  .step-content {
    margin-bottom: 16px;
  }

  /* Navigation row */
  .nav-row {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
  }

  .nav-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 10px 18px;
    border: none;
    border-radius: var(--cr-radius);
    font-size: 13px;
    font-weight: 600;
    font-family: inherit;
    cursor: pointer;
    transition: background 0.15s, transform 0.1s;
  }

  .nav-btn.primary {
    background: var(--cr-accent);
    color: var(--cr-bg-base);
  }

  .nav-btn.primary:hover:not(:disabled) {
    background: var(--cr-accent-hover);
    transform: translateY(-1px);
  }

  .nav-btn.primary:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .nav-btn.secondary {
    background: transparent;
    color: var(--cr-text-muted);
    border: 1px solid var(--cr-border);
  }

  .nav-btn.secondary:hover {
    color: var(--cr-text);
    border-color: var(--cr-text-muted);
  }
</style>
