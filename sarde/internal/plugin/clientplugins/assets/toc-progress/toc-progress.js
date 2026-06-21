const cfg = (window.__pluginConfig && window.__pluginConfig["toc-progress"]) || {};

const config = {
    highlightAll: cfg.highlight_all_visible !== false,
};

let observer = null;

let visibleIds = new Set();

let headings = [];

let tocNav = null;

let allLinks = [];

let rafId = null;

const HEADER_OFFSET = 100;

// ── Update ─────────────────────────────────────────────────────

function update() {
    if (!tocNav || allLinks.length === 0) return;

    // Find the single active link (first visible heading, or fallback to .active)
    var activeIdx = -1;

    for (var i = 0; i < allLinks.length; i++) {
        var href = allLinks[i].getAttribute('href') || '';
        var id = href.charAt(0) === '#' ? href.slice(1) : '';
        if (id && visibleIds.has(id)) {
            activeIdx = i;
            break;
        }
    }

    if (activeIdx === -1) {
        for (var k = 0; k < allLinks.length; k++) {
            if (allLinks[k].classList.contains('active')) {
                activeIdx = k;
                break;
            }
        }
    }

    for (var j = 0; j < allLinks.length; j++) {
        if (j === activeIdx) {
            allLinks[j].classList.add('sarde-toc-progress-visible');
        } else {
            allLinks[j].classList.remove('sarde-toc-progress-visible');
        }
    }
}

// ── Observer ───────────────────────────────────────────────────

function setupObserver() {
    if (observer) { observer.disconnect(); observer = null; }
    visibleIds = new Set();
    headings = [];

    allLinks.forEach(function (link) {
        var href = link.getAttribute('href') || '';
        var id = href.charAt(0) === '#' ? href.slice(1) : '';
        if (!id) return;
        var el = document.getElementById(id);
        if (!el) return;
        var heading = (el.closest('h1,h2,h3,h4,h5,h6') || el);
        headings.push({ id: id, el: heading });
    });

    if (headings.length === 0) return;

    observer = new IntersectionObserver(function (entries) {
        entries.forEach(function (entry) {
            for (var i = 0; i < headings.length; i++) {
                if (headings[i].el === entry.target) {
                    if (entry.isIntersecting) {
                        visibleIds.add(headings[i].id);
                    } else {
                        visibleIds.delete(headings[i].id);
                    }
                    break;
                }
            }
        });

        if (visibleIds.size === 0) {
            var scrollY = window.scrollY + HEADER_OFFSET;
            var best = null;
            var bestDist = Infinity;
            headings.forEach(function (h) {
                var t = h.el.getBoundingClientRect().top + window.scrollY;
                if (t <= scrollY && (scrollY - t) < bestDist) {
                    bestDist = scrollY - t;
                    best = h.id;
                }
            });
            if (best) visibleIds.add(best);
        }

        update();
    }, {
        root: null,
        rootMargin: '-' + HEADER_OFFSET + 'px 0px -70% 0px',
        threshold: 0,
    });

    headings.forEach(function (h) { observer.observe(h.el); });
}

// ── Scroll ─────────────────────────────────────────────────────

function onScroll() {
    if (rafId) return;
    rafId = requestAnimationFrame(function () {
        rafId = null;
        var atBottom = (window.innerHeight + window.scrollY) >=
            (document.documentElement.scrollHeight - 50);
        if (atBottom && headings.length > 0) {
            visibleIds.add(headings[headings.length - 1].id);
        }
        update();
    });
}

// ── Lifecycle ──────────────────────────────────────────────────

function cleanup() {
    if (observer) { observer.disconnect(); observer = null; }
    if (rafId) { cancelAnimationFrame(rafId); rafId = null; }
    visibleIds = new Set();
    headings = [];
    for (var i = 0; i < allLinks.length; i++) {
        allLinks[i].classList.remove('sarde-toc-progress-visible');
    }
    window.removeEventListener('scroll', onScroll);
    if (tocNav) tocNav.classList.remove('sarde-toc-progress-active');
    tocNav = null;
    allLinks = [];
}

function init() {
    cleanup();
    tocNav = document.querySelector('nav.sarde-toc-nav');
    if (!tocNav) return;

    tocNav.classList.add('sarde-toc-progress-active');
    allLinks = Array.from(tocNav.querySelectorAll('a'));
    if (allLinks.length === 0) return;

    setupObserver();
    window.addEventListener('scroll', onScroll, { passive: true });
    setTimeout(update, 100);
}

init();
