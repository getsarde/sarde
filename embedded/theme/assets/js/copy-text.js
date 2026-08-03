(function () {
  'use strict';

  // Shared visually-hidden live region so copy confirmations reach screen
  // readers, not just the visual checkmark swap (WCAG 4.1.3).
  var liveRegion = null;
  function announce(text) {
    if (!liveRegion) {
      liveRegion = document.createElement('span');
      liveRegion.className = 'sr-only';
      liveRegion.setAttribute('role', 'status');
      document.body.appendChild(liveRegion);
    }
    liveRegion.textContent = '';
    setTimeout(function () { liveRegion.textContent = text; }, 50);
  }

  document.addEventListener('click', function (e) {
    var btn = e.target.closest('.sarde-copy-text__btn');
    if (!btn) return;

    var widget = btn.closest('.sarde-copy-text');
    if (!widget) return;

    var text = widget.getAttribute('data-copy-text');
    if (!text) return;

    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(function () {
        widget.classList.add('is-copied');
        announce('Copied to clipboard');
        setTimeout(function () {
          widget.classList.remove('is-copied');
        }, 1500);
      });
    }
  });
})();
