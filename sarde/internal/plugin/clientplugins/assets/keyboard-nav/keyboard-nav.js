const cfg = (window.__SARDE__ && window.__SARDE__.pluginConfig && window.__SARDE__.pluginConfig["keyboard-nav"]) || {};

const config = {
    showHint: cfg.show_hint !== false,
};

// -- Navigation --

function isTyping() {
    const el = document.activeElement;
    if (!el) return false;
    const tag = el.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true;
    if (el.isContentEditable) return true;
    if (el.closest('.cm-editor')) return true;
    return false;
}

function getPrevUrl() {
    const link = document.querySelector('nav.pagination .pagination-prev');
    return link ? link.href : null;
}

function getNextUrl() {
    const link = document.querySelector('nav.pagination .pagination-next');
    return link ? link.href : null;
}

function navigateTo(linkClass) {
    const link = document.querySelector('nav.pagination .' + linkClass);
    if (!link) return;
    link.classList.add('sarde-kbd-nav-active');
    setTimeout(function () { link.click(); }, 80);
}

function onKeyDown(e) {
    if (e.altKey || e.ctrlKey || e.metaKey || e.shiftKey) return;
    if (isTyping()) return;
    if (e.key === 'ArrowLeft') {
        if (getPrevUrl()) { e.preventDefault(); navigateTo('pagination-prev'); }
    } else if (e.key === 'ArrowRight') {
        if (getNextUrl()) { e.preventDefault(); navigateTo('pagination-next'); }
    }
}

// -- Hint Toast --

function showHint() {
    if (!config.showHint) return;
    const old = document.querySelector('.sarde-kbd-nav-hint');
    if (old) old.remove();
    const hasNav = document.querySelector('nav.pagination .pagination-prev, nav.pagination .pagination-next');
    if (!hasNav) return;
    const hint = document.createElement('div');
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
showHint();
