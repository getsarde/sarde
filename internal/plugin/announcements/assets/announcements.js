// Announcements Plugin: display modes, dismissal, rotation
(function () {
    'use strict';

    var STORAGE_PREFIX = 'announcement-dismissed-';
    var ROTATE_INDEX_KEY = 'announcement-rotate-index';

    var container = document.querySelector('.sarde-announcement-container');
    if (!container) return;

    var displayMode = container.dataset.displayMode || 'stack';
    var rotateInterval = parseInt(container.dataset.rotateInterval, 10) || 5000;
    var showIndicator = container.dataset.showRotateIndicator !== 'false';

    // ── Filtering ──
    // Dates and page targeting are evaluated by the inline prefilter, which
    // marks banners that do not apply to this page with data-applies="false".

    function isDismissed(id) {
        try { return !!localStorage.getItem(STORAGE_PREFIX + id); } catch (e) { return false; }
    }

    function dismiss(id) {
        try { localStorage.setItem(STORAGE_PREFIX + id, '1'); } catch (e) { /* noop */ }
    }

    function getAllBanners() {
        return Array.prototype.slice.call(
            container.querySelectorAll('.sarde-announcement-banner[data-announcement-id]')
        );
    }

    function getVisibleBanners() {
        return getAllBanners().filter(function (b) {
            var id = b.dataset.announcementId;
            return b.dataset.applies !== 'false' && !isDismissed(id);
        });
    }

    // Above the header, the layout offsets content by the banner strip while
    // a banner shows. Drop that offset once the last banner is dismissed.
    var inMasthead = !!container.closest('.sarde-masthead');

    function syncMasthead() {
        if (!inMasthead) return;
        var showing = getAllBanners().some(function (b) {
            return !b.classList.contains('dismissed');
        });
        document.documentElement.classList.toggle('sd-has-banner', showing);
        document.dispatchEvent(new CustomEvent('sarde:banner-change'));
    }

    // ── Stack mode ──

    function initStack() {
        var visible = getVisibleBanners();
        var all = getAllBanners();

        all.forEach(function (b) {
            if (visible.indexOf(b) === -1) b.classList.add('dismissed');
        });

        container.addEventListener('click', function (e) {
            var btn = e.target.closest('.sarde-announcement-dismiss');
            if (!btn) return;
            var id = btn.dataset.announcementId;
            var banner = btn.closest('.sarde-announcement-banner');
            if (banner) banner.classList.add('dismissed');
            if (id) dismiss(id);
            syncMasthead();
        });
    }

    // ── First mode ──

    function initFirst() {
        function showFirst() {
            var all = getAllBanners();
            var visible = getVisibleBanners();
            all.forEach(function (b) { b.classList.add('dismissed'); });
            if (visible.length > 0) visible[0].classList.remove('dismissed');
        }

        showFirst();

        container.addEventListener('click', function (e) {
            var btn = e.target.closest('.sarde-announcement-dismiss');
            if (!btn) return;
            var id = btn.dataset.announcementId;
            if (id) dismiss(id);
            showFirst();
            syncMasthead();
        });
    }

    // ── Rotate mode ──

    function initRotate() {
        var rotateVisible = getVisibleBanners();
        if (rotateVisible.length === 0) {
            getAllBanners().forEach(function (b) { b.classList.add('dismissed'); });
            return;
        }

        var rotateIndex = 0;
        var rotateTimer = null;
        var dotContainer = null;
        var prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');

        // Restore saved index
        try {
            var saved = parseInt(localStorage.getItem(ROTATE_INDEX_KEY) || '0', 10);
            if (saved >= 0 && saved < rotateVisible.length) rotateIndex = saved;
        } catch (e) { /* noop */ }

        // Hide all banners initially
        getAllBanners().forEach(function (b) {
            b.classList.add('dismissed');
            b.setAttribute('aria-hidden', 'true');
        });

        function goTo(index) {
            rotateVisible.forEach(function (b, i) {
                if (i === index) {
                    b.classList.remove('dismissed');
                    b.setAttribute('aria-hidden', 'false');
                } else {
                    b.classList.add('dismissed');
                    b.setAttribute('aria-hidden', 'true');
                }
            });
            rotateIndex = index;
            updateDots();
            try { localStorage.setItem(ROTATE_INDEX_KEY, String(index)); } catch (e) { /* noop */ }
        }

        function next() {
            if (rotateVisible.length === 0) return;
            goTo((rotateIndex + 1) % rotateVisible.length);
        }

        function startRotation() {
            if (rotateTimer) return;
            if (prefersReducedMotion.matches) return;
            if (rotateVisible.length <= 1) return;
            rotateTimer = setInterval(next, rotateInterval);
        }

        function stopRotation() {
            if (rotateTimer) {
                clearInterval(rotateTimer);
                rotateTimer = null;
            }
        }

        // ── Dot indicator ──

        function updateDots() {
            if (!dotContainer) return;
            var dots = dotContainer.querySelectorAll('.sarde-announcement-dot');
            dots.forEach(function (dot, i) {
                var isActive = i === rotateIndex;
                dot.classList.toggle('is-active', isActive);
                dot.setAttribute('aria-selected', String(isActive));
                dot.setAttribute('tabindex', isActive ? '0' : '-1');
            });
        }

        function buildDots() {
            if (dotContainer) dotContainer.remove();
            dotContainer = null;

            if (!showIndicator || rotateVisible.length <= 1) return;

            dotContainer = document.createElement('div');
            dotContainer.className = 'sarde-announcement-dots';
            dotContainer.setAttribute('role', 'tablist');
            dotContainer.setAttribute('aria-label', 'Announcements');

            rotateVisible.forEach(function (_, i) {
                var dot = document.createElement('button');
                dot.className = 'sarde-announcement-dot';
                dot.setAttribute('role', 'tab');
                dot.setAttribute('aria-label', 'Announcement ' + (i + 1) + ' of ' + rotateVisible.length);
                dot.setAttribute('aria-selected', 'false');
                dot.setAttribute('tabindex', '-1');
                dot.dataset.index = String(i);
                dot.addEventListener('click', function () {
                    stopRotation();
                    goTo(i);
                    startRotation();
                });
                dotContainer.appendChild(dot);
            });

            dotContainer.addEventListener('keydown', function (e) {
                var key = e.key;
                var handled = true;
                if (key === 'ArrowRight' || key === 'ArrowDown') {
                    goTo((rotateIndex + 1) % rotateVisible.length);
                } else if (key === 'ArrowLeft' || key === 'ArrowUp') {
                    goTo((rotateIndex - 1 + rotateVisible.length) % rotateVisible.length);
                } else if (key === 'Home') {
                    goTo(0);
                } else if (key === 'End') {
                    goTo(rotateVisible.length - 1);
                } else {
                    handled = false;
                }
                if (handled) {
                    e.preventDefault();
                    var activeDot = dotContainer.querySelector('.sarde-announcement-dot.is-active');
                    if (activeDot) activeDot.focus();
                }
            });

            container.appendChild(dotContainer);
            updateDots();
        }

        // ── Dismiss in rotate ──

        function handleDismiss(id) {
            dismiss(id);
            rotateVisible = rotateVisible.filter(function (b) {
                return b.dataset.announcementId !== id;
            });
            if (rotateVisible.length === 0) {
                stopRotation();
                getAllBanners().forEach(function (b) { b.classList.add('dismissed'); });
                if (dotContainer) dotContainer.remove();
                syncMasthead();
                return;
            }
            rotateIndex = Math.min(rotateIndex, rotateVisible.length - 1);
            buildDots();
            goTo(rotateIndex);
        }

        container.addEventListener('click', function (e) {
            var btn = e.target.closest('.sarde-announcement-dismiss');
            if (!btn) return;
            var id = btn.dataset.announcementId;
            if (id) handleDismiss(id);
        });

        // ── Pause on hover/focus (WCAG 2.2.2) ──

        container.addEventListener('mouseenter', stopRotation);
        container.addEventListener('mouseleave', startRotation);
        container.addEventListener('focusin', stopRotation);
        container.addEventListener('focusout', function (e) {
            if (!container.contains(e.relatedTarget)) startRotation();
        });

        // ── Reduced motion listener ──

        prefersReducedMotion.addEventListener('change', function () {
            if (prefersReducedMotion.matches) {
                stopRotation();
            } else {
                startRotation();
            }
        });

        // ── Init ──

        goTo(rotateIndex);
        buildDots();
        startRotation();
    }

    // ── Entry point ──

    switch (displayMode) {
        case 'first': initFirst(); break;
        case 'rotate': initRotate(); break;
        default: initStack(); break;
    }
})();
