// Last Updated Badge Plugin
// Reads data-last-modified from the article, formats as relative/absolute date, injects badge.

const cfg = (window.__pluginConfig && window.__pluginConfig["last-updated"]) || {};

const config = {
    position: cfg.position || 'bottom',
    useRelativeTime: cfg.use_relative_time !== false,
    dateFormat: cfg.date_format || 'short',
};

const ICON_SVG = '<svg class="last-updated__icon" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">'
    + '<path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>'
    + '<path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>'
    + '</svg>';

function formatRelative(timestamp) {
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

    return date.toLocaleDateString(undefined, options);
}

function init() {
    if (document.querySelector('.sarde-last-updated')) return;

    const article = document.querySelector('article.sarde-markdown-content[data-last-modified]');
    if (!article) return;

    const timestamp = parseInt(article.dataset.lastModified || '', 10);
    if (!timestamp || isNaN(timestamp)) return;

    const dateText = config.useRelativeTime
        ? formatRelative(timestamp)
        : formatAbsolute(timestamp, config.dateFormat);

    // Build badge with safe DOM methods
    const badge = document.createElement('div');
    badge.className = 'last-updated last-updated--' + config.position;

    const iconContainer = document.createElement('span');
    iconContainer.innerHTML = ICON_SVG; // hardcoded SVG, no user input
    badge.appendChild(iconContainer);

    const text = document.createElement('span');
    text.textContent = 'Last updated: ' + dateText;
    badge.appendChild(text);

    if (config.useRelativeTime) {
        badge.title = formatAbsolute(timestamp, 'long');
    }

    if (config.position === 'top') {
        const readingTime = document.querySelector('.sarde-reading-time');
        if (readingTime && readingTime.parentNode) {
            if (!readingTime.parentNode.classList.contains('last-updated-row')) {
                const row = document.createElement('div');
                row.className = 'last-updated-row';
                readingTime.parentNode.insertBefore(row, readingTime);
                row.appendChild(readingTime);
            }
            readingTime.parentNode.appendChild(badge);
        } else {
            const anchor = document.querySelector('.page-description') || document.querySelector('.page-title');
            if (anchor && anchor.parentNode) {
                anchor.parentNode.insertBefore(badge, anchor.nextSibling);
            }
        }
    } else {
        const pagination = article.querySelector('nav.pagination');
        if (pagination) {
            pagination.parentNode.insertBefore(badge, pagination);
        } else {
            article.appendChild(badge);
        }
    }
}

init();
