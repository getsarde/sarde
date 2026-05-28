(function () {
  'use strict';

  var bar = document.getElementById('starter-progress');
  if (!bar) return;

  function update() {
    var el = document.documentElement;
    var scrollTop = el.scrollTop || document.body.scrollTop;
    var scrollHeight = el.scrollHeight - el.clientHeight;
    var pct = scrollHeight > 0 ? (scrollTop / scrollHeight) * 100 : 0;
    bar.style.width = Math.min(pct, 100).toFixed(2) + '%';
  }

  document.addEventListener('scroll', update, { passive: true });
  update();
})();
