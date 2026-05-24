// Reading Preferences Plugin
// Floating control panel for adjusting typography, spacing, and layout.

const cfg = (window.__pluginConfig && window.__pluginConfig["reading-preferences"]) || {};

const config = {
    minFontSize: cfg.min_font_size || 75,
    maxFontSize: cfg.max_font_size || 300,
    showWidthControl: cfg.show_width_control !== false,
};

const STORAGE_KEY = 'sarde_reading_prefs';

const DEFAULTS = {
    fontFamily: 'sans',
    fontSize: 100,
    letterSpacing: 0,
    lineHeight: 1.85,
    paragraphSpacing: 1.25,
    textAlign: 'left',
    contentWidth: 768,
};

const MIN_LINE_HEIGHT = 1.2;
const MAX_LINE_HEIGHT = 2.4;
const LINE_HEIGHT_STEP = 0.05;
const MIN_LETTER_SPACING = -0.5;
const MAX_LETTER_SPACING = 3;
const LETTER_SPACING_STEP = 0.1;
const MIN_PARA_SPACING = 0.5;
const MAX_PARA_SPACING = 3;
const PARA_SPACING_STEP = 0.125;
const MIN_WIDTH = 600;
const MAX_WIDTH = 1100;
const WIDTH_STEP = 10;

const FONT_STACKS = {
    sans: "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
    serif: 'Georgia, "Times New Roman", Times, serif',
    mono: "ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace",
    dyslexic: '"OpenDyslexic", "Comic Sans MS", sans-serif',
};

const FONT_LABELS = { sans: 'Sans', serif: 'Serif', mono: 'Mono', dyslexic: 'Dyslexic' };

let dyslexicFontLoaded = false;
let prefs = loadPrefs();
let btn = null;
let panel = null;
let panelOpen = false;

function clamp(val, min, max) { return Math.min(Math.max(val, min), max); }

function loadPrefs() {
    try {
        const stored = localStorage.getItem(STORAGE_KEY);
        if (stored) {
            const p = JSON.parse(stored);
            return {
                fontFamily: p.fontFamily || DEFAULTS.fontFamily,
                fontSize: clamp(p.fontSize || DEFAULTS.fontSize, config.minFontSize, config.maxFontSize),
                letterSpacing: clamp(p.letterSpacing != null ? p.letterSpacing : DEFAULTS.letterSpacing, MIN_LETTER_SPACING, MAX_LETTER_SPACING),
                lineHeight: clamp(p.lineHeight || DEFAULTS.lineHeight, MIN_LINE_HEIGHT, MAX_LINE_HEIGHT),
                paragraphSpacing: clamp(p.paragraphSpacing != null ? p.paragraphSpacing : DEFAULTS.paragraphSpacing, MIN_PARA_SPACING, MAX_PARA_SPACING),
                textAlign: p.textAlign || DEFAULTS.textAlign,
                contentWidth: clamp(p.contentWidth || DEFAULTS.contentWidth, MIN_WIDTH, MAX_WIDTH),
            };
        }
    } catch (e) { /* ignore */ }
    return JSON.parse(JSON.stringify(DEFAULTS));
}

function savePrefs() {
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs)); } catch (e) { /* ignore */ }
}

function loadDyslexicFont() {
    if (dyslexicFontLoaded) return;
    dyslexicFontLoaded = true;
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = 'https://fonts.cdnfonts.com/css/open-dyslexic';
    document.head.appendChild(link);
}

function applyPrefs() {
    const root = document.documentElement;
    if (prefs.fontFamily === 'dyslexic') loadDyslexicFont();
    root.style.setProperty('--rp-font-family', FONT_STACKS[prefs.fontFamily] || FONT_STACKS.sans);
    root.style.setProperty('--rp-font-size', prefs.fontSize + '%');
    root.style.setProperty('--rp-letter-spacing', prefs.letterSpacing + 'px');
    root.style.setProperty('--rp-line-height', String(prefs.lineHeight));
    root.style.setProperty('--rp-paragraph-spacing', prefs.paragraphSpacing + 'em');
    root.style.setProperty('--rp-text-align', prefs.textAlign);
    if (config.showWidthControl) {
        root.style.setProperty('--rp-content-width', prefs.contentWidth + 'px');
    }
}

