const cfg = (window.__pluginConfig && window.__pluginConfig["scroll-to-top"]) || {};

const THRESHOLD = cfg.threshold || 30;
const SHOW_TOOLTIP = cfg.showTooltip || false;
const SHOW_PROGRESS_RING = cfg.showProgressRing || false;
const SMOOTH_SCROLL = cfg.smoothScroll !== false;
const BORDER_RADIUS = (cfg.borderRadius != null ? cfg.borderRadius : 15) + '%';
const PROGRESS_RING_COLOR = cfg.progressRingColor || null;
const SVG_PATH = 'M18 15l-6-6-6 6';
let isKeyboard = false;

// Create button
const btn = document.createElement('button');
btn.className = 'scroll-to-top';
btn.style.borderRadius = BORDER_RADIUS;
if (PROGRESS_RING_COLOR) btn.style.setProperty('--scroll-progress-color', PROGRESS_RING_COLOR);
btn.setAttribute('aria-label', 'Scroll to top');
btn.setAttribute('role', 'button');
btn.setAttribute('tabindex', '0');

// Build inner HTML (hardcoded SVG -- no user data)
const progressRingHtml = SHOW_PROGRESS_RING
    ? '<svg class="scroll-progress-ring" viewBox="0 0 47 47">' +
      '<circle cx="23.5" cy="23.5" r="22" fill="none" stroke-width="3" class="scroll-progress-track"/>' +
      '<circle cx="23.5" cy="23.5" r="22" fill="none" stroke-width="3" stroke-linecap="round" class="scroll-progress-circle" style="transform:rotate(-90deg);transform-origin:center;"/>' +
      '</svg>'
    : '';

btn.innerHTML = progressRingHtml +
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="' + SVG_PATH + '"/></svg>';

document.body.appendChild(btn);

// Tooltip
let tooltip = null;
if (SHOW_TOOLTIP) {
    tooltip = document.createElement('div');
    tooltip.className = 'scroll-to-top-tooltip';
    tooltip.id = 'scroll-to-top-tooltip';
    tooltip.textContent = 'Scroll to top';
    const arrow = document.createElement('div');
    arrow.className = 'scroll-to-top-tooltip-arrow';
    tooltip.appendChild(arrow);
    btn.appendChild(tooltip);
    btn.setAttribute('aria-describedby', 'scroll-to-top-tooltip');
}

function showTooltip() {
    if (tooltip) tooltip.classList.add('visible');
}

function hideTooltip() {
    if (tooltip) tooltip.classList.remove('visible');
}

function doScrollToTop() {
    hideTooltip();
    window.scrollTo({ top: 0, behavior: SMOOTH_SCROLL ? 'smooth' : 'auto' });
    btn.classList.remove('active');
}

// Throttle for scroll performance (~60fps)
function throttle(fn, limit) {
    let inThrottle;
    return function () {
        if (!inThrottle) {
            fn();
            inThrottle = true;
            setTimeout(function () { inThrottle = false; }, limit);
        }
    };
}

// Show/hide based on scroll percentage
function onScroll() {
    const scrollPos = window.scrollY;
    const viewportHeight = window.innerHeight;
    const pageHeight = document.documentElement.scrollHeight;
    const scrollPct = scrollPos / (pageHeight - viewportHeight);

    // Update progress ring
    if (SHOW_PROGRESS_RING) {
        const circle = btn.querySelector('.scroll-progress-circle');
        if (circle) {
            const progress = Math.min(Math.max(scrollPct * 100, 0), 100);
            const circumference = 138.23;
            circle.style.strokeDashoffset = (circumference - (progress / 100) * circumference).toString();
        }
    }

    const thresholdVal = THRESHOLD >= 10 && THRESHOLD <= 99 ? THRESHOLD : 30;
    if (scrollPct > thresholdVal / 100) {
        btn.classList.add('visible');
    } else {
        btn.classList.remove('visible');
    }
}

const throttledScroll = throttle(onScroll, 16);
window.addEventListener('scroll', throttledScroll, { passive: true });
onScroll();

// Click
btn.addEventListener('click', function (e) {
    e.preventDefault();
    doScrollToTop();
});

// Keyboard accessibility
document.addEventListener('keydown', function (e) {
    if (e.key === 'Tab') { isKeyboard = true; }
});

btn.addEventListener('mousedown', function () { isKeyboard = false; });

btn.addEventListener('keydown', function (e) {
    if (e.key === 'Enter') {
        doScrollToTop();
        btn.classList.remove('keyboard-focus');
    }
});

btn.addEventListener('focus', function () {
    if (isKeyboard) { showTooltip(); btn.classList.add('keyboard-focus'); }
});

btn.addEventListener('blur', function () {
    hideTooltip();
    btn.classList.remove('keyboard-focus');
});

// Touch handlers
btn.addEventListener('touchstart', function (e) {
    e.preventDefault();
    btn.classList.add('active');
}, { passive: false });

btn.addEventListener('touchend', function (e) {
    e.preventDefault();
    doScrollToTop();
    btn.classList.remove('active');
}, { passive: false });

// Tooltip hover
btn.addEventListener('mouseenter', showTooltip);
btn.addEventListener('mouseleave', hideTooltip);

// Zoom detection -- hide at >300%
function checkZoom() {
    const zoom = Math.round((window.outerWidth / window.innerWidth) * 100) / 100;
    btn.style.display = zoom > 3 ? 'none' : 'flex';
}

window.addEventListener('resize', checkZoom);
checkZoom();
