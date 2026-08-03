const cfg = (window.__SARDE__ && window.__SARDE__.pluginConfig && window.__SARDE__.pluginConfig["copy_section_link"]) || {};

const config = {
    tooltipText: cfg.tooltip_text || 'Link copied!',
    tooltipDuration: cfg.tooltip_duration || 1500,
};

// Show a tooltip above the anchor element.
function showTooltip(anchor) {
    // Remove any existing tooltip
    const old = anchor.querySelector('.sarde-copy-section-tooltip');
    if (old) old.remove();

    const tip = document.createElement('span');
    tip.className = 'sarde-copy-section-tooltip';
    tip.textContent = config.tooltipText;
    anchor.appendChild(tip);

    // Animate in
    requestAnimationFrame(function () {
        requestAnimationFrame(function () {
            tip.classList.add('is-visible');
        });
    });

    // Fade out and remove
    setTimeout(function () {
        tip.classList.remove('is-visible');
        setTimeout(function () {
            tip.remove();
        }, 200);
    }, config.tooltipDuration);
}

// Handle clicks on heading anchors via event delegation.
function onClick(e) {
    const anchor = e.target.closest('.sarde-heading-anchor');
    if (!anchor) return;

    e.preventDefault();

    // Build the full URL
    const hash = anchor.getAttribute('href') || '';
    const url = window.location.origin + window.location.pathname + hash;

    // Copy to clipboard
    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(url).then(function () {
            showTooltip(anchor);
        });
    }
}

document.addEventListener('click', onClick);
