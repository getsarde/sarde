const cfg = (window.__pluginConfig && window.__pluginConfig["search-highlighter"]) || {};

const config = {
    highlightColor: cfg.highlight_color || '',
    activeOutlineColor: cfg.active_outline_color || '',
    autoScroll: cfg.auto_scroll !== false,
    showBadge: cfg.show_badge !== false,
};

// Apply custom colors as CSS custom properties
if (config.activeOutlineColor) {
    document.documentElement.style.setProperty('--search-hl-active-outline', config.activeOutlineColor);
}

// ── State ────────────────────────────────────────────────────

let marks = [];
let currentIndex = -1;

let badge = null;

// ── URL Helpers ──────────────────────────────────────────────

function getSearchQuery() {
    try {
        var params = new URLSearchParams(window.location.search);
        var q = params.get('q');
        return q ? q.trim() : '';
    } catch (e) {
        return '';
    }
}

function cleanUrl() {
    try {
        var url = new URL(window.location.href);
        if (url.searchParams.has('q')) {
            url.searchParams.delete('q');
            var clean = url.pathname + (url.search || '') + (url.hash || '');
            window.history.replaceState(null, '', clean);
        }
    } catch (e) {
        // URL API not supported or security restriction
    }
}

// ── Cleanup ──────────────────────────────────────────────────

function cleanup() {
    for (var i = 0; i < marks.length; i++) {
        var mark = marks[i];
        var parent = mark.parentNode;
        if (parent) {
            var text = document.createTextNode(mark.textContent || '');
            parent.replaceChild(text, mark);
            parent.normalize();
        }
    }
    marks = [];
    currentIndex = -1;

    if (badge) {
        badge.remove();
        badge = null;
    }
}

// ── Highlighting ─────────────────────────────────────────────

/** Tags to skip when highlighting (content inside these is left alone) */
const SKIP_TAGS = ['SCRIPT', 'STYLE', 'CODE', 'PRE', 'KBD', 'MARK', 'TEXTAREA', 'INPUT', 'SVG'];

function highlightTerms(container, terms) {
    // Build regex from escaped terms
    var escaped = terms.map(function (t) {
        return t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    });
    var pattern = new RegExp('(' + escaped.join('|') + ')', 'gi');

    // Collect matching text nodes via TreeWalker
    var walker = document.createTreeWalker(
        container,
        NodeFilter.SHOW_TEXT,
        {
            /** @param {Node} node */
            acceptNode: function (node) {
                var parent = node.parentElement;
                if (!parent) return NodeFilter.FILTER_REJECT;
                // Walk up to check if any ancestor is a skip tag
                var el = parent;
                while (el && el !== container) {
                    if (SKIP_TAGS.indexOf(el.tagName) !== -1) {
                        return NodeFilter.FILTER_REJECT;
                    }
                    el = el.parentElement;
                }
                return NodeFilter.FILTER_ACCEPT;
            }
        }
    );

    
    var textNodes = [];
    while (walker.nextNode()) {
        var nodeText = walker.currentNode.textContent || '';
        pattern.lastIndex = 0;
        if (pattern.test(nodeText)) {
            textNodes.push( (walker.currentNode));
        }
    }

    // Process each text node: split on matches, wrap in <mark>
    for (var i = 0; i < textNodes.length; i++) {
        var node = textNodes[i];
        var text = node.textContent || '';
        pattern.lastIndex = 0;

        var frag = document.createDocumentFragment();
        var lastIndex = 0;
        var match;

        while ((match = pattern.exec(text)) !== null) {
            // Text before match
            if (match.index > lastIndex) {
                frag.appendChild(document.createTextNode(text.slice(lastIndex, match.index)));
            }
            // Wrap the match in <mark>
            var mark = document.createElement('mark');
            mark.className = 'sarde-search-hl';
            mark.textContent = match[0];
            if (config.highlightColor) {
                mark.style.backgroundColor = config.highlightColor;
            }
            frag.appendChild(mark);
            marks.push(mark);
            lastIndex = pattern.lastIndex;
        }

        // Remaining text after last match
        if (lastIndex < text.length) {
            frag.appendChild(document.createTextNode(text.slice(lastIndex)));
        }

        node.parentNode.replaceChild(frag, node);
    }
}