function resetPrefs() {
    prefs = JSON.parse(JSON.stringify(DEFAULTS));
    savePrefs();
    applyPrefs();
    if (panel) {
        const wasVisible = panelOpen;
        panel.remove();
        panel = null;
        if (wasVisible) {
            createPanel();
            panelOpen = true;
            if (btn) btn.classList.add('is-panel-open');
            panel.offsetHeight;
            panel.classList.add('sarde-reading-prefs-panel--visible');
        }
    }
}

function togglePanel() { panelOpen ? closePanel() : openPanel(); }

function openPanel() {
    if (!panel) createPanel();
    panelOpen = true;
    if (btn) btn.classList.add('is-panel-open');
    if (panel) { panel.offsetHeight; panel.classList.add('sarde-reading-prefs-panel--visible'); }
}

function closePanel() {
    panelOpen = false;
    if (btn) btn.classList.remove('is-panel-open');
    if (panel) panel.classList.remove('sarde-reading-prefs-panel--visible');
}

function formatValue(val, unit) {
    if (unit === 'x') return val.toFixed(2) + 'x';
    if (unit === '%') return val + '%';
    if (unit === 'px') return val + 'px';
    if (unit === 'em') return val.toFixed(2) + 'em';
    return String(val);
}

function createSlider(label, id, min, max, step, value, unit, onChange) {
    const group = document.createElement('div');
    group.className = 'sarde-reading-prefs-group';
    const labelRow = document.createElement('div');
    labelRow.className = 'sarde-reading-prefs-label';
    const labelText = document.createElement('span');
    labelText.textContent = label;
    labelRow.appendChild(labelText);
    const valueText = document.createElement('span');
    valueText.className = 'sarde-reading-prefs-value';
    valueText.textContent = formatValue(value, unit);
    labelRow.appendChild(valueText);
    group.appendChild(labelRow);
    const range = document.createElement('input');
    range.type = 'range';
    range.className = 'sarde-reading-prefs-range';
    range.id = 'rp-' + id;
    range.min = String(min);
    range.max = String(max);
    range.step = String(step);
    range.value = String(value);
    range.setAttribute('aria-label', label);
    range.addEventListener('input', () => {
        const val = parseFloat(range.value);
        valueText.textContent = formatValue(val, unit);
        onChange(val);
    });
    group.appendChild(range);
    return group;
}

function createButtonGroup(label, options, activeValue, onChange) {
    const group = document.createElement('div');
    group.className = 'sarde-reading-prefs-group';
    const labelEl = document.createElement('div');
    labelEl.className = 'sarde-reading-prefs-label';
    const labelText = document.createElement('span');
    labelText.textContent = label;
    labelEl.appendChild(labelText);
    group.appendChild(labelEl);
    const btnGroup = document.createElement('div');
    btnGroup.className = 'sarde-reading-prefs-btngroup';
    for (const key of Object.keys(options)) {
        const item = document.createElement('button');
        item.className = 'sarde-reading-prefs-btngroup__item';
        if (key === activeValue) item.classList.add('is-active');
        item.textContent = options[key];
        item.setAttribute('aria-label', options[key]);
        item.addEventListener('click', () => {
            for (const s of btnGroup.querySelectorAll('.sarde-reading-prefs-btngroup-item')) s.classList.remove('is-active');
            item.classList.add('is-active');
            onChange(key);
        });
        btnGroup.appendChild(item);
    }
    group.appendChild(btnGroup);
    return group;
}

function createDivider() {
    const hr = document.createElement('hr');
    hr.className = 'sarde-reading-prefs-divider';
    return hr;
}

