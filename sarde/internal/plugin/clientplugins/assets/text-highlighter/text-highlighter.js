const cfg = (window.__pluginConfig && window.__pluginConfig["text-highlighter"]) || {};

const config = {
    maxHighlights: cfg.max_highlights_per_page || 50,
    contextChars: cfg.context_chars || 40,
};

const STORAGE_PREFIX = 'thl:';
const COLORS = ['yellow', 'green', 'blue', 'pink', 'orange'];
const SKIP_TAGS = ['SCRIPT', 'STYLE', 'CODE', 'PRE', 'KBD', 'TEXTAREA', 'INPUT', 'SVG'];

// ── State ────────────────────────────────────────────────────

let highlights = [];

let toolbar = null;

let removePopup = null;
let toolbarVisible = false;
let removeVisible = false;

// ── Storage ──────────────────────────────────────────────────

function getPageKey() {
    return STORAGE_PREFIX + window.location.pathname;
}

function loadHighlights() {
    try {
        var data = localStorage.getItem(getPageKey());
        if (data) {
            var parsed = JSON.parse(data);
            if (Array.isArray(parsed)) return parsed;
        }
    } catch (e) { /* ignore */ }
    return [];
}

function saveHighlights() {
    try {
        // Enforce max
        while (highlights.length > config.maxHighlights) {
            highlights.shift(); // remove oldest
        }
        if (highlights.length === 0) {
            localStorage.removeItem(getPageKey());
        } else {
            localStorage.setItem(getPageKey(), JSON.stringify(highlights));
        }
    } catch (e) { /* storage full */ }
}

// ── Prose Container ──────────────────────────────────────────

function getProseContainer() {
    return document.querySelector('article.sarde-markdown-content');
}

function isInSkipTag(node, container) {
    var el = node.nodeType === 1 ?  (node) : node.parentElement;
    while (el && el !== container) {
        if (SKIP_TAGS.indexOf(el.tagName) !== -1) return true;
        // Skip search-highlighter marks but not our own
        if (el.tagName === 'MARK' && el.classList.contains('sarde-search-hl')) return true;
        el = el.parentElement;
    }
    return false;
}

// ── Context Extraction ───────────────────────────────────────

function getContainerText(container) {
    var walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, {
        acceptNode: function (node) {
            return isInSkipTag(node, container) ? NodeFilter.FILTER_REJECT : NodeFilter.FILTER_ACCEPT;
        }
    });
    var text = '';
    while (walker.nextNode()) {
        text += walker.currentNode.textContent || '';
    }
    return text;
}

function getTextOffset(container, targetNode, targetOffset) {
    var walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, {
        acceptNode: function (node) {
            return isInSkipTag(node, container) ? NodeFilter.FILTER_REJECT : NodeFilter.FILTER_ACCEPT;
        }
    });
    var offset = 0;
    while (walker.nextNode()) {
        if (walker.currentNode === targetNode) {
            return offset + targetOffset;
        }
        offset += (walker.currentNode.textContent || '').length;
    }
    return offset;
}

function extractDescriptor(container, range, color) {
    var selectedText = range.toString();
    if (!selectedText || !selectedText.trim()) return null;

    var fullText = getContainerText(container);
    var startOffset = getTextOffset(container, range.startContainer, range.startOffset);
    var endOffset = startOffset + selectedText.length;

    var prefix = fullText.slice(Math.max(0, startOffset - config.contextChars), startOffset);
    var suffix = fullText.slice(endOffset, endOffset + config.contextChars);

    return {
        text: selectedText,
        prefix: prefix,
        suffix: suffix,
        color: color,
        timestamp: Date.now(),
    };
}

// ── Restore Algorithm ────────────────────────────────────────

function findMatch(fullText, desc) {
    var searchText = desc.text;
    if (!searchText) return null;

    var bestScore = -1;
    var bestStart = -1;
    var idx = 0;

    while (true) {
        idx = fullText.indexOf(searchText, idx);
        if (idx === -1) break;

        // Score by prefix/suffix similarity
        var actualPrefix = fullText.slice(Math.max(0, idx - desc.prefix.length), idx);
        var actualSuffix = fullText.slice(idx + searchText.length, idx + searchText.length + desc.suffix.length);
        var score = similarity(desc.prefix, actualPrefix) + similarity(desc.suffix, actualSuffix);

        if (score > bestScore) {
            bestScore = score;
            bestStart = idx;
        }
        idx++;
    }

    // Require at least 40% match on context
    if (bestStart === -1 || bestScore < 0.4) return null;

    return { start: bestStart, end: bestStart + searchText.length };
}

