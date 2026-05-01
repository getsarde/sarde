<script>
  import { ui, siteConfig, addToast } from '../stores/app.svelte.js'
  import { deploy as apiDeploy } from '../api.js'
  import { X, Loader, Rocket, CheckCircle, AlertCircle } from 'lucide-svelte'

  let status = $state('idle') // 'idle' | 'deploying' | 'success' | 'error'
  let resultMessage = $state('')

  let provider = $derived(siteConfig.data?.deploy?.provider)
  let configured = $derived(!!provider)

  // Reset state when modal opens
  $effect(() => {
    if (ui.deployOpen) {
      status = 'idle'
      resultMessage = ''
    }
  })

  function close() {
    ui.deployOpen = false
  }

  function onKeydown(e) {
    if (e.key === 'Escape') close()
  }

  function openDeploySettings() {
    close()
    ui.settingsSection = 'deploy'
    ui.settingsOpen = true
  }

  function providerLabel(p) {
    const labels = {
      github: 'GitHub Pages',
      netlify: 'Netlify',
      cloudflare: 'Cloudflare Pages',
      vercel: 'Vercel',
      custom: 'Custom',
    }
    return labels[p] || p
  }

  async function doDeploy() {
    status = 'deploying'
    resultMessage = ''
    try {
      const resp = await apiDeploy()
      status = 'success'
      resultMessage = `Deployed to ${providerLabel(resp?.data?.provider || provider)}`
      addToast('success', resultMessage)
    } catch (e) {
      status = 'error'
      resultMessage = e.message || 'Deployment failed'
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="modal-overlay" onclick={(e) => { if (e.target === e.currentTarget) close() }} onkeydown={onKeydown} role="dialog" aria-modal="true" aria-label="Deploy" tabindex="-1">
  <div class="modal-content">
    <div class="modal-header">
      <h2>Deploy</h2>
      <button class="modal-close" onclick={close} title="Close">
        <X size={16} />
      </button>
    </div>

    <div class="modal-body">
      {#if !configured}
        <div class="deploy-state">
          <div class="deploy-icon muted">
            <Rocket size={32} />
          </div>
          <p class="deploy-message">No deploy provider configured.</p>
          <button class="btn btn-primary" onclick={openDeploySettings}>
            Configure in Settings
          </button>
        </div>

      {:else if status === 'idle'}
        <div class="deploy-state">
          <div class="deploy-icon">
            <Rocket size={32} />
          </div>
          <div class="provider-badge">{providerLabel(provider)}</div>
          <button class="btn btn-primary" onclick={doDeploy}>
            Deploy Now
          </button>
        </div>

      {:else if status === 'deploying'}
        <div class="deploy-state">
          <div class="deploy-icon spinning">
            <Loader size={32} />
          </div>
          <p class="deploy-message">Deploying to {providerLabel(provider)}...</p>
        </div>

      {:else if status === 'success'}
        <div class="deploy-state">
          <div class="deploy-icon success">
            <CheckCircle size={32} />
          </div>
          <p class="deploy-message">{resultMessage}</p>
          <button class="btn btn-secondary" onclick={close}>Close</button>
        </div>

      {:else if status === 'error'}
        <div class="deploy-state">
          <div class="deploy-icon error">
            <AlertCircle size={32} />
          </div>
          <p class="deploy-message error-text">{resultMessage}</p>
          <div class="btn-group">
            <button class="btn btn-primary" onclick={() => { status = 'idle' }}>Try Again</button>
            <button class="btn btn-secondary" onclick={close}>Close</button>
          </div>
        </div>
      {/if}
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
    width: 400px;
    max-width: 90vw;
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
    padding: 32px 20px;
  }

  .deploy-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    text-align: center;
  }

  .deploy-icon {
    color: var(--color-accent, #6366f1);
  }

  .deploy-icon.muted {
    color: var(--color-text-muted, #6c7086);
  }

  .deploy-icon.success {
    color: #22c55e;
  }

  .deploy-icon.error {
    color: #f38ba8;
  }

  .deploy-icon.spinning {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .deploy-message {
    margin: 0;
    font-size: 14px;
    color: var(--color-text-muted, #6c7086);
  }

  .error-text {
    color: #f38ba8;
  }

  .provider-badge {
    display: inline-block;
    padding: 4px 12px;
    border-radius: 16px;
    font-size: 13px;
    font-weight: 500;
    background: var(--color-hover, rgba(255, 255, 255, 0.06));
    color: var(--color-text, #cdd6f4);
  }

  .btn {
    padding: 8px 20px;
    border: none;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .btn:hover {
    opacity: 0.9;
  }

  .btn-primary {
    background: var(--color-accent, #6366f1);
    color: #fff;
  }

  .btn-secondary {
    background: var(--color-hover, rgba(255, 255, 255, 0.06));
    color: var(--color-text, #cdd6f4);
  }

  .btn-group {
    display: flex;
    gap: 8px;
  }
</style>
