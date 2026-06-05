const cfg = (window.__pluginConfig && window.__pluginConfig["code-fullscreen"]) || {};

const config = {
  fullscreenButtonTooltip: cfg.fullscreen_button_tooltip || 'Toggle fullscreen view',
  enableEscapeKey: cfg.enable_escape_key !== false,
  exitOnBrowserBack: cfg.exit_on_browser_back !== false,
  showOnHoverOnly: cfg.show_on_hover_only !== false,
  animationDuration: cfg.animation_duration ?? 200,
  minBlockHeight: cfg.min_block_height ?? 60,
};

const CONSTANTS = {
  MIN_FONT_SIZE: 60,
  MAX_FONT_SIZE: 500,
  DEFAULT_FONT_SIZE: 100,
  FONT_ADJUSTMENT: 10,
  DOUBLE_CLICK_THRESHOLD: 600,
  HINT_DISPLAY_TIME: 4000,
  FADE_TRANSITION_TIME: 500,
  STORAGE_KEY: 'codeFullscreenFontSize',
};

const state = {
  isFullscreenActive: false,
  scrollPosition: 0,
  originalCodeBlock: null,
  fontSize: CONSTANTS.DEFAULT_FONT_SIZE,
  focusTrapHandler: null,
};

// ========================================
// SVG helpers (hardcoded paths, no user input)
// ========================================

function makeSvg(pathD, size = 16) {
  const NS = 'http://www.w3.org/2000/svg';
  const svg = document.createElementNS(NS, 'svg');
  svg.setAttribute('xmlns', NS);
  svg.setAttribute('width', String(size));
  svg.setAttribute('height', String(size));
  svg.setAttribute('viewBox', '0 0 24 24');
  svg.setAttribute('fill', 'none');
  svg.setAttribute('stroke', 'currentColor');
  svg.setAttribute('stroke-width', '2');
  const path = document.createElementNS(NS, 'path');
  path.setAttribute('d', pathD);
  svg.appendChild(path);
  return svg;
}

// ========================================
// Font Size Manager
// ========================================

const fontManager = {
  loadFontSize() {
    try {
      const saved = localStorage.getItem(CONSTANTS.STORAGE_KEY);
      if (saved) {
        const parsed = parseInt(saved, 10);
        if (parsed >= CONSTANTS.MIN_FONT_SIZE && parsed <= CONSTANTS.MAX_FONT_SIZE) {
          return parsed;
        }
      }
    } catch (e) { /* localStorage unavailable */ }
    return CONSTANTS.DEFAULT_FONT_SIZE;
  },

  saveFontSize(size) {
    try { localStorage.setItem(CONSTANTS.STORAGE_KEY, size.toString()); }
    catch (e) { /* localStorage unavailable */ }
  },

  adjustFontSize(change, codeBlock) {
    const newSize = Math.max(CONSTANTS.MIN_FONT_SIZE, Math.min(CONSTANTS.MAX_FONT_SIZE, state.fontSize + change));
    state.fontSize = newSize;
    this.saveFontSize(newSize);
    this.applyFontSize(codeBlock);
  },

  resetFontSize(codeBlock) {
    state.fontSize = CONSTANTS.DEFAULT_FONT_SIZE;
    this.saveFontSize(CONSTANTS.DEFAULT_FONT_SIZE);
    this.applyFontSize(codeBlock);
  },

  applyFontSize(codeBlock) {
    if (!codeBlock) return;
    const scale = state.fontSize / 100;
    const newSize = `${14 * scale}px`;
    const pre = codeBlock.tagName === 'PRE' ? codeBlock : codeBlock.querySelector('pre.chroma');
    if (pre) {
      pre.style.setProperty('--code-font-scale', String(scale));
      pre.style.setProperty('font-size', newSize, 'important');
      const code = pre.querySelector('code.chroma-code');
      if (code) code.style.setProperty('font-size', newSize, 'important');
    }
  },
};

// ========================================
// DOM Creation
// ========================================

function createFullscreenContainer() {
  if (document.querySelector('.sarde-code-fullscreen-container')) return;
  const container = document.createElement('div');
  container.className = 'sarde-code-fullscreen-container';
  container.setAttribute('role', 'dialog');
  container.setAttribute('aria-modal', 'true');
  container.setAttribute('aria-label', 'Code block in fullscreen view');
  container.setAttribute('tabindex', '-1');
  container.style.setProperty('--fs-animation-duration', config.animationDuration + 'ms');
  document.body.appendChild(container);
}

function createFullscreenButton() {
  const btn = document.createElement('button');
  btn.className = 'sarde-fullscreen-btn';
  btn.title = config.fullscreenButtonTooltip;
  btn.setAttribute('aria-label', config.fullscreenButtonTooltip);
  const icon = document.createElement('span');
  icon.className = 'sarde-fullscreen-icon';
  btn.appendChild(icon);
  return btn;
}