function similarity(a, b) {
    if (!a.length && !b.length) return 1;
    if (!a.length || !b.length) return 0;
    var maxLen = Math.max(a.length, b.length);
    var matches = 0;
    var minLen = Math.min(a.length, b.length);
    for (var i = 0; i < minLen; i++) {
        if (a[a.length - minLen + i] === b[b.length - minLen + i]) matches++;
    }
    return matches / maxLen;
}

function offsetToRange(container, start, end) {
    var walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, {
        acceptNode: function (node) {
            return isInSkipTag(node, container) ? NodeFilter.FILTER_REJECT : NodeFilter.FILTER_ACCEPT;
        }
    });

    var offset = 0;
    var range = document.createRange();
    var foundStart = false;

    while (walker.nextNode()) {
        var node = walker.currentNode;
        var len = (node.textContent || '').length;

        if (!foundStart && offset + len > start) {
            range.setStart(node, start - offset);
            foundStart = true;
        }
        if (foundStart && offset + len >= end) {
            range.setEnd(node, end - offset);
            return range;
        }
        offset += len;
    }
    return null;
}

// ── DOM Manipulation ─────────────────────────────────────────

function wrapRange(range, color, hlId) {
    // Single text node case
    if (range.startContainer === range.endContainer && range.startContainer.nodeType === 3) {
        var mark = createMark(color, hlId);
        range.surroundContents(mark);
        return;
    }

    // Multi-node: collect text nodes in range, then wrap each
    var container = getProseContainer();
    if (!container) return;

    var textNodes = [];
    var walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, {
        acceptNode: function (node) {
            return isInSkipTag(node, container) ? NodeFilter.FILTER_REJECT : NodeFilter.FILTER_ACCEPT;
        }
    });

    var inRange = false;
    while (walker.nextNode()) {
        var node = walker.currentNode;
        if (node === range.startContainer) inRange = true;
        if (inRange) textNodes.push(node);
        if (node === range.endContainer) break;
    }

    // Wrap each node (split start/end as needed)
    for (var i = 0; i < textNodes.length; i++) {
        var tn = textNodes[i];
        var r = document.createRange();

        if (tn === range.startContainer) {
            r.setStart(tn, range.startOffset);
            r.setEnd(tn, (tn.textContent || '').length);
        } else if (tn === range.endContainer) {
            r.setStart(tn, 0);
            r.setEnd(tn, range.endOffset);
        } else {
            r.selectNodeContents(tn);
        }

        if (r.toString().length > 0) {
            var m = createMark(color, hlId);
            r.surroundContents(m);
        }
    }
}

function createMark(color, hlId) {
    var mark = document.createElement('mark');
    mark.className = 'sarde-text-hl';
    mark.setAttribute('data-color', color);
    mark.setAttribute('data-thl-id', String(hlId));
    return mark;
}

function unwrapHighlight(hlId) {
    var marks = document.querySelectorAll('mark.sarde-text-hl[data-thl-id="' + hlId + '"]');
    for (var i = 0; i < marks.length; i++) {
        var mark = marks[i];
        var parent = mark.parentNode;
        if (parent) {
            while (mark.firstChild) {
                parent.insertBefore(mark.firstChild, mark);
            }
            parent.removeChild(mark);
            parent.normalize();
        }
    }
}

function unwrapAll() {
    var marks = document.querySelectorAll('mark.sarde-text-hl');
    for (var i = 0; i < marks.length; i++) {
        var mark = marks[i];
        var parent = mark.parentNode;
        if (parent) {
            while (mark.firstChild) {
                parent.insertBefore(mark.firstChild, mark);
            }
            parent.removeChild(mark);
            parent.normalize();
        }
    }
}

function restoreAllHighlights() {
    unwrapAll();
    highlights = loadHighlights();

    var container = getProseContainer();
    if (!container || highlights.length === 0) return;

    var fullText = getContainerText(container);
    var valid = [];

    for (var i = 0; i < highlights.length; i++) {
        var desc = highlights[i];
        var match = findMatch(fullText, desc);
        if (!match) continue; // content changed too much, drop silently

        var range = offsetToRange(container, match.start, match.end);
        if (!range) continue;

        wrapRange(range, desc.color, i);
        valid.push(desc);
    }

    // Self-heal: remove highlights that could no longer be located
    if (valid.length !== highlights.length) {
        highlights = valid;
        saveHighlights();
    }
}

// ── Toolbar UI ───────────────────────────────────────────────

