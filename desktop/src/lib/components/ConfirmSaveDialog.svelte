<script>
  import AppDialog from './primitives/AppDialog.svelte'

  let { open = false, fileName = '', onSave, onDiscard, onCancel } = $props()
</script>

<AppDialog {open} onOpenChange={(v) => { if (!v) onCancel?.() }} ariaLabel="Unsaved changes" width="380px">
  <div class="confirm-save">
    <div class="confirm-title">Unsaved Changes</div>
    <p class="confirm-message">
      <strong>{fileName}</strong> has unsaved changes. Do you want to save before closing?
    </p>
    <div class="confirm-actions">
      <button class="btn btn-ghost" onclick={onDiscard}>Don't Save</button>
      <div class="spacer"></div>
      <button class="btn btn-secondary" onclick={onCancel}>Cancel</button>
      <button class="btn btn-primary" onclick={onSave}>Save</button>
    </div>
  </div>
</AppDialog>

<style>
  .confirm-save {
    padding: 20px;
  }

  .confirm-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--cr-text);
    margin-bottom: 8px;
  }

  .confirm-message {
    font-size: 13px;
    color: var(--cr-text-muted);
    line-height: 1.5;
    margin: 0 0 20px;
  }

  .confirm-message strong {
    color: var(--cr-text);
  }

  .confirm-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .spacer {
    flex: 1;
  }

  .btn {
    padding: 6px 14px;
    border: none;
    border-radius: var(--cr-radius-sm);
    font-size: 13px;
    cursor: pointer;
    font-weight: 500;
    transition: background 0.15s;
  }

  .btn-primary {
    background: var(--cr-accent);
    color: var(--cr-bg-base);
  }

  .btn-primary:hover {
    filter: brightness(1.1);
  }

  .btn-secondary {
    background: var(--cr-bg-elevated);
    color: var(--cr-text);
  }

  .btn-secondary:hover {
    background: var(--cr-bg-surface);
  }

  .btn-ghost {
    background: transparent;
    color: var(--cr-text-muted);
  }

  .btn-ghost:hover {
    background: var(--cr-bg-elevated);
    color: var(--cr-text);
  }
</style>
