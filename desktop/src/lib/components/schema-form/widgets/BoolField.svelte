<script>
  import FieldWrapper from './FieldWrapper.svelte'

  let { name, label = name, value = false, required = false, error = '', disabled = false, onchange } = $props()
</script>

<FieldWrapper {name} {label} {required} {error}>
  <button
    class="toggle"
    class:active={!!value}
    role="switch"
    aria-checked={!!value}
    aria-label={label}
    {disabled}
    onclick={() => onchange(!value)}
  >
    <span class="toggle-knob"></span>
  </button>
</FieldWrapper>

<style>
  .toggle {
    position: relative;
    width: 36px;
    height: 20px;
    border-radius: 10px;
    border: none;
    background: var(--cr-border);
    cursor: pointer;
    padding: 0;
    transition: background 0.2s;
  }

  .toggle.active {
    background: var(--cr-accent);
  }

  .toggle:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .toggle-knob {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: #fff;
    transition: transform 0.2s;
    pointer-events: none;
  }

  .toggle.active .toggle-knob {
    transform: translateX(16px);
  }
</style>
