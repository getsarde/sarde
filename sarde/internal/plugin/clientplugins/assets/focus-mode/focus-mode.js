// Focus Mode Plugin
// Hides sidebar and TOC for distraction-free reading.
// Toggle via floating button or F hotkey. Preference persists via localStorage.

const cfg = (window.__pluginConfig && window.__pluginConfig["focus-mode"]) || {};

const config = {
    enableHotkey: cfg.enable_hotkey !== false,
    showButton: cfg.show_button !== false,
    buttonPosition: cfg.button_position || 'bottom-right',
    animationSpeed: cfg.animation_speed != null ? cfg.animation_speed : 300,
};

const STORAGE_KEY = 'sarde_focus_mode';

// Set the CSS custom property for animation speed
document.documentElement.style.setProperty('--focus-mode-speed', config.animationSpeed + 'ms');

// ── State ────────────────────────────────────────────────────

function isEnabled() {
    try {
        return localStorage.getItem(STORAGE_KEY) === '1';
    } catch (e) {
        return false;
    }
}

function save(enabled) {
    try {
        if (enabled) {
            localStorage.setItem(STORAGE_KEY, '1');
        } else {
            localStorage.removeItem(STORAGE_KEY);
        }
    } catch (e) {
        // localStorage unavailable
    }
}

// ── Toggle ───────────────────────────────────────────────────

function applyFocusMode(enabled) {
    if (enabled) {
        document.body.classList.add('focus-mode');
    } else {
        document.body.classList.remove('focus-mode');
    }
    updateTooltip(enabled);
}

function toggle() {
    const next = !document.body.classList.contains('focus-mode');
    applyFocusMode(next);
    save(next);
}

// ── Button ───────────────────────────────────────────────────

let btn = null;
let tooltip = null;
let isKeyboard = false;

// Expand icon (enter focus mode) — compress arrows pointing inward
const ICON_ENTER = '<svg class="sarde-icon-enter" viewBox="0 0 24 24"><polyline points="4 14 10 14 10 20"/><polyline points="20 10 14 10 14 4"/><line x1="14" y1="10" x2="21" y2="3"/><line x1="3" y1="21" x2="10" y2="14"/></svg>';

// Exit icon (leave focus mode) — expand arrows pointing outward
const ICON_EXIT = '<svg class="sarde-icon-exit" viewBox="0 0 24 24"><polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/></svg>';

function updateTooltip(enabled) {
    if (tooltip) {
        tooltip.firstChild.textContent = enabled ? 'Exit focus mode' : 'Focus mode';
    }
    if (btn) {
        btn.setAttribute('aria-label', enabled ? 'Exit focus mode' : 'Enter focus mode');
        btn.setAttribute('aria-pressed', enabled ? 'true' : 'false');
    }
}

function showTooltip() {
    if (tooltip) tooltip.classList.add('visible');
}

function hideTooltip() {
    if (tooltip) tooltip.classList.remove('visible');
}

function createButton() {
    if (!config.showButton) return;
    if (document.querySelector('.sarde-focus-mode-btn')) return;

    btn = document.createElement('button');
    btn.className = 'sarde-focus-mode-btn pos-' + config.buttonPosition;
    btn.setAttribute('aria-label', 'Enter focus mode');
    btn.setAttribute('aria-pressed', 'false');
    btn.setAttribute('role', 'button');
    btn.setAttribute('tabindex', '0');
    btn.innerHTML = ICON_ENTER + ICON_EXIT;

    // Tooltip
    tooltip = document.createElement('div');
    tooltip.className = 'sarde-focus-mode-tooltip';
    tooltip.id = 'sarde-focus-mode-tooltip';
    tooltip.textContent = 'Focus mode';

    const arrow = document.createElement('div');
    arrow.className = 'sarde-focus-mode-tooltip-arrow';
    tooltip.appendChild(arrow);

    btn.appendChild(tooltip);
    btn.setAttribute('aria-describedby', 'sarde-focus-mode-tooltip');

    document.body.appendChild(btn);

    // Show button after a short delay (matches scroll-to-top pattern)
    requestAnimationFrame(function () {
        requestAnimationFrame(function () {
            if (btn) btn.classList.add('visible');
        });
    });

    // Events
    btn.addEventListener('click', function (e) {
        e.preventDefault();
        toggle();
    });

    btn.addEventListener('mouseenter', showTooltip);
    btn.addEventListener('mouseleave', hideTooltip);

    // Keyboard accessibility
    btn.addEventListener('keydown', function (e) {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            toggle();
            btn.classList.remove('keyboard-focus');
        }
    });

    btn.addEventListener('focus', function () {
        if (isKeyboard) {
            showTooltip();
            if (btn) btn.classList.add('keyboard-focus');
        }
    });

    btn.addEventListener('blur', function () {
        hideTooltip();
        if (btn) btn.classList.remove('keyboard-focus');
    });

    btn.addEventListener('mousedown', function () {
        isKeyboard = false;
    });

    // Touch
    btn.addEventListener('touchstart', function (e) {
        e.preventDefault();
        if (btn) btn.classList.add('is-active');
    }, { passive: false });

    btn.addEventListener('touchend', function (e) {
        e.preventDefault();
        toggle();
        if (btn) btn.classList.remove('is-active');
    }, { passive: false });
}

// ── Hotkey ───────────────────────────────────────────────────

function isTyping() {
    const el = document.activeElement;
    if (!el) return false;
    const tag = el.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true;
    if (el.isContentEditable) return true;
    if (el.closest('.cm-editor')) return true;
    return false;
}

function onKeyDown(e) {
    if (!config.enableHotkey) return;
    if (e.altKey || e.ctrlKey || e.metaKey || e.shiftKey) return;
    if (isTyping()) return;

    // Track keyboard usage for focus styling
    if (e.key === 'Tab') {
        isKeyboard = true;
    }

    if (e.key === 'f' || e.key === 'F') {
        e.preventDefault();
        toggle();
    }
}

// ── Init ─────────────────────────────────────────────────────

function init() {
    // Only activate on pages where sidebar exists
    if (!document.querySelector('aside.sarde-sidebar')) return;

    createButton();
    document.addEventListener('keydown', onKeyDown);

    // Restore persisted state
    if (isEnabled()) {
        applyFocusMode(true);
    }
}

init();
