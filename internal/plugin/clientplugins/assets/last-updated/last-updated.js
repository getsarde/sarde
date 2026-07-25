// Last Updated Badge Plugin
// Reads data-last-modified from the article, formats as relative/absolute date, injects badge.

const cfg = (window.__SARDE__ && window.__SARDE__.pluginConfig && window.__SARDE__.pluginConfig["last-updated"]) || {};

// The page's language, emitted by the Head component. Drives both relative and
// absolute formatting so dates follow the page rather than the visitor's browser.
const lang = (window.__SARDE__ && window.__SARDE__.lang) || undefined;

const config = {
    position: cfg.position || 'bottom',
    useRelativeTime: cfg.use_relative_time !== false,
    dateFormat: cfg.date_format || 'short',
};

const ICON_SVG = '<svg class="sarde-last-updated-icon" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">'
    + '<path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>'
    + '<path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>'
    + '</svg>';

// Intl.RelativeTimeFormat handles translation and pluralization for every
// locale, so relative phrases need no string table of their own.
const supportsRelativeTimeFormat =
    typeof Intl !== 'undefined' && typeof Intl.RelativeTimeFormat === 'function';

const RELATIVE_DIVISIONS = [
    { amount: 60, unit: 'second' },
    { amount: 60, unit: 'minute' },
    { amount: 24, unit: 'hour' },
    { amount: 30, unit: 'day' },
    { amount: 12, unit: 'month' },
    { amount: Infinity, unit: 'year' },
];

function formatRelativeIntl(timestamp) {
    const rtf = new Intl.RelativeTimeFormat(lang, { numeric: 'auto' });
    let elapsed = Date.now() / 1000 - timestamp;
    for (const { amount, unit } of RELATIVE_DIVISIONS) {
        if (Math.abs(elapsed) < amount) {
            return rtf.format(-Math.round(elapsed), unit);
        }
        elapsed /= amount;
    }
    return rtf.format(-Math.round(elapsed), 'year');
}

function formatRelative(timestamp) {
    return supportsRelativeTimeFormat
        ? formatRelativeIntl(timestamp)
        : formatRelativeFallback(timestamp);
}

// English-only fallback for engines without Intl.RelativeTimeFormat.
function formatRelativeFallback(timestamp) {
    const now = Date.now() / 1000;
    const diff = Math.floor(now - timestamp);

    if (diff < 60) return 'Just now';
    if (diff < 3600) {
        const mins = Math.floor(diff / 60);
        return mins === 1 ? '1 minute ago' : mins + ' minutes ago';
    }
    if (diff < 86400) {
        const hours = Math.floor(diff / 3600);
        return hours === 1 ? '1 hour ago' : hours + ' hours ago';
    }
    if (diff < 2592000) {
        const days = Math.floor(diff / 86400);
        return days === 1 ? 'Yesterday' : days + ' days ago';
    }
    if (diff < 31536000) {
        const months = Math.floor(diff / 2592000);
        return months === 1 ? '1 month ago' : months + ' months ago';
    }
    const years = Math.floor(diff / 31536000);
    return years === 1 ? '1 year ago' : years + ' years ago';
}

function formatAbsolute(timestamp, format) {
    const date = new Date(timestamp * 1000);

    if (format === 'iso') {
        return date.toISOString().split('T')[0];
    }

    const options = {
        year: 'numeric',
        month: format === 'long' ? 'long' : 'short',
        day: 'numeric',
    };

    return date.toLocaleDateString(lang, options);
}

// Upgrade a server-rendered badge in place. The LastUpdated component emits an
// absolute date so the page is correct without JS and does not shift on load;
// this rewrites it to the configured format. The template owns placement, so
// the position option does not apply here.
function enhance(existing, timestamp, dateText, label) {
    const time = existing.querySelector('time');
    if (time) {
        time.textContent = dateText;
    } else {
        existing.textContent = label + ': ' + dateText;
    }
    if (config.useRelativeTime) {
        existing.title = formatAbsolute(timestamp, 'long');
    }
}

function init() {
    const article = document.querySelector('article.sarde-markdown-content[data-last-modified]');
    if (!article) return;

    const timestamp = parseInt(article.dataset.lastModified || '', 10);
    if (!timestamp || isNaN(timestamp)) return;

    const label = article.dataset.lastUpdatedLabel || 'Last updated';
    const dateText = config.useRelativeTime
        ? formatRelative(timestamp)
        : formatAbsolute(timestamp, config.dateFormat);

    // A layout may already render the badge server-side. Enhance it rather than
    // adding a second one.
    const existing = document.querySelector('.sarde-last-updated');
    if (existing) {
        enhance(existing, timestamp, dateText, label);
        return;
    }

    // Build badge with safe DOM methods
    const badge = document.createElement('div');
    badge.className = 'sarde-last-updated sarde-last-updated-' + config.position;

    const iconContainer = document.createElement('span');
    iconContainer.innerHTML = ICON_SVG; // hardcoded SVG, no user input
    badge.appendChild(iconContainer);

    const text = document.createElement('span');
    text.textContent = label + ': ' + dateText;
    badge.appendChild(text);

    if (config.useRelativeTime) {
        badge.title = formatAbsolute(timestamp, 'long');
    }

    if (config.position === 'top') {
        const readingTime = document.querySelector('.sarde-reading-time');
        if (readingTime && readingTime.parentNode) {
            if (!readingTime.parentNode.classList.contains('sarde-last-updated-row')) {
                const row = document.createElement('div');
                row.className = 'sarde-last-updated-row';
                readingTime.parentNode.insertBefore(row, readingTime);
                row.appendChild(readingTime);
            }
            readingTime.parentNode.appendChild(badge);
        } else {
            const anchor = document.querySelector('.sarde-page-description') || document.querySelector('.sarde-page-title');
            if (anchor && anchor.parentNode) {
                anchor.parentNode.insertBefore(badge, anchor.nextSibling);
            }
        }
    } else {
        // Page navigation is a sibling of <article>, not a descendant, and
        // carries a sarde- prefixed class. Searching the article for
        // "nav.pagination" never matched, so the badge always fell through to
        // the end of the prose instead of sitting above the navigation.
        const pagination = document.querySelector('nav.sarde-pagination, nav.sarde-post-navigation');
        if (pagination && pagination.parentNode) {
            pagination.parentNode.insertBefore(badge, pagination);
        } else {
            article.appendChild(badge);
        }
    }
}

init();
