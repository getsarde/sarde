const cfg = (window.__pluginConfig && window.__pluginConfig["code-collapsible"]) || {};

const config = {
  lineThreshold: cfg.line_threshold ?? 15,
  previewLines: cfg.preview_lines ?? 8,
  defaultCollapsed: cfg.default_collapsed !== false,
  expandButtonText: cfg.expand_button_text || 'Show more',
  collapseButtonText: cfg.collapse_button_text || 'Show less',
};

const FALLBACK_LINE_HEIGHT = 21.6;
const FALLBACK_PADDING = 56;

// ========================================
// Screen Reader Support
// ========================================

function getOrCreateLiveRegion() {
  let region = document.getElementById('sarde-cc-live-region');
  if (!region) {
    region = document.createElement('div');
    region.id = 'sarde-cc-live-region';
    region.setAttribute('aria-live', 'polite');
    region.setAttribute('aria-atomic', 'true');
    region.style.cssText =
      'position:absolute;width:1px;height:1px;padding:0;margin:-1px;' +
      'overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0;';
    document.body.appendChild(region);
  }
  return region;
}

function announceStateChange(isExpanded) {
  const region = getOrCreateLiveRegion();
  region.textContent = isExpanded ? 'Code block expanded' : 'Code block collapsed';
  setTimeout(() => { region.textContent = ''; }, 1000);
}

// ========================================
// Height & Color Detection
// ========================================

function calcPreviewHeight(codeBlock, previewLines) {
  const codeEl = codeBlock.querySelector('code.chroma-code');
  if (codeEl) {
    const style = window.getComputedStyle(codeEl);
    const lineHeight = parseFloat(style.lineHeight) || FALLBACK_LINE_HEIGHT;

    const preEl = codeBlock.querySelector('pre.chroma');
    let padding = FALLBACK_PADDING;
    if (preEl) {
      const preStyle = window.getComputedStyle(preEl);
      padding = parseFloat(preStyle.paddingTop) + parseFloat(preStyle.paddingBottom);
    }

    const headerEl = codeBlock.querySelector('.header');
    const headerHeight = headerEl ? headerEl.getBoundingClientRect().height : 0;

    return (previewLines * lineHeight) + padding + headerHeight;
  }
  return (previewLines * FALLBACK_LINE_HEIGHT) + FALLBACK_PADDING;
}

function detectAndSetBgColor(codeBlock) {
  const preEls = codeBlock.querySelectorAll('pre.chroma');
  for (const pre of preEls) {
    const style = window.getComputedStyle(pre);
    if (style.display !== 'none' && style.visibility !== 'hidden') {
      codeBlock.style.setProperty('--cc-bg-color', style.backgroundColor);
      return;
    }
  }
}

// ========================================
// Toggle Logic
// ========================================

function toggleCollapse(codeBlock) {
  const isCollapsed = codeBlock.classList.contains('sarde-cc-collapsed');

  if (isCollapsed) {
    codeBlock.classList.remove('sarde-cc-collapsed');
    codeBlock.classList.add('sarde-cc-expanded');
  } else {
    codeBlock.classList.remove('sarde-cc-expanded');
    codeBlock.classList.add('sarde-cc-collapsed');

    // If collapsing and block is above viewport, scroll it back into view
    const rect = codeBlock.getBoundingClientRect();
    if (rect.top < 0) {
      const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
      codeBlock.scrollIntoView({
        behavior: prefersReducedMotion ? 'instant' : 'smooth',
        block: 'start',
      });
    }
  }

  const expanded = codeBlock.classList.contains('sarde-cc-expanded');
  codeBlock.querySelectorAll('.sarde-cc-toggle').forEach(btn => {
    btn.setAttribute('aria-expanded', expanded ? 'true' : 'false');
  });

  announceStateChange(expanded);
}

// ========================================
// Toggle Button Creation
// ========================================

