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

  function back() {
    if (step > 1) step--
    else onBack()
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
    if (e.key === 'Escape') back()
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="wizard">
  <button class="back-link" onclick={back}>
    <ArrowLeft size={16} /> {step > 1 ? 'Back' : 'Home'}
  </button>

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

    <!-- Navigation buttons (steps 1-2 only; step 3 has its own Create button) -->
    {#if step < 3}
      <div class="nav-row">
        {#if step > 1}
          <button class="nav-btn secondary" onclick={() => step--}>
            <ArrowLeft size={14} /> Back
          </button>
        {:else}
          <div></div>
        {/if}
        <button class="nav-btn primary" onclick={next} disabled={!canNext}>
          Next <ArrowRight size={14} />
        </button>
      </div>
    {/if}
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

  .back-link {
    position: absolute;
    top: 20px;
    left: 20px;
    display: flex;
    align-items: center;
    gap: 6px;
    border: none;
    background: transparent;
    color: var(--color-text-muted, #6c7086);
    font-size: 13px;
    cursor: pointer;
    padding: 6px 10px;
    border-radius: 6px;
  }

  .back-link:hover {
    color: var(--color-text, #cdd6f4);
    background: var(--color-hover, rgba(255, 255, 255, 0.06));
  }

  .wizard-card {
    width: 440px;
    max-width: 90vw;
    padding: 32px;
    border: 1px solid var(--color-border, #313244);
    border-radius: 14px;
    background: var(--color-surface-alt, #181825);
  }

  h2 {
    margin: 0 0 20px;
    font-size: 20px;
    font-weight: 700;
    color: var(--color-text, #cdd6f4);
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
    border: 2px solid var(--color-border, #313244);
    color: var(--color-text-muted, #6c7086);
    background: transparent;
    transition: all 0.2s;
  }

  .step-item.active .step-dot {
    border-color: var(--color-accent, #89b4fa);
    color: var(--color-accent, #89b4fa);
    background: rgba(137, 180, 250, 0.1);
  }

  .step-item.done .step-dot {
    border-color: var(--color-success, #a6e3a1);
    color: var(--color-success, #a6e3a1);
    background: rgba(166, 227, 161, 0.1);
  }

  .step-label {
    font-size: 11px;
    color: var(--color-text-muted, #6c7086);
    white-space: nowrap;
  }

  .step-item.active .step-label {
    color: var(--color-text, #cdd6f4);
    font-weight: 600;
  }

  .step-item.done .step-label {
    color: var(--color-success, #a6e3a1);
  }

  .step-line {
    flex: 1;
    height: 1px;
    background: var(--color-border, #313244);
    margin: 0 8px;
    min-width: 12px;
    transition: background 0.2s;
  }

  .step-line.done {
    background: var(--color-success, #a6e3a1);
  }

  /* Step content area */
  .step-content {
    margin-bottom: 4px;
  }

  /* Navigation row */
  .nav-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 8px;
  }

  .nav-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 10px 18px;
    border: none;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 600;
    font-family: inherit;
    cursor: pointer;
    transition: background 0.15s, transform 0.1s;
  }

  .nav-btn.primary {
    background: var(--color-accent, #89b4fa);
    color: var(--color-surface, #1e1e2e);
  }

  .nav-btn.primary:hover:not(:disabled) {
    background: #74c7ec;
    transform: translateY(-1px);
  }

  .nav-btn.primary:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .nav-btn.secondary {
    background: transparent;
    color: var(--color-text-muted, #6c7086);
    border: 1px solid var(--color-border, #313244);
  }

  .nav-btn.secondary:hover {
    color: var(--color-text, #cdd6f4);
    border-color: var(--color-text-muted, #6c7086);
  }
</style>