// ── Match Navigation ─────────────────────────────────────────

function goToMatch(index) {
    if (marks.length === 0) return;

    // Remove active class from current
    if (currentIndex >= 0 && marks[currentIndex]) {
        marks[currentIndex].classList.remove('is-active');
    }

    // Wrap around
    if (index < 0) index = marks.length - 1;
    if (index >= marks.length) index = 0;

    currentIndex = index;
    marks[currentIndex].classList.add('is-active');

    // Scroll into view
    marks[currentIndex].scrollIntoView({
        behavior: 'smooth',
        block: 'center',
    });

    // Update badge count
    updateBadgeCount();
}

function updateBadgeCount() {
    if (!badge) return;
    var countEl = badge.querySelector('.sarde-search-hl-badge-count');
    if (!countEl) return;

    if (currentIndex >= 0) {
        countEl.textContent = (currentIndex + 1) + ' / ' + marks.length;
    } else {
        countEl.textContent = marks.length + ' match' + (marks.length !== 1 ? 'es' : '');
    }
}

// ── Badge UI ─────────────────────────────────────────────────

function createBadge() {
    if (!config.showBadge) return;

    badge = document.createElement('div');
    badge.className = 'sarde-search-hl-badge';
    badge.setAttribute('role', 'status');
    badge.setAttribute('aria-live', 'polite');

    // Match count
    var count = document.createElement('span');
    count.className = 'sarde-search-hl-badge-count';
    count.textContent = marks.length + ' match' + (marks.length !== 1 ? 'es' : '');
    badge.appendChild(count);

    // Separator
    var sep = document.createElement('span');
    sep.className = 'sarde-search-hl-badge-sep';
    badge.appendChild(sep);

    // Prev button
    var prev = document.createElement('button');
    prev.className = 'sarde-search-hl-badge-btn';
    prev.setAttribute('aria-label', 'Previous match');
    prev.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>';
    prev.addEventListener('click', function () { goToMatch(currentIndex - 1); });
    badge.appendChild(prev);

    // Next button
    var next = document.createElement('button');
    next.className = 'sarde-search-hl-badge-btn';
    next.setAttribute('aria-label', 'Next match');
    next.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>';
    next.addEventListener('click', function () { goToMatch(currentIndex + 1); });
    badge.appendChild(next);

    // Separator
    var sep2 = document.createElement('span');
    sep2.className = 'sarde-search-hl-badge-sep';
    badge.appendChild(sep2);

    // Dismiss button
    var dismiss = document.createElement('button');
    dismiss.className = 'sarde-search-hl-badge-btn sarde-search-hl-badge-btn-dismiss';
    dismiss.setAttribute('aria-label', 'Dismiss highlights');
    dismiss.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>';
    dismiss.addEventListener('click', function () {
        cleanup();
        cleanUrl();
    });
    badge.appendChild(dismiss);

    // Insert at the top of the content area (before the prose div)
    var content = document.querySelector('article.sarde-markdown-content');
    if (content && content.parentNode) {
        content.parentNode.insertBefore(badge, content);
    } else {
        document.body.appendChild(badge);
    }

    // Make visible — force a layout read to ensure the transition plays
    badge.offsetHeight;
    badge.classList.add('is-visible');
}

// ── Init ─────────────────────────────────────────────────────

function init() {
    cleanup();

    var query = getSearchQuery();
    if (!query) return;

    var container = document.querySelector('article.sarde-markdown-content');
    if (!container) return;

    // Split query into terms, filter out single-char terms
    var terms = query.split(/\s+/).filter(function (t) { return t.length > 1; });
    if (terms.length === 0) return;

    highlightTerms( (container), terms);

    if (marks.length === 0) {
        cleanUrl();
        return;
    }

    createBadge();

    if (config.autoScroll) {
        setTimeout(function () { goToMatch(0); }, 200);
    }
}

// ── Bootstrap ────────────────────────────────────────────────

init();