function createChevronSvg() {
  const NS = 'http://www.w3.org/2000/svg';
  const svg = document.createElementNS(NS, 'svg');
  svg.setAttribute('xmlns', NS);
  svg.setAttribute('width', '14');
  svg.setAttribute('height', '14');
  svg.setAttribute('viewBox', '0 0 24 24');
  svg.setAttribute('fill', 'none');
  svg.setAttribute('stroke', 'currentColor');
  svg.setAttribute('stroke-width', '2.5');
  svg.setAttribute('stroke-linecap', 'round');
  svg.setAttribute('stroke-linejoin', 'round');
  const path = document.createElementNS(NS, 'path');
  path.setAttribute('d', 'M6 9l6 6 6-6');
  svg.appendChild(path);
  return svg;
}

function createToggleButton(codeBlock) {
  const btn = document.createElement('button');
  btn.className = 'sarde-cc-toggle';
  btn.setAttribute('aria-expanded', config.defaultCollapsed ? 'false' : 'true');
  btn.setAttribute('type', 'button');

  const icon = document.createElement('span');
  icon.className = 'sarde-cc-icon';
  icon.appendChild(createChevronSvg());
  btn.appendChild(icon);

  const textExpand = document.createElement('span');
  textExpand.className = 'sarde-cc-text-expand';
  textExpand.textContent = config.expandButtonText;
  btn.appendChild(textExpand);

  const textCollapse = document.createElement('span');
  textCollapse.className = 'sarde-cc-text-collapse';
  textCollapse.textContent = config.collapseButtonText;
  btn.appendChild(textCollapse);

  btn.addEventListener('click', (e) => {
    e.preventDefault();
    toggleCollapse(codeBlock);
  });

  return btn;
}

function createGradientOverlay() {
  const grad = document.createElement('div');
  grad.className = 'sarde-cc-gradient';
  grad.setAttribute('aria-hidden', 'true');
  return grad;
}

// ========================================
// Initialization
// ========================================

function initCodeBlock(codeBlock) {
  // Already initialized
  if (codeBlock.dataset.ccInit) return;
  codeBlock.dataset.ccInit = 'true';

  const collapseAttr = codeBlock.dataset.collapse;
  const lineCount = parseInt(codeBlock.dataset.lines || '0', 10);

  // Skip if explicitly disabled
  if (collapseAttr === 'none') return;

  // Apply collapse if forced or line count meets threshold
  const shouldCollapse = collapseAttr === 'force' || lineCount >= config.lineThreshold;
  if (!shouldCollapse) return;

  codeBlock.classList.add('sarde-cc-collapsible');

  // Set preview height and bg color
  const previewHeight = calcPreviewHeight(codeBlock, config.previewLines);
  codeBlock.style.setProperty('--cc-preview-height', previewHeight + 'px');
  detectAndSetBgColor(codeBlock);

  // Inject gradient into the .frame element (sits above pre.chroma)
  const frame = codeBlock.querySelector('.frame');
  if (frame) {
    frame.appendChild(createGradientOverlay());
  }

  // Inject toggle button after the .frame (or after the block)
  const toggleBtn = createToggleButton(codeBlock);
  codeBlock.appendChild(toggleBtn);

  // Set initial state
  if (config.defaultCollapsed) {
    codeBlock.classList.add('sarde-cc-collapsed');
  } else {
    codeBlock.classList.add('sarde-cc-expanded');
  }
}

function initAllCodeBlocks() {
  document.querySelectorAll('.code-block[data-lines]').forEach(block => {
    initCodeBlock(block);
  });
}

// ========================================
// Debounce
// ========================================

function debounce(fn, delay) {
  let timeoutId;
  return function () {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(fn, delay);
  };
}

// ========================================
// Initialize
// ========================================

initAllCodeBlocks();

const debouncedInit = debounce(initAllCodeBlocks, 100);
new MutationObserver(debouncedInit).observe(document.body, {
  childList: true,
  subtree: true,
});
