const cfg = (window.__SARDE__ && window.__SARDE__.pluginConfig && window.__SARDE__.pluginConfig["image_lightbox"]) || {};

const config = {
    bgOpacity: cfg.background_opacity || 0.92,
};

const MIN_ZOOM = 0.5;
const MAX_ZOOM = 4;
const ZOOM_STEP = 0.25;

let lightbox = null;
let lightboxImg = null;
let lightboxCaption = null;
let zoomLevelEl = null;
let toolbarButtons = [];
let closeBtnEl = null;
let lastTrigger = null;

let currentZoom = 1;
let panX = 0;
let panY = 0;
let isDragging = false;
let dragStartX = 0;
let dragStartY = 0;
let panStartX = 0;
let panStartY = 0;

// -- Zoom & Pan --

function applyTransform() {
    if (!lightboxImg || !zoomLevelEl) return;
    lightboxImg.style.transform = 'scale(' + currentZoom + ') translate(' + panX + 'px, ' + panY + 'px)';
    zoomLevelEl.textContent = Math.round(currentZoom * 100) + '%';
    lightboxImg.style.cursor = currentZoom > 1 ? (isDragging ? 'grabbing' : 'grab') : 'default';
}

function applyZoom() { applyTransform(); }

function zoomIn() {
    currentZoom = Math.min(currentZoom + ZOOM_STEP, MAX_ZOOM);
    applyZoom();
}

function zoomOut() {
    currentZoom = Math.max(currentZoom - ZOOM_STEP, MIN_ZOOM);
    applyZoom();
}

function zoomReset() {
    currentZoom = 1;
    panX = 0;
    panY = 0;
    applyTransform();
}

// -- Lightbox Element --

function svgIcon(pathD, size) {
    const s = size || 18;
    return '<svg xmlns="http://www.w3.org/2000/svg" width="' + s + '" height="' + s + '" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' + pathD + '</svg>';
}

function ensureLightbox() {
    if (lightbox) return;

    lightbox = document.createElement('div');
    lightbox.className = 'sarde-image-lightbox';
    lightbox.id = 'sarde-image-lightbox';
    lightbox.setAttribute('role', 'dialog');
    lightbox.setAttribute('aria-modal', 'true');
    lightbox.setAttribute('aria-label', 'Image viewer');
    lightbox.style.setProperty('--il-bg-opacity', String(config.bgOpacity));

    const toolbar = document.createElement('div');
    toolbar.className = 'sarde-image-lightbox-toolbar';

    // Zoom out button
    const zoomOutBtn = document.createElement('button');
    zoomOutBtn.className = 'sarde-image-lightbox-btn';
    zoomOutBtn.setAttribute('aria-label', 'Zoom out');
    zoomOutBtn.innerHTML = svgIcon('<line x1="5" y1="12" x2="19" y2="12"/>');
    zoomOutBtn.addEventListener('click', function (e) { e.stopPropagation(); zoomOut(); });

    // Zoom level indicator (clickable to reset)
    zoomLevelEl = document.createElement('button');
    zoomLevelEl.className = 'sarde-image-lightbox-btn sarde-image-lightbox-zoom-level';
    zoomLevelEl.setAttribute('aria-label', 'Reset zoom');
    zoomLevelEl.textContent = '100%';
    zoomLevelEl.style.width = 'auto';
    zoomLevelEl.style.minWidth = '48px';
    zoomLevelEl.style.borderRadius = '20px';
    zoomLevelEl.style.fontSize = '0.75rem';
    zoomLevelEl.addEventListener('click', function (e) { e.stopPropagation(); zoomReset(); });

    // Zoom in button
    const zoomInBtn = document.createElement('button');
    zoomInBtn.className = 'sarde-image-lightbox-btn';
    zoomInBtn.setAttribute('aria-label', 'Zoom in');
    zoomInBtn.innerHTML = svgIcon('<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>');
    zoomInBtn.addEventListener('click', function (e) { e.stopPropagation(); zoomIn(); });

    // Close button
    const closeBtn = document.createElement('button');
    closeBtn.className = 'sarde-image-lightbox-btn sarde-image-lightbox-btn-close';
    closeBtn.setAttribute('aria-label', 'Close lightbox');
    closeBtn.innerHTML = '&times;';
    closeBtn.addEventListener('click', function (e) { e.stopPropagation(); closeLightbox(); });

    toolbar.appendChild(zoomOutBtn);
    toolbar.appendChild(zoomLevelEl);
    toolbar.appendChild(zoomInBtn);
    toolbar.appendChild(closeBtn);

    toolbarButtons = [zoomOutBtn, zoomLevelEl, zoomInBtn, closeBtn];
    closeBtnEl = closeBtn;

    lightboxImg = document.createElement('img');
    lightboxImg.className = 'sarde-image-lightbox-img';
    lightboxImg.alt = '';
    lightboxImg.draggable = false;

    // Drag-to-pan when zoomed in
    lightboxImg.addEventListener('mousedown', function (e) {
        if (currentZoom <= 1) return;
        e.preventDefault();
        e.stopPropagation();
        isDragging = true;
        dragStartX = e.clientX;
        dragStartY = e.clientY;
        panStartX = panX;
        panStartY = panY;
        if (lightbox) lightbox.classList.add('is-dragging');
        applyTransform();
    });

    document.addEventListener('mousemove', function (e) {
        if (!isDragging) return;
        e.preventDefault();
        panX = panStartX + (e.clientX - dragStartX) / currentZoom;
        panY = panStartY + (e.clientY - dragStartY) / currentZoom;
        applyTransform();
    });

    document.addEventListener('mouseup', function () {
        if (!isDragging) return;
        isDragging = false;
        if (lightbox) lightbox.classList.remove('is-dragging');
        applyTransform();
    });

    lightboxImg.addEventListener('touchstart', function (e) {
        if (currentZoom <= 1) return;
        if (e.touches.length !== 1) return;
        e.preventDefault();
        e.stopPropagation();
        isDragging = true;
        dragStartX = e.touches[0].clientX;
        dragStartY = e.touches[0].clientY;
        panStartX = panX;
        panStartY = panY;
        if (lightbox) lightbox.classList.add('is-dragging');
    }, { passive: false });

    document.addEventListener('touchmove', function (e) {
        if (!isDragging) return;
        if (e.touches.length !== 1) return;
        e.preventDefault();
        panX = panStartX + (e.touches[0].clientX - dragStartX) / currentZoom;
        panY = panStartY + (e.touches[0].clientY - dragStartY) / currentZoom;
        applyTransform();
    }, { passive: false });

    document.addEventListener('touchend', function () {
        if (!isDragging) return;
        isDragging = false;
        if (lightbox) lightbox.classList.remove('is-dragging');
        applyTransform();
    });

    lightboxImg.addEventListener('click', function (e) { e.stopPropagation(); });

    lightboxCaption = document.createElement('div');
    lightboxCaption.className = 'sarde-image-lightbox-caption';

    lightbox.appendChild(toolbar);
    lightbox.appendChild(lightboxImg);
    lightbox.appendChild(lightboxCaption);

    lightbox.addEventListener('click', function (e) {
        if (e.target === lightbox) { closeLightbox(); }
    });

    lightbox.addEventListener('wheel', function (e) {
        if (!lightbox.classList.contains('active')) return;
        e.preventDefault();
        if (e.deltaY < 0) { zoomIn(); } else { zoomOut(); }
    }, { passive: false });

    document.body.appendChild(lightbox);
}

