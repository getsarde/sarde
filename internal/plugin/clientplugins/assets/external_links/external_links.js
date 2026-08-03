const cfg = (window.__SARDE__ && window.__SARDE__.pluginConfig && window.__SARDE__.pluginConfig["external_links"]) || {};

const config = {
    showIcon: cfg.show_icon !== false,
    openInNewTab: cfg.open_in_new_tab !== false,
    excludeDomains: parseExcludeDomains(cfg.exclude_domains || ''),
};

// External-link SVG icon (arrow-up-right)
const ICON_SVG = '<svg class="sarde-external-link-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">'
    + '<path d="M7 7h10v10"/><path d="M7 17 17 7"/>'
    + '</svg>';

// Parse comma-separated domain list into a Set.
function parseExcludeDomains(raw) {
    const set = new Set();
    if (!raw) return set;
    raw.split(',').forEach(function (d) {
        const trimmed = d.trim().toLowerCase();
        if (trimmed) set.add(trimmed);
    });
    return set;
}

// Check if a URL is external.
function isExternal(link) {
    // Skip non-http links (mailto:, tel:, javascript:, #anchors)
    const href = link.getAttribute('href') || '';
    if (!href || href.startsWith('#') || href.startsWith('mailto:') || href.startsWith('tel:') || href.startsWith('javascript:')) {
        return false;
    }

    // Compare hostnames
    try {
        // link.hostname is resolved by the browser
        if (link.hostname === window.location.hostname) return false;
        if (!link.hostname) return false;

        // Check excluded domains
        const host = link.hostname.toLowerCase();
        if (config.excludeDomains.has(host)) return false;

        return true;
    } catch (e) {
        return false;
    }
}

// Process all external links within a container.
function processLinks(container) {
    const links = container.querySelectorAll('a[href]');

    for (let i = 0; i < links.length; i++) {
        const link = links[i];

        // Skip already processed
        if (link.dataset.externalProcessed) continue;
        link.dataset.externalProcessed = 'true';

        if (!isExternal(link)) continue;

        // Set new tab behavior
        if (config.openInNewTab) {
            link.setAttribute('target', '_blank');
            link.setAttribute('rel', 'noopener noreferrer');
        }

        // Append icon
        if (config.showIcon) {
            link.insertAdjacentHTML('beforeend', ICON_SVG);
        }
    }
}

// Process links in the main content area.
function init() {
    const content = document.querySelector('article.sarde-markdown-content');
    if (content) {
        processLinks(content);
    }
}

init();
