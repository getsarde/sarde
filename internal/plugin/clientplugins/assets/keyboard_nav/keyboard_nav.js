const cfg = (window.__SARDE__ && window.__SARDE__.pluginConfig && window.__SARDE__.pluginConfig["keyboard_nav"]) || {};

// Allowlisted because the value comes from user YAML and lands in an attribute
// that CSS selectors match on.
var SIDE_NAV_SIZES = { small: true, medium: true, large: true };
var AUTO_HIDE_MODES = { compact: true, always: true, off: true };

const config = {
    showHint: cfg.show_hint !== false,
    showSideNav: cfg.show_side_nav !== false,
    showTooltip: cfg.show_tooltip !== false,
    showCompactNav: cfg.show_compact_nav !== false,
    sideNavSize: Object.prototype.hasOwnProperty.call(SIDE_NAV_SIZES, cfg.side_nav_size) ? cfg.side_nav_size : 'medium',
    // A YAML `auto_hide: false` reads naturally, so accept it as "off".
    autoHide: cfg.auto_hide === false ? 'off'
        : Object.prototype.hasOwnProperty.call(AUTO_HIDE_MODES, cfg.auto_hide) ? cfg.auto_hide : 'compact',
    hideDelay: typeof cfg.hide_delay === 'number' && cfg.hide_delay >= 0 ? cfg.hide_delay : 2000,
    scrollThreshold: typeof cfg.scroll_threshold === 'number' && cfg.scroll_threshold >= 0 ? cfg.scroll_threshold : 100,
};

var CHEVRON_LEFT = '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>';
var CHEVRON_RIGHT = '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>';

// -- Navigation --

function isTyping() {
    var el = document.activeElement;
    if (!el) return false;
    var tag = el.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true;
    if (el.isContentEditable) return true;
    if (el.closest('.cm-editor')) return true;
    return false;
}

function getPrevUrl() {
    var link = document.querySelector('.sarde-side-nav-prev') ||
               document.querySelector('.sarde-pagination-prev');
    return link ? link.href : null;
}

function getNextUrl() {
    var link = document.querySelector('.sarde-side-nav-next') ||
               document.querySelector('.sarde-pagination-next');
    return link ? link.href : null;
}

function navigateTo(cls) {
    var link = document.querySelector('.' + cls);
    if (!link) return;
    link.classList.add('sarde-kbd-nav-active');
    setTimeout(function () { link.click(); }, 80);
}

function onKeyDown(e) {
    if (e.altKey || e.ctrlKey || e.metaKey || e.shiftKey) return;
    if (isTyping()) return;
    if (e.key === 'ArrowLeft') {
        if (getPrevUrl()) { e.preventDefault(); navigateTo('sarde-side-nav-prev'); }
        else if (document.querySelector('.sarde-pagination-prev')) { e.preventDefault(); navigateTo('sarde-pagination-prev'); }
    } else if (e.key === 'ArrowRight') {
        if (getNextUrl()) { e.preventDefault(); navigateTo('sarde-side-nav-next'); }
        else if (document.querySelector('.sarde-pagination-next')) { e.preventDefault(); navigateTo('sarde-pagination-next'); }
    }
}

// -- Side Navigation Arrows --

// The stylesheet reserves page margin for the arrows by default, so medium
// needs no attribute and the common case paints without a reflow. Only the
// opt-out and the off-default sizes stamp the root.
function applySideNavSize() {
    var root = document.documentElement;
    if (!config.showSideNav) {
        root.setAttribute('data-sarde-side-nav', 'off');
        return;
    }
    if (config.sideNavSize !== 'medium') {
        root.setAttribute('data-sarde-side-nav', config.sideNavSize);
    }
}

// Builds one side arrow from its server-rendered pagination link. The pager's
// .sarde-pagination-dir text is already localized (t "nav.previous"/"nav.next"),
// so reusing it here localizes the aria-label for free; the English fallback
// only covers a pager markup change.
function buildArrow(srcLink, cls, rel, fallbackDir, keyGlyph, chevron) {
    var titleEl = srcLink.querySelector('.sarde-pagination-title');
    var title = titleEl ? titleEl.textContent.trim() : '';
    var dirEl = srcLink.querySelector('.sarde-pagination-dir');
    var dir = (dirEl && dirEl.textContent.trim()) || fallbackDir;

    var a = document.createElement('a');
    a.href = srcLink.href;
    a.className = cls;
    a.setAttribute('rel', rel);
    a.setAttribute('aria-label', title ? dir + ': ' + title : dir);
    a.innerHTML = chevron;

    if (config.showTooltip) {
        // The visual tooltip duplicates the aria-label, so hide it from AT.
        var tip = document.createElement('span');
        tip.className = 'sarde-side-nav-tooltip';
        tip.setAttribute('aria-hidden', 'true');
        var tipTitle = document.createElement('span');
        tipTitle.className = 'sarde-side-nav-tooltip-title';
        tipTitle.textContent = title || dir;
        var tipKey = document.createElement('kbd');
        tipKey.className = 'sarde-kbd-nav-hint-key';
        tipKey.textContent = keyGlyph;
        tip.appendChild(tipTitle);
        tip.appendChild(tipKey);
        a.appendChild(tip);
    } else if (title) {
        // Custom tooltip disabled: fall back to the native browser tooltip.
        a.title = title;
    }

    return a;
}

