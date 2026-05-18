<script>
  import { ui, siteConfig, addToast } from '../stores/app.svelte.js'
  import { deploy as apiDeploy } from '../api.js'
  import { X, Loader, Rocket, CheckCircle, AlertCircle } from 'lucide-svelte'
  import AppDialog from './primitives/AppDialog.svelte'
  import AppButton from './primitives/AppButton.svelte'

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
      resultMessage = `Deployed to ${providerLabel(resp?.provider || provider)}`
      addToast('success', resultMessage)
    } catch (e) {
      status = 'error'
      resultMessage = e.message || 'Deployment failed'
    }
  }
</script>

<AppDialog
  open={ui.deployOpen}
  onOpenChange={(v) => (ui.deployOpen = v)}
  ariaLabel="Deploy"
  width="400px"
>
  <div class="modal-header">
    <h2>Deploy</h2>
    <AppButton variant="ghost" size="icon" onclick={close}>
      <X size={16} />
    </AppButton>
  </div>

  <div class="modal-body">
    {#if !configured}
      <div class="deploy-state">
        <div class="deploy-icon muted">
          <Rocket size={32} />
        </div>
        <p class="deploy-message">No deploy provider configured.</p>
        <AppButton variant="primary" onclick={openDeploySettings}>
          Configure in Settings
        </AppButton>
      </div>

    {:else if status === 'idle'}
      <div class="deploy-state">
        <div class="deploy-icon">
          <Rocket size={32} />
        </div>
        <div class="provider-badge">{providerLabel(provider)}</div>
        <AppButton variant="primary" onclick={doDeploy}>
          Deploy Now
        </AppButton>
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
        <AppButton variant="secondary" onclick={close}>Close</AppButton>
      </div>

    {:else if status === 'error'}
      <div class="deploy-state">
        <div class="deploy-icon error">
          <AlertCircle size={32} />
        </div>
        <p class="deploy-message error-text">{resultMessage}</p>
        <div class="btn-group">
          <AppButton variant="primary" onclick={() => { status = 'idle' }}>Try Again</AppButton>
          <AppButton variant="secondary" onclick={close}>Close</AppButton>
        </div>
      </div>
    {/if}
  </div>
</AppDialog>

<style>
  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid var(--cr-border);
  }

  .modal-header h2 {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    color: var(--cr-text);
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
    color: var(--cr-accent);
  }

  .deploy-icon.muted {
    color: var(--cr-text-muted);
  }

  .deploy-icon.success {
    color: var(--cr-success);
  }

  .deploy-icon.error {
    color: var(--cr-danger);
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
    color: var(--cr-text-muted);
  }

  .error-text {
    color: var(--cr-danger);
  }

  .provider-badge {
    display: inline-block;
    padding: 4px 12px;
    border-radius: 16px;
    font-size: 13px;
    font-weight: 500;
    background: var(--cr-hover);
    color: var(--cr-text);
  }

  .btn-group {
    display: flex;
    gap: 8px;
  }
</style>
