// Reading Progress Plugin
// Scroll progress bar fixed at the top of the page.
// Estimated reading time badge injected into lesson headers.
// Content-aware: only activates on lesson pages (configurable).

const cfg = (window.__SARDE__ && window.__SARDE__.pluginConfig && window.__SARDE__.pluginConfig["reading-progress"]) || {};

const config = {
    barHeight: cfg.bar_height || 3,
    barColor: cfg.bar_color || '#6366f1',
    useGradient: cfg.use_gradient !== false,
    lessonPagesOnly: cfg.lesson_pages_only !== false,
    showReadingTime: cfg.show_reading_time !== false,
    wordsPerMinute: cfg.words_per_minute || 200,
};

// ── Color Utilities ────────────────────────────────────────────

function hexToHsl(hex) {
    hex = hex.replace(/^#/, '');
    if (hex.length === 3) {
        hex = hex[0] + hex[0] + hex[1] + hex[1] + hex[2] + hex[2];
    }
    const r = parseInt(hex.substring(0, 2), 16) / 255;
    const g = parseInt(hex.substring(2, 4), 16) / 255;
    const b = parseInt(hex.substring(4, 6), 16) / 255;

    const max = Math.max(r, g, b);
    const min = Math.min(r, g, b);
    let h = 0, s = 0;
    const l = (max + min) / 2;

    if (max !== min) {
        const d = max - min;
        s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
        if (max === r) h = ((g - b) / d + (g < b ? 6 : 0)) / 6;
        else if (max === g) h = ((b - r) / d + 2) / 6;
        else h = ((r - g) / d + 4) / 6;
    }

    return { h: Math.round(h * 360), s: Math.round(s * 100), l: Math.round(l * 100) };
}

// Generate a CSS gradient from a base color by shifting hue. Creates a vibrant 3-stop gradient.
function buildGradient(hex) {
    const hsl = hexToHsl(hex);
    const h = hsl.h, s = hsl.s, l = hsl.l;

    // Three stops: hue-shifted left, base, hue-shifted right
    const h1 = (h - 40 + 360) % 360;
    const h2 = h;
    const h3 = (h + 50) % 360;

    return 'linear-gradient(90deg, hsl(' + h1 + ',' + s + '%,' + l + '%), hsl(' + h2 + ',' + s + '%,' + l + '%), hsl(' + h3 + ',' + s + '%,' + l + '%))';
}

function resolveBarColor() {
    let color = config.barColor;
    if (!color || color === '') color = '#6366f1';
    if (config.useGradient) return buildGradient(color);
    return color;
}

// ── Helpers ────────────────────────────────────────────────────

function isLessonPage() {
    return !!document.querySelector('article.sarde-markdown-content');
}

function countWords(el) {
    const text = el.textContent || '';
    const words = text.trim().split(/\s+/);
    return words.length > 0 && words[0] !== '' ? words.length : 0;
}

function formatReadingTime(minutes) {
    if (minutes < 1) return 'Less than 1 min read';
    if (minutes === 1) return '1 min read';
    return Math.ceil(minutes) + ' min read';
}

// ── Progress Bar ───────────────────────────────────────────────

let progressBarEl = null;

function createProgressBar() {
    // Remove existing bar if present
    const existing = document.getElementById('sarde-reading-progress');
    if (existing) existing.remove();

    const container = document.createElement('div');
    container.className = 'sarde-reading-progress';
    container.id = 'sarde-reading-progress';
    container.style.setProperty('--rp-bar-height', config.barHeight + 'px');

    const bar = document.createElement('div');
    bar.className = 'sarde-reading-progress-bar';
    bar.id = 'reading-progress-bar';
    bar.style.background = resolveBarColor();

    container.appendChild(bar);
    document.body.appendChild(container);

    progressBarEl = bar;
}

function updateProgress() {
    if (!progressBarEl) return;
    const scrollTop = window.scrollY;
    const docHeight = document.documentElement.scrollHeight - window.innerHeight;
    if (docHeight <= 0) {
        progressBarEl.style.width = '0%';
        return;
    }
    const progress = Math.min((scrollTop / docHeight) * 100, 100);
    progressBarEl.style.width = progress + '%';
}

// ── Reading Time ───────────────────────────────────────────────

function injectReadingTime() {
    // Don't duplicate
    if (document.querySelector('.sarde-reading-time')) return;

    const contentEl = document.querySelector('article.sarde-markdown-content');
    if (!contentEl) return;

    const words = countWords(contentEl);
    const minutes = words / config.wordsPerMinute;
    const timeText = formatReadingTime(minutes);

    // Clock SVG icon
    const clockSvg = '<svg class="sarde-reading-time-icon" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>';

    const badge = document.createElement('div');
    badge.className = 'sarde-reading-time';
    badge.innerHTML = clockSvg + '<span>' + timeText + '</span>';

    // Insert after page-tags, page-description, or page-title (whichever is last)
    const anchor = document.querySelector('.sarde-page-tags') || document.querySelector('.sarde-page-description') || document.querySelector('.sarde-page-title');
    if (anchor && anchor.parentNode) {
        anchor.parentNode.insertBefore(badge, anchor.nextSibling);
    }
}

// ── Initialization ─────────────────────────────────────────────

function init() {
    const shouldShow = !config.lessonPagesOnly || isLessonPage();

    if (shouldShow) {
        createProgressBar();
        window.addEventListener('scroll', updateProgress, { passive: true });
        updateProgress();

        if (config.showReadingTime) {
            injectReadingTime();
        }
    } else {
        // Remove bar if on a non-lesson page
        const existing = document.getElementById('sarde-reading-progress');
        if (existing) existing.remove();
        progressBarEl = null;
    }
}

init();