function createFontSizeControls() {
  const controls = document.createElement('div');
  controls.className = 'sarde-code-fullscreen-font-controls';

  const decreaseBtn = document.createElement('button');
  decreaseBtn.className = 'sarde-code-fullscreen-font-btn sarde-code-fullscreen-font-btn-decrease';
  decreaseBtn.setAttribute('aria-label', 'Decrease font size');
  decreaseBtn.title = 'Decrease font size (Double-click to reset)';
  decreaseBtn.appendChild(makeSvg('M5 12h14'));
  controls.appendChild(decreaseBtn);

  const increaseBtn = document.createElement('button');
  increaseBtn.className = 'sarde-code-fullscreen-font-btn sarde-code-fullscreen-font-btn-increase';
  increaseBtn.setAttribute('aria-label', 'Increase font size');
  increaseBtn.title = 'Increase font size';
  increaseBtn.appendChild(makeSvg('M12 5v14m-7-7h14'));
  controls.appendChild(increaseBtn);

  return controls;
}

function createFullscreenHint() {
  const hint = document.createElement('div');
  hint.className = 'sarde-code-fullscreen-hint';
  hint.textContent = 'Press ';
  const kbd = document.createElement('kbd');
  kbd.textContent = 'Esc';
  hint.appendChild(kbd);
  hint.appendChild(document.createTextNode(' to exit full screen'));
  return hint;
}

function createCloseButton(container) {
  const btn = document.createElement('button');
  btn.className = 'sarde-code-fullscreen-close';
  btn.setAttribute('aria-label', 'Exit fullscreen');
  btn.appendChild(makeSvg('M18 6L6 18M6 6l12 12', 20));
  btn.addEventListener('click', () => exitFullscreen(container));
  return btn;
}

// ========================================
// Button Injection
// ========================================

function initializeCodeBlocks() {
  const blocks = document.querySelectorAll('.sarde-code-block');

  blocks.forEach(block => {
    if (block.querySelector('.sarde-fullscreen-btn')) return;

    const frame = block.querySelector('.frame');
    if (frame && frame.offsetHeight < config.minBlockHeight) return;

    const btn = createFullscreenButton();

    let toolbar = block.querySelector('.sarde-code-block-wrapper');
    if (toolbar) {
      toolbar.appendChild(btn);
    } else {
      toolbar = document.createElement('div');
      toolbar.className = 'sarde-code-block-wrapper';
      toolbar.appendChild(btn);
      const figure = block.querySelector('figure.frame');
      if (figure) figure.appendChild(toolbar);
    }

    btn.addEventListener('click', (e) => {
      e.preventDefault();
      e.stopPropagation();
      toggleFullscreen(block);
    });
  });
}

// ========================================
// Fullscreen Toggle
// ========================================

function toggleFullscreen(codeBlock) {
  const container = document.querySelector('.sarde-code-fullscreen-container');
  if (state.isFullscreenActive) {
    exitFullscreen(container);
  } else {
    enterFullscreen(codeBlock, container);
  }
}

function enterFullscreen(codeBlock, container) {
  state.originalCodeBlock = codeBlock;
  state.fontSize = fontManager.loadFontSize();
  state.scrollPosition = window.scrollY || document.documentElement.scrollTop;
  document.body.style.overflow = 'hidden';

  // Clone the entire .code-block wrapper
  const clonedBlock = codeBlock.cloneNode(true);
  clonedBlock.classList.add('sarde-code-fullscreen-active');
  clonedBlock.querySelectorAll('.fullscreen-btn').forEach(btn => btn.remove());

  const contentWrapper = document.createElement('div');
  contentWrapper.className = 'sarde-code-fullscreen-content';

  // Toolbar: spacer | font controls | close button
  const toolbar = document.createElement('div');
  toolbar.className = 'sarde-code-fullscreen-toolbar';

  const spacer = document.createElement('div');
  spacer.className = 'sarde-code-fullscreen-toolbar-spacer';
  toolbar.appendChild(spacer);

  const fontControls = createFontSizeControls();
  toolbar.appendChild(fontControls);

  const toolbarEnd = document.createElement('div');
  toolbarEnd.className = 'sarde-code-fullscreen-toolbar-end';
  toolbarEnd.appendChild(createCloseButton(container));
  toolbar.appendChild(toolbarEnd);

  contentWrapper.appendChild(toolbar);
  contentWrapper.appendChild(clonedBlock);
  container.appendChild(contentWrapper);

  addFontControlListeners(fontControls, clonedBlock);
  fontManager.applyFontSize(clonedBlock);

  if (config.enableEscapeKey) {
    const hint = createFullscreenHint();
    container.appendChild(hint);
    setTimeout(() => {
      if (hint && hint.parentNode) {
        hint.style.transition = 'opacity 0.5s ease';
        hint.style.opacity = '0';
        setTimeout(() => hint.remove(), CONSTANTS.FADE_TRANSITION_TIME);
      }
    }, CONSTANTS.HINT_DISPLAY_TIME);
  }

  container.classList.add('is-open');
  state.isFullscreenActive = true;
  container.focus();

  if (config.enableEscapeKey) {
    document.addEventListener('keyup', handleKeyup);
  }
  if (config.exitOnBrowserBack) {
    history.pushState({ fullscreenActive: true }, '', window.location.href);
    window.addEventListener('popstate', handlePopState);
  }

  addFocusTrap(container);
}

