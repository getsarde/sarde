const cfg = (window.__SARDE__ && window.__SARDE__.pluginConfig && window.__SARDE__.pluginConfig["scroll_to_top"]) || {};

const THRESHOLD = cfg.threshold != null && cfg.threshold >= 0 ? cfg.threshold : 300;
const POSITION = cfg.position || 'center';
const SHOW_TOOLTIP = cfg.show_tooltip || false;
const SHOW_PROGRESS_RING = cfg.show_progress_ring || false;
const SMOOTH_SCROLL = cfg.smooth_scroll !== false;
const BORDER_RADIUS = (cfg.border_radius != null ? cfg.border_radius : 15) + '%';
const PROGRESS_RING_COLOR = cfg.progress_ring_color || null;
const SVG_PATH = 'M18 15l-6-6-6 6';
let isKeyboard = false;

const footer = document.querySelector('.sarde-footer');

// Create button
const btn = document.createElement('button');
btn.className = 'sarde-scroll-to-top pos-' + POSITION;
btn.style.borderRadius = BORDER_RADIUS;
if (PROGRESS_RING_COLOR) btn.style.setProperty('--scroll-progress-color', PROGRESS_RING_COLOR);
btn.setAttribute('aria-label', 'Scroll to top');
btn.setAttribute('role', 'button');
btn.setAttribute('tabindex', '0');

// Build inner HTML (hardcoded SVG -- no user data)
const progressRingHtml = SHOW_PROGRESS_RING
    ? '<svg class="sarde-scroll-progress-ring" viewBox="0 0 47 47">' +
      '<circle cx="23.5" cy="23.5" r="22" fill="none" stroke-width="3" class="sarde-scroll-progress-track"/>' +
      '<circle cx="23.5" cy="23.5" r="22" fill="none" stroke-width="3" stroke-linecap="round" class="sarde-scroll-progress-circle" style="transform:rotate(-90deg);transform-origin:center;"/>' +
      '</svg>'
    : '';

btn.innerHTML = progressRingHtml +
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="' + SVG_PATH + '"/></svg>';

document.body.appendChild(btn);

// Tooltip
let tooltip = null;
if (SHOW_TOOLTIP) {
    tooltip = document.createElement('div');
    tooltip.className = 'sarde-scroll-to-top-tooltip';
    tooltip.id = 'scroll-to-top-tooltip';
    tooltip.textContent = 'Scroll to top';
    const arrow = document.createElement('div');
    arrow.className = 'sarde-scroll-to-top-tooltip-arrow';
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
    btn.classList.remove('is-active');
}

// Coalesce scroll events into one update per frame. A plain leading-edge
// throttle drops the trailing event of a burst, which left the button in a
// stale state after inertial scrolls (the final position was never processed).
let scrollScheduled = false;
function scheduleScroll() {
    if (scrollScheduled) return;
    scrollScheduled = true;
    requestAnimationFrame(function () {
        scrollScheduled = false;
        onScroll();
    });
}

// Show/hide based on scroll position (pixels) + footer proximity
function onScroll() {
    const scrollPos = window.scrollY;
    const viewportHeight = window.innerHeight;
    const pageHeight = document.documentElement.scrollHeight;

    // Update progress ring
    if (SHOW_PROGRESS_RING) {
        const scrollPct = scrollPos / (pageHeight - viewportHeight);
        const circle = btn.querySelector('.sarde-scroll-progress-circle');
        if (circle) {
            const progress = Math.min(Math.max(scrollPct * 100, 0), 100);
            const circumference = 138.23;
            circle.style.strokeDashoffset = (circumference - (progress / 100) * circumference).toString();
        }
    }

    const pastThreshold = scrollPos > THRESHOLD;
    // Footer-hide only applies when the page leaves a usable window between
    // "past threshold" and "footer in view". On tall viewports a short page
    // brings the footer into view before the threshold is ever crossed, and
    // hiding there would suppress the button for the whole page.
    const footerEnterScroll = footer
        ? footer.getBoundingClientRect().top + scrollPos - viewportHeight
        : Infinity;
    const atFooter = footerEnterScroll > THRESHOLD + 100 && scrollPos >= footerEnterScroll;
    const shouldShow = pastThreshold && !atFooter;

    btn.classList.toggle('visible', shouldShow);
}

window.addEventListener('scroll', scheduleScroll, { passive: true });
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
    btn.classList.add('is-active');
}, { passive: false });

btn.addEventListener('touchend', function (e) {
    e.preventDefault();
    doScrollToTop();
    btn.classList.remove('is-active');
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
