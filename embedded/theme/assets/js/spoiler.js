(function () {
  'use strict';
  var i18n = (window.__SARDE__ && window.__SARDE__.i18n) || {};
  var LABEL_REVEAL = i18n.revealSpoiler || 'Reveal spoiler';
  var LABEL_HIDE = i18n.hideSpoiler || 'Hide spoiler';
  function toggle(el) {
    var revealed = el.classList.toggle('revealed');
    el.setAttribute('aria-label', revealed ? LABEL_HIDE : LABEL_REVEAL);
    el.setAttribute('aria-expanded', revealed ? 'true' : 'false');
  }
  // The server-rendered initial label is English; localize it at load.
  document.querySelectorAll('.sarde-spoiler:not(.revealed)').forEach(function (el) {
    el.setAttribute('aria-label', LABEL_REVEAL);
  });
  document.addEventListener('click', function (e) {
    var s = e.target.closest && e.target.closest('.sarde-spoiler');
    if (s) toggle(s);
  });
  document.addEventListener('keydown', function (e) {
    if (e.key !== 'Enter' && e.key !== ' ') return;
    var s = e.target.closest && e.target.closest('.sarde-spoiler');
    if (s) {
      e.preventDefault();
      toggle(s);
    }
  });
})();