function createPanel() {
    panel = document.createElement('div');
    panel.className = 'sarde-reading-prefs-panel';
    panel.setAttribute('role', 'dialog');
    panel.setAttribute('aria-label', 'Reading preferences');
    const header = document.createElement('div');
    header.className = 'sarde-reading-prefs-header';
    const title = document.createElement('span');
    title.className = 'sarde-reading-prefs-title';
    title.textContent = 'Reading';
    header.appendChild(title);
    const resetBtn = document.createElement('button');
    resetBtn.className = 'sarde-reading-prefs-reset';
    resetBtn.textContent = 'Reset';
    resetBtn.setAttribute('aria-label', 'Reset to defaults');
    resetBtn.addEventListener('click', resetPrefs);
    header.appendChild(resetBtn);
    panel.appendChild(header);

    const update = (fn) => (val) => { fn(val); savePrefs(); applyPrefs(); };

    panel.appendChild(createButtonGroup('Font', FONT_LABELS, prefs.fontFamily, update((v) => { prefs.fontFamily = v; })));
    panel.appendChild(createSlider('Size', 'font-size', config.minFontSize, config.maxFontSize, 5, prefs.fontSize, '%', update((v) => { prefs.fontSize = v; })));
    panel.appendChild(createSlider('Letter spacing', 'letter-spacing', MIN_LETTER_SPACING, MAX_LETTER_SPACING, LETTER_SPACING_STEP, prefs.letterSpacing, 'px', update((v) => { prefs.letterSpacing = Math.round(v * 10) / 10; })));
    panel.appendChild(createDivider());
    panel.appendChild(createSlider('Line spacing', 'line-height', MIN_LINE_HEIGHT, MAX_LINE_HEIGHT, LINE_HEIGHT_STEP, prefs.lineHeight, 'x', update((v) => { prefs.lineHeight = Math.round(v * 100) / 100; })));
    panel.appendChild(createSlider('Paragraph spacing', 'para-spacing', MIN_PARA_SPACING, MAX_PARA_SPACING, PARA_SPACING_STEP, prefs.paragraphSpacing, 'em', update((v) => { prefs.paragraphSpacing = Math.round(v * 1000) / 1000; })));
    panel.appendChild(createButtonGroup('Alignment', { left: 'Left', justify: 'Justified' }, prefs.textAlign, update((v) => { prefs.textAlign = v; })));

    if (config.showWidthControl) {
        panel.appendChild(createDivider());
        panel.appendChild(createSlider('Content width', 'content-width', MIN_WIDTH, MAX_WIDTH, WIDTH_STEP, prefs.contentWidth, 'px', update((v) => { prefs.contentWidth = v; })));
    }

    document.body.appendChild(panel);
    document.addEventListener('click', onDocClick);
    document.addEventListener('keydown', onEscape);
}

function onDocClick(e) {
    if (!panelOpen) return;
    if (panel && panel.contains(e.target)) return;
    if (btn && btn.contains(e.target)) return;
    closePanel();
}

function onEscape(e) {
    if (e.key === 'Escape' && panelOpen) closePanel();
}

function createButton() {
    if (document.querySelector('.sarde-reading-prefs-btn')) return;
    btn = document.createElement('button');
    btn.className = 'sarde-reading-prefs-btn';
    btn.setAttribute('aria-label', 'Reading preferences');
    btn.setAttribute('aria-haspopup', 'dialog');
    btn.setAttribute('aria-expanded', 'false');

    // Typography icon (T with serifs)
    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.setAttribute('viewBox', '0 0 24 24');
    const path1 = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    path1.setAttribute('d', 'M4 7V4h16v3');
    const line1 = document.createElementNS('http://www.w3.org/2000/svg', 'line');
    line1.setAttribute('x1', '12'); line1.setAttribute('y1', '4');
    line1.setAttribute('x2', '12'); line1.setAttribute('y2', '20');
    const line2 = document.createElementNS('http://www.w3.org/2000/svg', 'line');
    line2.setAttribute('x1', '8'); line2.setAttribute('y1', '20');
    line2.setAttribute('x2', '16'); line2.setAttribute('y2', '20');
    svg.appendChild(path1);
    svg.appendChild(line1);
    svg.appendChild(line2);
    btn.appendChild(svg);

    document.body.appendChild(btn);
    requestAnimationFrame(() => {
        requestAnimationFrame(() => { if (btn) btn.classList.add('is-visible'); });
    });

    btn.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        togglePanel();
        btn.setAttribute('aria-expanded', panelOpen ? 'true' : 'false');
    });
    btn.addEventListener('touchstart', (e) => {
        e.preventDefault();
        if (btn) btn.classList.add('is-active');
    }, { passive: false });
    btn.addEventListener('touchend', (e) => {
        e.preventDefault();
        togglePanel();
        if (btn) btn.classList.remove('is-active');
    }, { passive: false });
}

function init() {
    if (!document.querySelector('aside.docs-sidebar')) return;
    createButton();
    applyPrefs();
}

init();