// -- Open / Close --

function openLightbox(src, alt, trigger) {
    ensureLightbox();
    if (!lightbox || !lightboxImg || !lightboxCaption) return;
    lastTrigger = trigger || null;
    currentZoom = 1; panX = 0; panY = 0; isDragging = false;
    applyTransform();
    lightboxImg.src = src;
    lightboxImg.alt = alt;
    lightboxCaption.textContent = alt;
    lightbox.classList.add('active');
    document.body.style.overflow = 'hidden';
    if (closeBtnEl) closeBtnEl.focus();
}

function closeLightbox() {
    if (!lightbox) return;
    lightbox.classList.remove('active');
    document.body.style.overflow = '';
    if (lastTrigger) {
        lastTrigger.focus();
        lastTrigger = null;
    }
}

// -- Event Handlers --

function isLightboxable(img) {
    if (!img.closest('article.sarde-markdown-content')) return false;
    if (img.closest('.markdown-gallery')) return false;
    if (img.closest('.sarde-gallery-item')) return false;
    if (img.classList.contains('no-lightbox')) return false;
    if (img.naturalWidth > 0 && img.naturalWidth < 80) return false;
    return true;
}

function onClick(e) {
    const img = e.target;
    if (img.tagName !== 'IMG') return;
    if (!isLightboxable(img)) return;
    e.preventDefault();
    e.stopPropagation();
    openLightbox(img.dataset.src || img.src, img.alt || '', img);
}

function onKeyDown(e) {
    const active = lightbox && lightbox.classList.contains('active');

    if (!active) {
        // Enter/Space on a focused article image opens the lightbox.
        if ((e.key === 'Enter' || e.key === ' ') && e.target.tagName === 'IMG' && isLightboxable(e.target)) {
            e.preventDefault();
            openLightbox(e.target.dataset.src || e.target.src, e.target.alt || '', e.target);
        }
        return;
    }

    if (e.key === 'Escape') { e.preventDefault(); closeLightbox(); }
    else if (e.key === '+' || e.key === '=') { e.preventDefault(); zoomIn(); }
    else if (e.key === '-') { e.preventDefault(); zoomOut(); }
    else if (e.key === '0') { e.preventDefault(); zoomReset(); }
    else if (e.key === 'Tab') {
        // Keep focus cycling inside the modal toolbar while open.
        const idx = toolbarButtons.indexOf(document.activeElement);
        e.preventDefault();
        if (idx === -1) {
            toolbarButtons[0].focus();
        } else {
            const next = (idx + (e.shiftKey ? -1 : 1) + toolbarButtons.length) % toolbarButtons.length;
            toolbarButtons[next].focus();
        }
    }
}

// Make lightboxable article images reachable and operable by keyboard.
// The natural-width check in isLightboxable needs the image loaded, so it
// stays an open-time check; marking here is by structural exclusions only.
function markImages() {
    const imgs = document.querySelectorAll('article.sarde-markdown-content img');
    imgs.forEach(function (img) {
        if (img.closest('.markdown-gallery')) return;
        if (img.closest('.sarde-gallery-item')) return;
        if (img.classList.contains('no-lightbox')) return;
        img.setAttribute('tabindex', '0');
        img.setAttribute('role', 'button');
        if (!img.alt) img.setAttribute('aria-label', 'Open image in lightbox');
    });
}

// -- Init --

markImages();
document.addEventListener('click', onClick);
document.addEventListener('keydown', onKeyDown);
