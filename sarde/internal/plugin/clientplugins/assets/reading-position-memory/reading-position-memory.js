// Reading Position Memory Plugin
// Saves scroll position per page to localStorage. Shows a toast to restore on return.

const cfg = (window.__SARDE__ && window.__SARDE__.pluginConfig && window.__SARDE__.pluginConfig["reading-position-memory"]) || {};

const config = {
    toastDuration: cfg.toast_duration || 8,
    toastPosition: cfg.toast_position || 'bottom-center',
    scrollThreshold: cfg.scroll_threshold || 5,
    lessonPagesOnly: cfg.lesson_pages_only !== false,
};

const STORAGE_PREFIX = 'rpm:';
const SAVE_DEBOUNCE = 500;

let saveTimer = null;
let activeToast = null;
let toastTimeout = null;

function getPageKey() {
    return STORAGE_PREFIX + window.location.pathname;
}

function isContentPage() {
    return !!document.querySelector('article.sarde-markdown-content');
}

function getScrollPercent() {
    const scrollTop = window.scrollY || document.documentElement.scrollTop;
    const docHeight = document.documentElement.scrollHeight - document.documentElement.clientHeight;
    if (docHeight <= 0) return 0;
    return (scrollTop / docHeight) * 100;
}

function savePosition() {
    if (saveTimer) clearTimeout(saveTimer);

    saveTimer = setTimeout(() => {
        const percent = getScrollPercent();

        if (percent < config.scrollThreshold) {
            localStorage.removeItem(getPageKey());
            return;
        }

        const data = {
            scrollY: window.scrollY,
            percent: Math.round(percent),
            timestamp: Date.now(),
        };

        try {
            localStorage.setItem(getPageKey(), JSON.stringify(data));
        } catch (_e) {
            // Storage full
        }
    }, SAVE_DEBOUNCE);
}

function dismissToast() {
    if (toastTimeout) {
        clearTimeout(toastTimeout);
        toastTimeout = null;
    }
    if (activeToast) {
        activeToast.classList.remove('is-visible');
        const toast = activeToast;
        setTimeout(() => {
            if (toast.parentNode) toast.parentNode.removeChild(toast);
        }, 300);
        activeToast = null;
    }
}

function showToast(scrollY, percent) {
    dismissToast();

    const toast = document.createElement('div');
    toast.className = 'sarde-rpm-toast sarde-rpm-toast-' + config.toastPosition;
    toast.setAttribute('role', 'status');

    const message = document.createElement('span');
    message.className = 'sarde-rpm-toast-message';
    message.textContent = 'Continue where you left off? (' + percent + '%)';

    const actions = document.createElement('span');
    actions.className = 'sarde-rpm-toast-actions';

    const jumpBtn = document.createElement('button');
    jumpBtn.className = 'sarde-rpm-toast-btn sarde-rpm-toast-btn-jump';
    jumpBtn.textContent = 'Jump';
    jumpBtn.setAttribute('aria-label', 'Jump to saved position');
    jumpBtn.addEventListener('click', () => {
        window.scrollTo({ top: scrollY, behavior: 'smooth' });
        dismissToast();
    });

    const dismissBtn = document.createElement('button');
    dismissBtn.className = 'sarde-rpm-toast-btn sarde-rpm-toast-btn-dismiss';
    dismissBtn.textContent = '×';
    dismissBtn.setAttribute('aria-label', 'Dismiss');
    dismissBtn.addEventListener('click', dismissToast);

    actions.appendChild(jumpBtn);
    actions.appendChild(dismissBtn);
    toast.appendChild(message);
    toast.appendChild(actions);

    document.body.appendChild(toast);
    activeToast = toast;

    requestAnimationFrame(() => {
        requestAnimationFrame(() => {
            toast.classList.add('is-visible');
        });
    });

    toastTimeout = setTimeout(dismissToast, config.toastDuration * 1000);
}

function checkSavedPosition() {
    const raw = localStorage.getItem(getPageKey());
    if (!raw) return;

    let data;
    try {
        data = JSON.parse(raw);
    } catch (_e) {
        localStorage.removeItem(getPageKey());
        return;
    }

    if (Date.now() - data.timestamp < 10000) return;

    if (Date.now() - data.timestamp > 30 * 24 * 60 * 60 * 1000) {
        localStorage.removeItem(getPageKey());
        return;
    }

    if (data.scrollY > 100 && data.percent >= config.scrollThreshold) {
        setTimeout(() => {
            showToast(data.scrollY, data.percent);
        }, 400);
    }
}

function pruneOldEntries() {
    const maxAge = 30 * 24 * 60 * 60 * 1000;
    const now = Date.now();

    for (let i = localStorage.length - 1; i >= 0; i--) {
        const key = localStorage.key(i);
        if (!key || key.indexOf(STORAGE_PREFIX) !== 0) continue;

        try {
            const data = JSON.parse(localStorage.getItem(key) || '');
            if (now - data.timestamp > maxAge) {
                localStorage.removeItem(key);
            }
        } catch (_e) {
            localStorage.removeItem(key);
        }
    }
}

function init() {
    if (config.lessonPagesOnly && !isContentPage()) return;

    checkSavedPosition();
    window.addEventListener('scroll', savePosition, { passive: true });

    if (Math.random() < 0.1) {
        pruneOldEntries();
    }
}

init();
