(function () {
  'use strict';
  function toggle(el) {
    var revealed = el.classList.toggle('revealed');
    el.setAttribute('aria-label', revealed ? 'Hide spoiler' : 'Reveal spoiler');
  }
  document.addEventListener('click', function (e) {
    var s = e.target.closest && e.target.closest('.spoiler');
    if (s) toggle(s);
  });
  document.addEventListener('keydown', function (e) {
    if (e.key !== 'Enter' && e.key !== ' ') return;
    var s = e.target.closest && e.target.closest('.spoiler');
    if (s) {
      e.preventDefault();
      toggle(s);
    }
  });
})();