function exitFullscreen(container) {
  if (!state.isFullscreenActive) return;

  document.body.style.overflow = '';
  document.removeEventListener('keyup', handleKeyup);
  window.removeEventListener('popstate', handlePopState);

  if (history.state && history.state.fullscreenActive) {
    history.replaceState(null, '', window.location.href);
  }

  removeFocusTrap();

  container.classList.remove('is-open');
  container.innerHTML = '';
  state.isFullscreenActive = false;

  if (state.originalCodeBlock) {
    const btn = state.originalCodeBlock.querySelector('.sarde-fullscreen-btn');
    if (btn) btn.focus();
  }
  state.originalCodeBlock = null;

  setTimeout(() => {
    window.scrollTo({ top: state.scrollPosition, behavior: 'smooth' });
  }, 0);
}

// ========================================
// Event Handlers
// ========================================

function handleKeyup(event) {
  if (event.key === 'Escape' && state.isFullscreenActive) {
    const container = document.querySelector('.sarde-code-fullscreen-container');
    if (container) exitFullscreen(container);
  }
}

function handlePopState() {
  if (!state.isFullscreenActive) return;

  const container = document.querySelector('.sarde-code-fullscreen-container');
  if (!container) return;

  window.removeEventListener('popstate', handlePopState);
  document.removeEventListener('keyup', handleKeyup);
  document.body.style.overflow = '';

  removeFocusTrap();

  container.classList.remove('is-open');
  container.innerHTML = '';
  state.isFullscreenActive = false;

  if (state.originalCodeBlock) {
    const btn = state.originalCodeBlock.querySelector('.sarde-fullscreen-btn');
    if (btn) btn.focus();
  }
  state.originalCodeBlock = null;

  setTimeout(() => {
    window.scrollTo({ top: state.scrollPosition, behavior: 'smooth' });
  }, 0);
}

// ========================================
// Focus Trap
// ========================================

function addFocusTrap(container) {
  const focusable = container.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
  if (focusable.length === 0) return;

  const first = focusable[0];
  const last = focusable[focusable.length - 1];

  function handleTab(event) {
    if (event.key !== 'Tab') return;
    if (event.shiftKey) {
      if (document.activeElement === first) { event.preventDefault(); last.focus(); }
    } else {
      if (document.activeElement === last) { event.preventDefault(); first.focus(); }
    }
  }

  container.addEventListener('keydown', handleTab);
  state.focusTrapHandler = handleTab;
}

function removeFocusTrap() {
  const container = document.querySelector('.sarde-code-fullscreen-container');
  if (container && state.focusTrapHandler) {
    container.removeEventListener('keydown', state.focusTrapHandler);
    state.focusTrapHandler = null;
  }
}

// ========================================
// Font Control Listeners
// ========================================

function addFontControlListeners(fontControls, codeBlock) {
  const decreaseBtn = fontControls.querySelector('.sarde-code-fullscreen-font-btn-decrease');
  const increaseBtn = fontControls.querySelector('.sarde-code-fullscreen-font-btn-increase');
  let decreaseClickData = { lastClickTime: 0, clickCount: 0 };

  if (decreaseBtn) {
    decreaseBtn.addEventListener('click', (event) => {
      const now = Date.now();
      if (now - decreaseClickData.lastClickTime < CONSTANTS.DOUBLE_CLICK_THRESHOLD) {
        decreaseClickData.clickCount++;
        if (decreaseClickData.clickCount === 2) {
          fontManager.resetFontSize(codeBlock);
          decreaseClickData.clickCount = 0;
        }
      } else {
        decreaseClickData.clickCount = 1;
        fontManager.adjustFontSize(-CONSTANTS.FONT_ADJUSTMENT, codeBlock);
      }
      decreaseClickData.lastClickTime = now;
      event.target.blur();
    });
  }

  if (increaseBtn) {
    increaseBtn.addEventListener('click', (event) => {
      fontManager.adjustFontSize(CONSTANTS.FONT_ADJUSTMENT, codeBlock);
      event.target.blur();
    });
  }
}

// ========================================
// Initialize
// ========================================

createFullscreenContainer();
initializeCodeBlocks();

const observer = new MutationObserver(mutations => {
  for (const mutation of mutations) {
    for (const node of mutation.addedNodes) {
      if (node.nodeType === 1 && (
        node.matches?.('.sarde-code-block') ||
        node.querySelector?.('.sarde-code-block')
      )) {
        initializeCodeBlocks();
        return;
      }
    }
  }
});
observer.observe(document.body, { childList: true, subtree: true });