function ensureToolbar() {
    if (toolbar) return;

    toolbar = document.createElement('div');
    toolbar.className = 'sarde-thl-toolbar';
    toolbar.setAttribute('role', 'toolbar');
    toolbar.setAttribute('aria-label', 'Highlight colors');

    for (var i = 0; i < COLORS.length; i++) {
        var btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'sarde-thl-toolbar-color';
        btn.setAttribute('data-color', COLORS[i]);
        btn.setAttribute('aria-label', COLORS[i] + ' highlight');
        btn.addEventListener('click', onColorClick);
        toolbar.appendChild(btn);
    }

    // Separator
    var sep = document.createElement('span');
    sep.className = 'sarde-thl-toolbar-sep';
    toolbar.appendChild(sep);

    // Clear all button
    var clearBtn = document.createElement('button');
    clearBtn.type = 'button';
    clearBtn.className = 'sarde-thl-toolbar-clear';
    clearBtn.setAttribute('aria-label', 'Clear all highlights');
    clearBtn.title = 'Clear all highlights';
    clearBtn.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>';
    clearBtn.addEventListener('click', onClearAll);
    toolbar.appendChild(clearBtn);

    document.body.appendChild(toolbar);
}

function showToolbar(rect) {
    ensureToolbar();
    if (!toolbar) return;

    // Temporarily show to measure
    toolbar.style.visibility = 'hidden';
    toolbar.style.opacity = '0';
    toolbar.style.pointerEvents = 'none';
    toolbar.classList.add('is-visible');

    var tw = toolbar.offsetWidth;
    var th = toolbar.offsetHeight;
    var gap = 8;

    // Prefer above the selection; fall back to below
    var top = rect.top - th - gap;
    if (top < 4) top = rect.bottom + gap;

    // Center horizontally, clamp to viewport
    var left = rect.left + (rect.width / 2) - (tw / 2);
    left = Math.max(4, Math.min(left, window.innerWidth - tw - 4));

    toolbar.style.top = top + 'px';
    toolbar.style.left = left + 'px';
    toolbar.style.visibility = '';
    toolbar.style.opacity = '';
    toolbar.style.pointerEvents = '';

    toolbarVisible = true;
}

function hideToolbar() {
    if (toolbar) toolbar.classList.remove('is-visible');
    toolbarVisible = false;
}

// ── Remove Popup ─────────────────────────────────────────────

function showRemovePopup(markEl) {
    hideRemovePopup();

    removePopup = document.createElement('div');
    removePopup.className = 'sarde-thl-remove';

    var removeBtn = document.createElement('button');
    removeBtn.type = 'button';
    removeBtn.className = 'sarde-thl-remove-btn sarde-thl-remove-btn-danger';
    removeBtn.textContent = 'Remove';
    removeBtn.addEventListener('click', function () {
        var hlId = parseInt(markEl.getAttribute('data-thl-id') || '-1', 10);
        if (hlId >= 0 && hlId < highlights.length) {
            unwrapHighlight(hlId);
            highlights.splice(hlId, 1);
            saveHighlights();
            // Re-index remaining marks
            restoreAllHighlights();
        }
        hideRemovePopup();
    });
    removePopup.appendChild(removeBtn);

    document.body.appendChild(removePopup);

    // Position near the mark
    var rect = markEl.getBoundingClientRect();
    var pw = removePopup.offsetWidth;
    var top = rect.top - removePopup.offsetHeight - 6;
    if (top < 4) top = rect.bottom + 6;
    var left = rect.left + (rect.width / 2) - (pw / 2);
    left = Math.max(4, Math.min(left, window.innerWidth - pw - 4));

    removePopup.style.top = top + 'px';
    removePopup.style.left = left + 'px';

    removePopup.offsetHeight;
    removePopup.classList.add('is-visible');
    removeVisible = true;
}

function hideRemovePopup() {
    if (removePopup) {
        removePopup.remove();
        removePopup = null;
    }
    removeVisible = false;
}

// ── Event Handlers ───────────────────────────────────────────

