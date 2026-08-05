// Telescope: command-palette page navigation. Opens with Ctrl+/ (Cmd+/ on
// Mac), fuzzy-searches the build-time page index with Fuse.js, and keeps
// pinned and recently visited pages in localStorage (paths only; titles and
// descriptions are always re-resolved from the fetched index, so stored data
// self-heals when pages move or disappear).
(function () {
    'use strict';

    const sarde = window.__SARDE__ || {};
    const cfg = (sarde.pluginConfig && sarde.pluginConfig.telescope) || {};

    const FALLBACK = {
        trigger: 'Quick navigation',
        tab_search: 'Search',
        tab_recent: 'Recent',
        placeholder: 'Search pages...',
        filter_recent: 'Filter recent pages...',
        pin: 'Pin page',
        unpin: 'Unpin page',
        clear: 'Clear',
        clear_pinned: 'Clear pinned pages',
        clear_recent: 'Clear recent pages',
        no_recent: 'No recently visited pages',
        no_results: 'No pages found matching your search',
        close: 'Close',
        loading: 'Loading pages...',
        pinned_section: 'Pinned Pages',
        recent_section: 'Recently Visited',
        results_section: 'Search Results',
        results_count: '{count} results found',
        showing_of: 'Showing {shown} of {total} results. Refine your search to see more.',
        dialog_opened: 'Search dialog opened',
        recent_cleared: 'Recent pages cleared',
        pinned_cleared: 'Pinned pages cleared',
        hint_navigate: 'navigate',
        hint_select: 'select',
        hint_pin: 'pin',
        hint_close: 'close'
    };

    // Server-resolved i18n label, falling back to English when the key is
    // missing or came through unresolved (raw "telescope.*" key).
    function str(key) {
        const v = cfg.strings && cfg.strings[key];
        if (typeof v === 'string' && v !== '' && v.indexOf('telescope.') !== 0) return v;
        return FALLBACK[key] || key;
    }

    const SHORTCUT_KEY = (typeof cfg.shortcut_key === 'string' && cfg.shortcut_key.length > 0) ? cfg.shortcut_key : '/';
    const MAX_RESULTS = cfg.max_results > 0 ? cfg.max_results : 20;
    const MAX_RECENT = cfg.max_recent > 0 ? cfg.max_recent : 10;
    const MAX_PINNED = cfg.max_pinned > 0 ? cfg.max_pinned : 50;
    const DEBOUNCE_MS = (cfg.debounce_ms != null && cfg.debounce_ms >= 0) ? cfg.debounce_ms : 120;
    const DEFAULT_TAB = cfg.default_tab === 'recent' ? 'recent' : 'search';
    const RECENT_STORAGE_CAP = 50;

    const SITE_ID = sarde.siteId || 'site';
    const KEY_PINNED = 'telescope:' + SITE_ID + ':pinned';
    const KEY_RECENT = 'telescope:' + SITE_ID + ':recent';

    const BASE = (sarde.basePath || '/').replace(/\/+$/, '') + '/';
    const LANG = sarde.lang || '';

    // ── State ──
    let allPages = [];
    let byPath = new Map();
    let filteredPages = [];
    let searchResultsWithMatches = [];
    let searchQuery = '';
    let selectedIndex = 0;
    let currentTab = DEFAULT_TAB;
    let fuseInstance = null;
    let isLoading = false;
    let fetchDone = false;
    let fetchInProgress = null;
    let hasMouseMovedSinceOpen = false;
    let isInNavigationMode = false;
    let debounceTimeout = null;

    // ── localStorage (path arrays only) ──
    function loadPaths(key) {
        try {
            const raw = localStorage.getItem(key);
            const parsed = raw ? JSON.parse(raw) : [];
            return Array.isArray(parsed) ? parsed.filter(function (p) { return typeof p === 'string'; }) : [];
        } catch (_e) {
            return [];
        }
    }
    function savePaths(key, paths) {
        try {
            localStorage.setItem(key, JSON.stringify(paths));
        } catch (_e) { /* storage full or unavailable */ }
    }

    let recentPaths = loadPaths(KEY_RECENT);
    let pinnedPaths = loadPaths(KEY_PINNED);

    function normalizePath(path) {
        if (!path) return '/';
        // Pretty URLs end with a slash; leave file-like paths (with an
        // extension) alone.
        if (path.charAt(path.length - 1) !== '/' && path.lastIndexOf('.') <= path.lastIndexOf('/')) {
            return path + '/';
        }
        return path;
    }

    function recordVisit(path) {
        path = normalizePath(path);
        recentPaths = recentPaths.filter(function (p) { return p !== path; });
        recentPaths.unshift(path);
        recentPaths = recentPaths.slice(0, RECENT_STORAGE_CAP);
        savePaths(KEY_RECENT, recentPaths);
    }

    // Stored paths are only trusted once the index is loaded; unknown paths
    // (deleted pages, other sites on the same origin) are dropped.
    function validPages(paths) {
        if (allPages.length === 0) return [];
        const out = [];
        for (let i = 0; i < paths.length; i++) {
            const page = byPath.get(paths[i]);
            if (page) out.push(page);
        }
        return out;
    }

    function isPinned(path) {
        return pinnedPaths.indexOf(path) !== -1;
    }

    function togglePin(path) {
        const idx = pinnedPaths.indexOf(path);
        if (idx > -1) {
            pinnedPaths.splice(idx, 1);
        } else {
            if (pinnedPaths.length >= MAX_PINNED) pinnedPaths.shift();
            pinnedPaths.push(path);
        }
        savePaths(KEY_PINNED, pinnedPaths);
        renderSearchResults();
        renderRecentResults();
    }

    function clearRecent() {
        recentPaths = [];
        try { localStorage.removeItem(KEY_RECENT); } catch (_e) { /* ignore */ }
        renderSearchResults();
        renderRecentResults();
        announce(str('recent_cleared'));
    }

    function clearPinned() {
        pinnedPaths = [];
        try { localStorage.removeItem(KEY_PINNED); } catch (_e) { /* ignore */ }
        renderSearchResults();
        renderRecentResults();
        announce(str('pinned_cleared'));
    }

    // ── Index fetch (lazy, on first open) ──
    function ensurePages() {
        if (fetchDone) return Promise.resolve();
        if (fetchInProgress) return fetchInProgress;
        isLoading = true;
        updateLoadingState();
        fetchInProgress = fetch(BASE + 'telescope-pages.json')
            .then(function (response) {
                if (!response.ok) throw new Error('telescope index fetch failed: ' + response.status);
                return response.json();
            })
            .then(function (entries) {
                if (!Array.isArray(entries)) entries = [];
                allPages = entries.filter(function (e) {
                    return e && typeof e.path === 'string' && (!LANG || !e.lang || e.lang === LANG);
                });
                byPath = new Map();
                allPages.forEach(function (e) { byPath.set(e.path, e); });

                // Prune stored paths the index no longer knows.
                const validRecent = recentPaths.filter(function (p) { return byPath.has(p); });
                if (validRecent.length !== recentPaths.length) {
                    recentPaths = validRecent;
                    savePaths(KEY_RECENT, recentPaths);
                }
                const validPinned = pinnedPaths.filter(function (p) { return byPath.has(p); });
                if (validPinned.length !== pinnedPaths.length) {
                    pinnedPaths = validPinned;
                    savePaths(KEY_PINNED, pinnedPaths);
                }

                if (typeof Fuse !== 'undefined' && allPages.length > 0) {
                    fuseInstance = new Fuse(allPages, {
                        keys: [
                            { name: 'title', weight: 1.0 },
                            { name: 'path', weight: 0.6 },
                            { name: 'tags', weight: 0.5 },
                            { name: 'collection', weight: 0.4 },
                            { name: 'description', weight: 0.3 }
                        ],
                        threshold: 0.3,
                        ignoreLocation: true,
                        distance: 100,
                        minMatchCharLength: 2,
                        findAllMatches: false,
                        useExtendedSearch: true,
                        includeScore: true,
                        includeMatches: true
                    });
                }
                fetchDone = true;
            })
            .catch(function (_err) {
                allPages = [];
                byPath = new Map();
            })
            .finally(function () {
                isLoading = false;
                fetchInProgress = null;
                updateLoadingState();
            });
        return fetchInProgress;
    }

    // ── DOM helpers ──
    function $(id) { return document.getElementById(id); }

    function escapeHtml(text) {
        const map = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };
        return String(text).replace(/[&<>"']/g, function (ch) { return map[ch]; });
    }

    function announce(message) {
        const live = $('sarde-telescope-live');
        if (!live) return;
        live.textContent = '';
        requestAnimationFrame(function () { live.textContent = message; });
    }

    function updateLoadingState() {
        const loadingEl = $('sarde-telescope-loading');
        const resultsEl = $('sarde-telescope-results');
        if (loadingEl) loadingEl.style.display = isLoading ? 'flex' : 'none';
        if (resultsEl) resultsEl.style.display = isLoading ? 'none' : 'block';
    }

    // ── Modal ──
    function modalHTML() {
        return '<dialog id="sarde-telescope-dialog" class="sarde-telescope" aria-label="' + escapeHtml(str('trigger')) + '">'
            + '<div class="sarde-telescope-modal">'
            + '<button type="button" class="sarde-telescope-close" id="sarde-telescope-close" aria-label="' + escapeHtml(str('close')) + '">'
            + '<svg aria-hidden="true" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12" stroke-linecap="round"/></svg>'
            + '</button>'
            + '<div class="sarde-telescope-tabs" role="tablist">'
            + '<button type="button" class="sarde-telescope-tab" data-tab="search" role="tab" aria-selected="false" id="sarde-telescope-tab-search" aria-controls="sarde-telescope-search-section">' + escapeHtml(str('tab_search')) + '</button>'
            + '<button type="button" class="sarde-telescope-tab" data-tab="recent" role="tab" aria-selected="false" id="sarde-telescope-tab-recent" aria-controls="sarde-telescope-recent-section">' + escapeHtml(str('tab_recent')) + '</button>'
            + '</div>'
            + '<div class="sarde-telescope-search-header">'
            + '<input id="sarde-telescope-input" class="sarde-telescope-input" type="text" role="combobox" aria-expanded="true" aria-autocomplete="list" aria-haspopup="listbox" aria-controls="sarde-telescope-results" aria-activedescendant="" placeholder="' + escapeHtml(str('placeholder')) + '" autocomplete="off" spellcheck="false">'
            + '</div>'
            + '<div id="sarde-telescope-search-section" class="sarde-telescope-section" role="tabpanel" aria-labelledby="sarde-telescope-tab-search">'
            + '<div id="sarde-telescope-loading" class="sarde-telescope-loading"><div class="sarde-telescope-spinner"></div><span>' + escapeHtml(str('loading')) + '</span></div>'
            + '<ul id="sarde-telescope-results" class="sarde-telescope-results" role="listbox" aria-label="' + escapeHtml(str('results_section')) + '"></ul>'
            + '</div>'
            + '<div id="sarde-telescope-recent-section" class="sarde-telescope-section" role="tabpanel" aria-labelledby="sarde-telescope-tab-recent">'
            + '<ul id="sarde-telescope-recent-results" class="sarde-telescope-results" role="listbox" aria-label="' + escapeHtml(str('tab_recent')) + '"></ul>'
            + '</div>'
            + '<div id="sarde-telescope-live" class="sarde-telescope-sr-only" aria-live="polite" aria-atomic="true"></div>'
            + '<div class="sarde-telescope-footer"><div class="sarde-telescope-shortcuts">'
            + '<span><kbd>↑</kbd><kbd>↓</kbd> ' + escapeHtml(str('hint_navigate')) + '</span>'
            + '<span><kbd>↵</kbd> ' + escapeHtml(str('hint_select')) + '</span>'
            + '<span><kbd>Space</kbd> ' + escapeHtml(str('hint_pin')) + '</span>'
            + '<span><kbd>Esc</kbd> ' + escapeHtml(str('hint_close')) + '</span>'
            + '</div></div>'
            + '</div></dialog>';
    }

    let modalReady = false;
    function ensureModal() {
        if (modalReady && $('sarde-telescope-dialog')) return;
        if (!$('sarde-telescope-dialog')) {
            document.body.insertAdjacentHTML('beforeend', modalHTML());
        }
        bindModalListeners();
        modalReady = true;
    }

    function bindModalListeners() {
        const input = $('sarde-telescope-input');
        if (input) {
            input.addEventListener('input', function (event) {
                if (debounceTimeout) clearTimeout(debounceTimeout);
                debounceTimeout = setTimeout(function () { handleSearchInput(event); }, DEBOUNCE_MS);
            });
            input.addEventListener('keydown', handleSearchKeyDown);
        }
        const closeBtn = $('sarde-telescope-close');
        if (closeBtn) closeBtn.addEventListener('click', close);

        const dialog = $('sarde-telescope-dialog');
        if (dialog) {
            dialog.addEventListener('click', function (event) {
                const tab = event.target.closest('.sarde-telescope-tab');
                if (tab && tab.dataset.tab) {
                    switchTab(tab.dataset.tab);
                    return;
                }
                // Backdrop click: the dialog element itself, outside the panel.
                if (event.target === dialog) close();
            });
            // Native close (Esc via the browser) keeps state consistent.
            dialog.addEventListener('close', onDialogClosed);
        }
    }

    function getActiveResultsContainer() {
        return currentTab === 'recent' ? $('sarde-telescope-recent-results') : $('sarde-telescope-results');
    }

    // ── Search ──
    function getMatchesForPage(page) {
        for (let i = 0; i < searchResultsWithMatches.length; i++) {
            if (searchResultsWithMatches[i].item.path === page.path) {
                return searchResultsWithMatches[i].matches;
            }
        }
        return undefined;
    }

    function highlightMatches(text, matches, key) {
        if (!matches || !text) return escapeHtml(text || '');
        let match = null;
        for (let i = 0; i < matches.length; i++) {
            if (matches[i].key === key) { match = matches[i]; break; }
        }
        if (!match || !match.indices || match.indices.length === 0) return escapeHtml(text);

        let result = '';
        let lastIndex = 0;
        const sorted = match.indices.slice().sort(function (a, b) { return a[0] - b[0]; });
        for (let i = 0; i < sorted.length; i++) {
            const start = sorted[i][0];
            const end = sorted[i][1];
            if (start > lastIndex) result += escapeHtml(text.slice(lastIndex, start));
            if (start >= lastIndex) {
                result += '<mark class="sarde-telescope-highlight">' + escapeHtml(text.slice(start, end + 1)) + '</mark>';
                lastIndex = end + 1;
            }
        }
        result += escapeHtml(text.slice(lastIndex));
        return result;
    }

    function filterPages() {
        const query = searchQuery.trim();
        const lowerQuery = query.toLowerCase();

        if (!query) {
            filteredPages = allPages.slice();
            searchResultsWithMatches = [];
        } else if (fuseInstance) {
            const results = fuseInstance.search(query);
            // Exact title > title prefix > word start > raw Fuse score, so
            // quick-navigation matches beat deep fuzzy matches.
            results.sort(function (a, b) {
                const aTitle = (a.item.title || '').toLowerCase();
                const bTitle = (b.item.title || '').toLowerCase();
                const aExact = aTitle === lowerQuery;
                const bExact = bTitle === lowerQuery;
                if (aExact && !bExact) return -1;
                if (bExact && !aExact) return 1;
                const aPrefix = aTitle.indexOf(lowerQuery) === 0;
                const bPrefix = bTitle.indexOf(lowerQuery) === 0;
                if (aPrefix && !bPrefix) return -1;
                if (bPrefix && !aPrefix) return 1;
                const aWord = aTitle.split(/\s+/).some(function (w) { return w.indexOf(lowerQuery) === 0; });
                const bWord = bTitle.split(/\s+/).some(function (w) { return w.indexOf(lowerQuery) === 0; });
                if (aWord && !bWord) return -1;
                if (bWord && !aWord) return 1;
                return (a.score || 0) - (b.score || 0);
            });
            searchResultsWithMatches = results;
            filteredPages = results.map(function (r) { return r.item; });
        } else {
            filteredPages = allPages.filter(function (page) {
                return (page.title || '').toLowerCase().indexOf(lowerQuery) !== -1
                    || (page.path || '').toLowerCase().indexOf(lowerQuery) !== -1
                    || (page.description || '').toLowerCase().indexOf(lowerQuery) !== -1;
            });
        }

        selectedIndex = 0;
        renderSearchResults();
        updateSelectedResult();

        const count = filteredPages.length;
        if (count === 0) {
            announce(str('no_results'));
        } else if (count > MAX_RESULTS) {
            announce(str('showing_of').replace('{shown}', MAX_RESULTS).replace('{total}', count));
        } else {
            announce(str('results_count').replace('{count}', count));
        }
    }

    function handleSearchInput(event) {
        isInNavigationMode = false;
        searchQuery = event.target.value;

        if (currentTab === 'recent') {
            const lower = searchQuery.toLowerCase();
            filteredPages = validPages(recentPaths).filter(function (page) {
                return (page.title || '').toLowerCase().indexOf(lower) !== -1
                    || (page.path || '').toLowerCase().indexOf(lower) !== -1
                    || (page.description || '').toLowerCase().indexOf(lower) !== -1;
            });
            selectedIndex = 0;
            renderRecentResults();
            updateSelectedResult();
        } else {
            filterPages();
        }
    }

    // ── Keyboard ──
    function getKeyCode(key) {
        if (/^[a-zA-Z]$/.test(key)) return 'Key' + key.toUpperCase();
        if (/^[0-9]$/.test(key)) return 'Digit' + key;
        const special = {
            '/': 'Slash', '\\': 'Backslash', '-': 'Minus', '=': 'Equal',
            '[': 'BracketLeft', ']': 'BracketRight', ';': 'Semicolon',
            "'": 'Quote', ',': 'Comma', '.': 'Period', '`': 'Backquote', ' ': 'Space'
        };
        return special[key] || null;
    }

    function handleGlobalKeyDown(event) {
        const expectedCode = getKeyCode(SHORTCUT_KEY);
        const viaCharacter = event.key === SHORTCUT_KEY;
        const viaCode = expectedCode !== null && event.code === expectedCode;
        // Either Ctrl or Cmd opens the palette. For character matches shift is
        // ignored (some layouts need shift to type the character); for
        // physical-code matches shift must be up.
        const shiftOk = viaCharacter ? true : !event.shiftKey;
        if ((viaCharacter || viaCode) && (event.ctrlKey || event.metaKey) && shiftOk && !event.altKey) {
            event.preventDefault();
            event.stopPropagation();
            open();
            return;
        }
        const dialog = $('sarde-telescope-dialog');
        if (event.key === 'Escape' && dialog && dialog.open) {
            event.preventDefault();
            close();
        }
    }

    function handleSearchKeyDown(event) {
        const dialog = $('sarde-telescope-dialog');
        if (!dialog || !dialog.open) return;

        switch (event.key) {
            case 'Escape':
                event.preventDefault();
                close();
                break;
            case 'ArrowDown':
                event.preventDefault();
                isInNavigationMode = true;
                navigateResults(1);
                break;
            case 'ArrowUp':
                event.preventDefault();
                isInNavigationMode = true;
                navigateResults(-1);
                break;
            case 'Enter':
                event.preventDefault();
                selectCurrentItem();
                break;
            case ' ': {
                // Space pins when it cannot be intended as typed text.
                const input = $('sarde-telescope-input');
                if (input && (input.value === '' || input.selectionStart === 0 || isInNavigationMode)) {
                    event.preventDefault();
                    togglePinForSelectedItem();
                }
                break;
            }
            default:
                break;
        }
    }

    function navigateResults(direction) {
        const container = getActiveResultsContainer();
        if (!container) return;
        const totalItems = container.querySelectorAll('.sarde-telescope-result').length;
        if (totalItems === 0) return;
        selectedIndex = (selectedIndex + direction + totalItems) % totalItems;
        updateSelectedResult();
    }

    function updateSelectedResult() {
        const container = getActiveResultsContainer();
        if (!container) return;
        const items = container.querySelectorAll('.sarde-telescope-result');
        items.forEach(function (item) {
            item.classList.remove('sarde-telescope-result-selected');
            item.setAttribute('aria-selected', 'false');
        });
        const selected = container.querySelector('[data-index="' + selectedIndex + '"]');
        if (!selected) return;
        selected.classList.add('sarde-telescope-result-selected');
        selected.setAttribute('aria-selected', 'true');
        const input = $('sarde-telescope-input');
        if (input) input.setAttribute('aria-activedescendant', selected.id);
        selected.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }

    function selectCurrentItem() {
        const selected = document.querySelector('.sarde-telescope-result-selected');
        if (selected && selected.hasAttribute('data-path')) {
            navigateToPage(selected.getAttribute('data-path'));
        }
    }

    function togglePinForSelectedItem() {
        const selected = document.querySelector('.sarde-telescope-result-selected');
        if (selected && selected.hasAttribute('data-path')) {
            togglePin(selected.getAttribute('data-path'));
        }
    }

    function navigateToPage(path) {
        recordVisit(path);
        close();
        window.location.href = path;
    }

    // ── Rendering ──
    function createSectionHeader(text, ariaClearLabel, onClear) {
        const header = document.createElement('li');
        header.className = 'sarde-telescope-section-separator';
        header.setAttribute('role', 'presentation');
        const title = document.createElement('span');
        title.textContent = text;
        header.appendChild(title);
        if (onClear) {
            const clearBtn = document.createElement('button');
            clearBtn.type = 'button';
            clearBtn.className = 'sarde-telescope-clear-btn';
            clearBtn.textContent = str('clear');
            clearBtn.setAttribute('aria-label', ariaClearLabel);
            clearBtn.addEventListener('click', function (e) {
                e.stopPropagation();
                onClear();
            });
            header.appendChild(clearBtn);
        }
        return header;
    }

    function createNoResults(message) {
        const li = document.createElement('li');
        li.className = 'sarde-telescope-no-results';
        li.setAttribute('role', 'presentation');
        li.textContent = message;
        return li;
    }

    function createResultItem(page, index, containerId) {
        const li = document.createElement('li');
        li.id = containerId + '-option-' + index;
        li.className = 'sarde-telescope-result' + (index === selectedIndex ? ' sarde-telescope-result-selected' : '');
        li.setAttribute('role', 'option');
        li.setAttribute('aria-selected', index === selectedIndex ? 'true' : 'false');
        li.setAttribute('data-index', String(index));
        li.setAttribute('data-path', page.path);

        const pinned = isPinned(page.path);
        const pinButton = document.createElement('button');
        pinButton.type = 'button';
        pinButton.className = 'sarde-telescope-pin-btn' + (pinned ? ' sarde-telescope-pin-btn-pinned' : '');
        pinButton.innerHTML = '<svg aria-hidden="true" viewBox="0 0 24 24" width="14" height="14"><path d="M17 3H7c-1.1 0-1.99.9-1.99 2L5 21l7-3 7 3V5c0-1.1-.9-2-2-2z"></path></svg>';
        pinButton.title = pinned ? str('unpin') : str('pin');
        pinButton.setAttribute('aria-label', pinned ? str('unpin') : str('pin'));
        pinButton.addEventListener('click', function (event) {
            event.stopPropagation();
            event.preventDefault();
            togglePin(page.path);
        });

        const contentRow = document.createElement('div');
        contentRow.className = 'sarde-telescope-result-content';

        const matches = searchQuery.trim() ? getMatchesForPage(page) : undefined;

        const titleDiv = document.createElement('div');
        titleDiv.className = 'sarde-telescope-result-title';
        titleDiv.innerHTML = highlightMatches(page.title || '', matches, 'title');
        contentRow.appendChild(titleDiv);

        if (page.description) {
            const descDiv = document.createElement('div');
            descDiv.className = 'sarde-telescope-result-description';
            descDiv.innerHTML = highlightMatches(page.description, matches, 'description');
            contentRow.appendChild(descDiv);
        }

        if (page.collection) {
            const collectionSpan = document.createElement('span');
            collectionSpan.className = 'sarde-telescope-result-collection';
            collectionSpan.textContent = page.collection;
            contentRow.appendChild(collectionSpan);
        }

        if (page.tags && page.tags.length > 0) {
            const tagsDiv = document.createElement('div');
            tagsDiv.className = 'sarde-telescope-result-tags';
            page.tags.forEach(function (tag) {
                const span = document.createElement('span');
                span.className = 'sarde-telescope-tag';
                span.textContent = tag;
                tagsDiv.appendChild(span);
            });
            contentRow.appendChild(tagsDiv);
        }

        li.appendChild(contentRow);
        li.appendChild(pinButton);

        li.addEventListener('click', function () {
            navigateToPage(page.path);
        });
        li.addEventListener('mouseenter', function () {
            // Ignore hover until the mouse actually moves, so a modal opening
            // under the cursor cannot steal the selection.
            if (!hasMouseMovedSinceOpen) return;
            selectedIndex = index;
            updateSelectedResult();
        });

        return li;
    }

    function renderSearchResults() {
        const container = $('sarde-telescope-results');
        if (!container) return;
        container.innerHTML = '';

        const validPinned = validPages(pinnedPaths);
        const validRecent = validPages(recentPaths);

        let index = 0;

        if (validPinned.length > 0) {
            container.appendChild(createSectionHeader(str('pinned_section'), str('clear_pinned'), clearPinned));
            validPinned.forEach(function (page) {
                container.appendChild(createResultItem(page, index, 'sarde-telescope-results'));
                index++;
            });
        }

        const nonPinnedRecent = validRecent.filter(function (p) { return !isPinned(p.path); }).slice(0, MAX_RECENT);
        if (nonPinnedRecent.length > 0) {
            container.appendChild(createSectionHeader(str('recent_section'), str('clear_recent'), clearRecent));
            nonPinnedRecent.forEach(function (page) {
                container.appendChild(createResultItem(page, index, 'sarde-telescope-results'));
                index++;
            });
        }

        // Keep search results free of rows already shown above.
        const shownPaths = {};
        validPinned.forEach(function (p) { shownPaths[p.path] = true; });
        nonPinnedRecent.forEach(function (p) { shownPaths[p.path] = true; });
        const results = filteredPages.filter(function (p) { return !shownPaths[p.path]; });

        if (index > 0 && results.length > 0) {
            container.appendChild(createSectionHeader(str('results_section')));
        }

        if (results.length === 0 && index === 0) {
            container.appendChild(createNoResults(str('no_results')));
            return;
        }

        const display = results.slice(0, MAX_RESULTS);
        display.forEach(function (page) {
            container.appendChild(createResultItem(page, index, 'sarde-telescope-results'));
            index++;
        });

        if (results.length > MAX_RESULTS) {
            const indicator = document.createElement('li');
            indicator.className = 'sarde-telescope-more-results';
            indicator.setAttribute('role', 'presentation');
            indicator.textContent = str('showing_of').replace('{shown}', MAX_RESULTS).replace('{total}', results.length);
            container.appendChild(indicator);
        }
    }

    function renderRecentResults() {
        const container = $('sarde-telescope-recent-results');
        if (!container) return;
        container.innerHTML = '';

        const pages = searchQuery.trim() ? filteredPages : validPages(recentPaths).slice(0, MAX_RECENT);
        if (pages.length === 0) {
            container.appendChild(createNoResults(searchQuery.trim() ? str('no_results') : str('no_recent')));
            return;
        }
        pages.forEach(function (page, index) {
            container.appendChild(createResultItem(page, index, 'sarde-telescope-recent-results'));
        });
    }

    function switchTab(tabName) {
        currentTab = tabName === 'recent' ? 'recent' : 'search';
        selectedIndex = 0;
        searchQuery = '';
        const input = $('sarde-telescope-input');
        if (input) {
            input.value = '';
            input.placeholder = currentTab === 'recent' ? str('filter_recent') : str('placeholder');
        }
        filteredPages = allPages.slice();

        document.querySelectorAll('.sarde-telescope-tab').forEach(function (tab) {
            const active = tab.dataset.tab === currentTab;
            tab.classList.toggle('sarde-telescope-tab-active', active);
            tab.setAttribute('aria-selected', active ? 'true' : 'false');
        });
        document.querySelectorAll('.sarde-telescope-section').forEach(function (section) {
            section.classList.toggle('sarde-telescope-section-active', section.id === 'sarde-telescope-' + currentTab + '-section');
        });
        if (input) input.setAttribute('aria-controls', 'sarde-telescope-' + (currentTab === 'recent' ? 'recent-results' : 'results'));

        if (currentTab === 'recent') {
            renderRecentResults();
        } else {
            renderSearchResults();
        }
        updateSelectedResult();
    }

    // ── Open / close ──
    function open() {
        ensureModal();
        const dialog = $('sarde-telescope-dialog');
        if (!dialog || dialog.open) return;

        searchQuery = '';
        selectedIndex = 0;
        filteredPages = allPages.slice();
        searchResultsWithMatches = [];
        isInNavigationMode = false;

        dialog.showModal();

        dialog.classList.add('sarde-telescope-pointer-disabled');
        hasMouseMovedSinceOpen = false;
        dialog.addEventListener('mousemove', function () {
            hasMouseMovedSinceOpen = true;
            dialog.classList.remove('sarde-telescope-pointer-disabled');
        }, { once: true });

        updateLoadingState();
        switchTab(DEFAULT_TAB);

        ensurePages().then(function () {
            if (!dialog.open) return;
            filteredPages = allPages.slice();
            if (currentTab === 'recent') {
                renderRecentResults();
            } else {
                renderSearchResults();
            }
            updateSelectedResult();
        });

        requestAnimationFrame(function () {
            const input = $('sarde-telescope-input');
            if (input) {
                input.value = '';
                input.focus();
            }
        });

        announce(str('dialog_opened'));
    }

    function onDialogClosed() {
        if (debounceTimeout) {
            clearTimeout(debounceTimeout);
            debounceTimeout = null;
        }
        const input = $('sarde-telescope-input');
        if (input) input.setAttribute('aria-activedescendant', '');
    }

    function close() {
        const dialog = $('sarde-telescope-dialog');
        if (!dialog || !dialog.open) return;
        dialog.close();
        onDialogClosed();
    }

    // ── Trigger button ──
    function createTrigger() {
        const actions = document.querySelector('.sarde-header-actions');
        if (!actions) return;

        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'sarde-telescope-trigger';
        button.setAttribute('aria-label', str('trigger'));
        button.title = str('trigger');
        button.setAttribute('aria-keyshortcuts', 'Control+' + SHORTCUT_KEY);
        button.innerHTML = '<svg aria-hidden="true" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">'
            + '<path d="m10.065 12.493-6.18 1.318a.934.934 0 0 1-1.108-.702l-.537-2.15a1.07 1.07 0 0 1 .691-1.265l13.504-4.44"/>'
            + '<path d="m13.56 11.747 4.332-.924"/>'
            + '<path d="m16 21-3.105-6.21"/>'
            + '<path d="M16.485 5.94a2 2 0 0 1 1.455-2.425l1.09-.272a1 1 0 0 1 1.212.727l1.515 6.06a1 1 0 0 1-.727 1.213l-1.09.272a2 2 0 0 1-2.425-1.455z"/>'
            + '<path d="m6.158 8.633 1.114 4.456"/>'
            + '<path d="m8 21 3.105-6.21"/>'
            + '</svg>'
            + '<kbd class="sarde-telescope-shortcut"><kbd class="sarde-telescope-shortcut-mod">Ctrl</kbd><kbd>' + escapeHtml(SHORTCUT_KEY) + '</kbd></kbd>';

        const isMac = (navigator.userAgentData && navigator.userAgentData.platform === 'macOS')
            || /Mac|iPhone|iPod|iPad/i.test(navigator.userAgent);
        if (isMac) {
            const mod = button.querySelector('.sarde-telescope-shortcut-mod');
            if (mod) mod.textContent = '⌘';
            button.setAttribute('aria-keyshortcuts', 'Meta+' + SHORTCUT_KEY);
        }

        button.addEventListener('click', open);

        // Sit next to the built-in search trigger when present, otherwise
        // lead the header actions.
        const searchTrigger = actions.querySelector('.sarde-search-trigger');
        if (searchTrigger) {
            searchTrigger.insertAdjacentElement('afterend', button);
        } else {
            actions.insertAdjacentElement('afterbegin', button);
        }
    }

    // ── Init ──
    function init() {
        recordVisit(window.location.pathname);
        createTrigger();
        document.addEventListener('keydown', handleGlobalKeyDown, true);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
