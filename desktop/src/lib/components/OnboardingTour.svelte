<script>
  import { Folder, Search, Eye, PencilLine, Terminal, Keyboard, ArrowRight, X } from 'lucide-svelte'

  let { onDismiss } = $props()

  const STEPS = [
    {
      icon: Folder,
      title: 'Explorer',
      desc: 'Browse and manage your content files. Right-click for context menus, Ctrl+Click to multi-select.',
    },
    {
      icon: Search,
      title: 'Find & Replace',
      desc: 'Search across all files with regex support. Toggle replace mode to update content project-wide.',
    },
    {
      icon: PencilLine,
      title: 'Properties Panel',
      desc: 'Edit frontmatter visually via schema-driven forms, or switch to raw YAML mode.',
    },
    {
      icon: Eye,
      title: 'Live Preview',
      desc: 'Split or full-screen markdown preview. Test responsive layouts with viewport presets.',
    },
    {
      icon: Terminal,
      title: 'Build & Preview',
      desc: 'Start the dev server from the toolbar. Build logs are clickable — errors jump to the source line.',
    },
    {
      icon: Keyboard,
      title: 'Keyboard Shortcuts',
      desc: 'Ctrl+P: Command Palette, Ctrl+S: Save, Ctrl+\\: Toggle sidebar, Ctrl+,: Settings, Ctrl+F: Find in file.',
    },
  ]

  let currentStep = $state(0)
  let step = $derived(STEPS[currentStep])

  function next() {
    if (currentStep < STEPS.length - 1) currentStep++
    else onDismiss()
  }

  function prev() {
    if (currentStep > 0) currentStep--
  }
</script>

<div class="tour-overlay" role="dialog" aria-label="Welcome tour" aria-modal="true">
  <div class="tour-card">
    <button class="tour-close" onclick={onDismiss} aria-label="Close tour"><X size={16} /></button>

    <div class="tour-content">
      <div class="tour-icon"><step.icon size={28} /></div>
      <h2 class="tour-title">{step.title}</h2>
      <p class="tour-desc">{step.desc}</p>
    </div>

    <div class="tour-dots">
      {#each STEPS as _, i}
        <button
          class="tour-dot"
          class:active={i === currentStep}
          onclick={() => currentStep = i}
          aria-label="Step {i + 1}"
        ></button>
      {/each}
    </div>

    <div class="tour-footer">
      <button class="tour-btn skip" onclick={onDismiss}>Skip</button>
      <div class="tour-nav">
        {#if currentStep > 0}
          <button class="tour-btn" onclick={prev}>Back</button>
        {/if}
        <button class="tour-btn primary" onclick={next}>
          {currentStep < STEPS.length - 1 ? 'Next' : 'Get Started'}
          {#if currentStep < STEPS.length - 1}
            <ArrowRight size={14} />
          {/if}
        </button>
      </div>
    </div>
  </div>
</div>

<style>
  .tour-overlay {
    position: fixed;
    inset: 0;
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
  }

  .tour-card {
    position: relative;
    width: 400px;
    max-width: 90vw;
    padding: 32px;
    background: var(--cr-bg-base);
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius-lg);
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4);
  }

  .tour-close {
    position: absolute;
    top: 12px;
    right: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border: none;
    border-radius: var(--cr-radius-sm);
    background: transparent;
    color: var(--cr-text-muted);
    cursor: pointer;
  }

  .tour-close:hover {
    color: var(--cr-text);
    background: var(--cr-hover);
  }

  .tour-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: 12px;
    margin-bottom: 24px;
  }

  .tour-icon {
    width: 56px;
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 14px;
    background: var(--cr-active);
    color: var(--cr-accent);
  }

  .tour-title {
    margin: 0;
    font-size: 20px;
    font-weight: 700;
    color: var(--cr-text);
  }

  .tour-desc {
    margin: 0;
    font-size: 14px;
    color: var(--cr-text-muted);
    line-height: 1.6;
    max-width: 320px;
  }

  .tour-dots {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    margin-bottom: 20px;
  }

  .tour-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    border: none;
    background: var(--cr-border);
    cursor: pointer;
    padding: 0;
    transition: background 0.15s, transform 0.15s;
  }

  .tour-dot.active {
    background: var(--cr-accent);
    transform: scale(1.3);
  }

  .tour-dot:hover:not(.active) {
    background: var(--cr-text-muted);
  }

  .tour-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .tour-nav {
    display: flex;
    gap: 8px;
  }

  .tour-btn {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 8px 16px;
    border: 1px solid var(--cr-border);
    border-radius: var(--cr-radius);
    background: transparent;
    color: var(--cr-text-muted);
    font-size: 13px;
    font-weight: 500;
    font-family: inherit;
    cursor: pointer;
  }

  .tour-btn:hover {
    color: var(--cr-text);
    border-color: var(--cr-text-muted);
  }

  .tour-btn.skip {
    border-color: transparent;
  }

  .tour-btn.skip:hover {
    border-color: transparent;
    background: var(--cr-hover);
  }

  .tour-btn.primary {
    background: var(--cr-accent);
    border-color: var(--cr-accent);
    color: #fff;
    font-weight: 600;
  }

  .tour-btn.primary:hover {
    filter: brightness(1.1);
  }
</style>