/** @param {MouseEvent} e */
function onColorClick(e) {
    e.stopPropagation();
    var btn =  (e.currentTarget);
    var color = btn.getAttribute('data-color');
    if (!color) return;

    var sel = window.getSelection();
    if (!sel || sel.isCollapsed || sel.rangeCount === 0) {
        hideToolbar();
        return;
    }

    var range = sel.getRangeAt(0);
    var container = getProseContainer();
    if (!container) return;

    // Check selection is within the prose container
    if (!container.contains(range.commonAncestorContainer)) {
        hideToolbar();
        return;
    }

    // Remove any overlapping highlights first
    removeOverlapping(range);

    // Extract descriptor
    var desc = extractDescriptor(container, range, color);
    if (!desc) { hideToolbar(); return; }

    // Apply visual highlight
    var hlId = highlights.length;
    wrapRange(range, color, hlId);

    // Store
    highlights.push(desc);
    saveHighlights();

    // Clean up
    sel.removeAllRanges();
    hideToolbar();
}

function removeOverlapping(range) {
    var marks = document.querySelectorAll('mark.sarde-text-hl');
    var idsToRemove = {};

    for (var i = 0; i < marks.length; i++) {
        var mark = marks[i];
        if (range.intersectsNode(mark)) {
            var id = mark.getAttribute('data-thl-id');
            if (id !== null) idsToRemove[id] = true;
        }
    }

    // Remove from highest ID first to preserve indices
    var ids = Object.keys(idsToRemove).map(Number).sort(function (a, b) { return b - a; });
    for (var j = 0; j < ids.length; j++) {
        unwrapHighlight(ids[j]);
        highlights.splice(ids[j], 1);
    }
}

function onClearAll(e) {
    e.stopPropagation();
    unwrapAll();
    highlights = [];
    saveHighlights();
    hideToolbar();
    hideRemovePopup();
}

let mouseupTimer = null;

/** @param {MouseEvent} e */
function onMouseUp(e) {
    // Don't interfere with toolbar/popup clicks
    if (toolbar && toolbar.contains( (e.target))) return;
    if (removePopup && removePopup.contains( (e.target))) return;

    clearTimeout(mouseupTimer);
    mouseupTimer = setTimeout(function () {
        var sel = window.getSelection();
        if (!sel || sel.isCollapsed || sel.rangeCount === 0) {
            hideToolbar();
            return;
        }

        var range = sel.getRangeAt(0);
        var selectedText = sel.toString().trim();
        if (!selectedText) { hideToolbar(); return; }

        var container = getProseContainer();
        if (!container || !container.contains(range.commonAncestorContainer)) {
            hideToolbar();
            return;
        }

        // Don't show toolbar for selections inside skip tags
        if (isInSkipTag(range.startContainer, container)) {
            hideToolbar();
            return;
        }

        showToolbar(range.getBoundingClientRect());
    }, 10);
}

/** @param {MouseEvent} e */
function onMouseDown(e) {
    var target =  (e.target);

    // Clicking a highlight mark — show remove popup
    if (target.closest && target.closest('mark.sarde-text-hl')) {
        var mark =  (target.closest('mark.sarde-text-hl'));
        // Only show remove if no text is being selected
        setTimeout(function () {
            var sel = window.getSelection();
            if (!sel || sel.isCollapsed) {
                showRemovePopup(mark);
            }
        }, 150);
        return;
    }

    // Clicking outside toolbar/popup — hide them
    if (toolbar && !toolbar.contains(target)) hideToolbar();
    if (removePopup && !removePopup.contains(target)) hideRemovePopup();
}

/** @param {KeyboardEvent} e */
function onKeyDown(e) {
    if (e.key === 'Escape') {
        if (toolbarVisible) hideToolbar();
        if (removeVisible) hideRemovePopup();
    }
}

// ── Pruning ──────────────────────────────────────────────────

function maybePrune() {
    if (Math.random() > 0.1) return; // 1 in 10 chance
    var maxAge = 90 * 24 * 60 * 60 * 1000; // 90 days
    var now = Date.now();
    try {
        for (var i = localStorage.length - 1; i >= 0; i--) {
            var key = localStorage.key(i);
            if (!key || key.indexOf(STORAGE_PREFIX) !== 0) continue;
            var data = JSON.parse(localStorage.getItem(key) || '[]');
            if (Array.isArray(data) && data.length > 0 && data[0].timestamp) {
                // Check if the newest highlight is older than maxAge
                var newest = data[data.length - 1].timestamp || 0;
                if (now - newest > maxAge) {
                    localStorage.removeItem(key);
                }
            }
        }
    } catch (e) { /* ignore */ }
}

// ── Init ─────────────────────────────────────────────────────

function init() {
    if (!document.querySelector('article.sarde-markdown-content')) return;

    restoreAllHighlights();
    maybePrune();

    document.addEventListener('mouseup', onMouseUp);
    document.addEventListener('mousedown', onMouseDown);
    document.addEventListener('keydown', onKeyDown);
}

init();