function createSideNav() {
    if (!config.showSideNav) return;

    var prevLink = document.querySelector('.sarde-pagination-prev');
    var nextLink = document.querySelector('.sarde-pagination-next');
    if (!prevLink && !nextLink) return;

    var nav = document.createElement('nav');
    // The compact class lets the arrows restyle into small floating buttons
    // below 1280px; without it the stylesheet hides them there entirely.
    nav.className = config.showCompactNav ? 'sarde-side-nav sarde-side-nav-compact' : 'sarde-side-nav';
    nav.setAttribute('aria-label', 'Page navigation');

    if (prevLink) {
        nav.appendChild(buildArrow(prevLink, 'sarde-side-nav-prev', 'prev', 'Previous page', '←', CHEVRON_LEFT));
    }
    if (nextLink) {
        nav.appendChild(buildArrow(nextLink, 'sarde-side-nav-next', 'next', 'Next page', '→', CHEVRON_RIGHT));
    }

    document.body.appendChild(nav);
    setupAutoHide(nav);
}

// -- Auto-Hide --

// Fades the side navigation out after a spell of inactivity and brings it back
// once the user has scrolled scrollThreshold pixels in either direction.
// Ported from the scrollBehavior feature of starlight-custom-navigation.
function setupAutoHide(nav) {
    if (config.autoHide === 'off') return;

    // Compact mode only auto-hides below the strip breakpoint, where the
    // buttons float over content; the desktop strips live in reserved margin.
    var mq = window.matchMedia('(max-width: 1279px)');
    var timer = null;
    var lastY = window.scrollY;
    var traveled = 0;
    var ticking = false;

    function active() {
        return config.autoHide === 'always' || mq.matches;
    }

    function hide() {
        if (!active()) return;
        // A page too short to scroll could never reveal the buttons again.
        if (document.documentElement.scrollHeight - window.innerHeight <= config.scrollThreshold) return;
        // Don't pull them out from under the pointer or keyboard focus.
        if (nav.matches(':hover') || nav.contains(document.activeElement)) {
            arm();
            return;
        }
        nav.classList.add('sarde-side-nav-hidden');
        traveled = 0;
    }

    function arm() {
        clearTimeout(timer);
        timer = setTimeout(hide, config.hideDelay);
    }

    function show() {
        nav.classList.remove('sarde-side-nav-hidden');
        arm();
    }

    window.addEventListener('scroll', function () {
        if (ticking) return;
        ticking = true;
        requestAnimationFrame(function () {
            ticking = false;
            var y = window.scrollY;
            var delta = Math.abs(y - lastY);
            lastY = y;
            if (!active()) return;
            if (nav.classList.contains('sarde-side-nav-hidden')) {
                traveled += delta;
                if (traveled >= config.scrollThreshold) show();
            } else {
                arm();
            }
        });
    }, { passive: true });

    // Tabbing to a hidden arrow brings the pair back.
    nav.addEventListener('focusin', show);
    // Leaving after a hover restarts the countdown the hover was blocking.
    nav.addEventListener('mouseleave', arm);

    // Crossing up into the strip band while hidden must restore the strips.
    if (mq.addEventListener) {
        mq.addEventListener('change', function (e) {
            if (config.autoHide === 'compact' && !e.matches) show();
        });
    }

    arm();
}

// -- Hint Toast --

function showHint() {
    if (!config.showHint) return;
    var old = document.querySelector('.sarde-kbd-nav-hint');
    if (old) old.remove();
    var hasNav = document.querySelector('.sarde-pagination-prev, .sarde-pagination-next');
    if (!hasNav) return;
    var hint = document.createElement('div');
    hint.className = 'sarde-kbd-nav-hint';
    hint.setAttribute('role', 'status');
    hint.innerHTML =
        '<span class="sarde-kbd-nav-hint-keys">' +
            '<kbd class="sarde-kbd-nav-hint-key">&larr;</kbd>' +
            '<kbd class="sarde-kbd-nav-hint-key">&rarr;</kbd>' +
        '</span>' +
        '<span>Navigate between pages</span>';
    document.body.appendChild(hint);
    requestAnimationFrame(function () {
        requestAnimationFrame(function () { hint.classList.add('visible'); });
    });
}

// -- Init --

document.addEventListener('keydown', onKeyDown);
applySideNavSize();
createSideNav();
showHint();
