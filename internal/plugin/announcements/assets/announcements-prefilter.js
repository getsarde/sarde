// Announcements prefilter: emitted inline right after the banner container,
// so it runs before the header below is parsed. Banners that do not apply to
// this page never paint; announcements.js adds interaction later.
(function () {
    'use strict';

    var container = document.currentScript && document.currentScript.previousElementSibling;
    if (!container || !container.classList.contains('sarde-announcement-container')) return;

    function matchGlob(pattern, path) {
        var escaped = pattern
            .replace(/[.+^${}()|[\]\\]/g, '\\$&')
            .replace(/\*\*/g, '\x01')
            .replace(/\*/g, '[^/]*')
            .replace(/\x01/g, '.*');
        return new RegExp('^' + escaped + '$').test(path);
    }

    function isDismissed(id) {
        try { return !!localStorage.getItem('announcement-dismissed-' + id); } catch (e) { return false; }
    }

    function isDateActive(banner) {
        var now = Date.now();
        var start = banner.dataset.startDate;
        var end = banner.dataset.endDate;
        if (start && now < new Date(start).getTime()) return false;
        if (end && now > new Date(end).getTime()) return false;
        return true;
    }

    function isPageMatch(banner) {
        var path = window.location.pathname;
        var showOn = banner.dataset.showOn;
        var hideOn = banner.dataset.hideOn;
        var showPatterns = showOn ? showOn.split(',') : ['/**'];
        var hidePatterns = hideOn ? hideOn.split(',') : [];
        var shown = showPatterns.some(function (p) { return matchGlob(p.trim(), path); });
        var hidden = hidePatterns.some(function (p) { return matchGlob(p.trim(), path); });
        return shown && !hidden;
    }

    // Above the header the strip has a fixed height, so it shows one banner
    // at a time: stack behaves like first.
    var inMasthead = !!container.closest('.sarde-masthead');
    if (inMasthead && (container.dataset.displayMode || 'stack') === 'stack') {
        container.dataset.displayMode = 'first';
    }
    var single = (container.dataset.displayMode || 'stack') !== 'stack';

    var shown = 0;
    var banners = container.querySelectorAll('.sarde-announcement-banner[data-announcement-id]');
    Array.prototype.forEach.call(banners, function (b) {
        var applies = !isDismissed(b.dataset.announcementId) && isDateActive(b) && isPageMatch(b);
        if (!applies) {
            b.dataset.applies = 'false';
            b.classList.add('dismissed');
        } else if (single && shown > 0) {
            b.classList.add('dismissed');
        } else {
            shown++;
        }
    });

    if (inMasthead && shown > 0) {
        document.documentElement.classList.add('sd-has-banner');
    }
})();
