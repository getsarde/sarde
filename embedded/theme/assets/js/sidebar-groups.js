(function () {
  'use strict';

  var STORAGE_KEY = 'sd-sb';

  function getNav() {
    return document.querySelector('.sarde-sidebar-nav');
  }

  function saveState() {
    try {
      var nav = getNav();
      if (!nav) return;
      var hash = nav.getAttribute('data-sd-hash');
      if (!hash) return;
      var groups = nav.querySelectorAll('details[data-sd-idx]');
      var open = [];
      for (var i = 0; i < groups.length; i++) {
        var idx = parseInt(groups[i].getAttribute('data-sd-idx'), 10);
        if (idx >= 0) open[idx] = groups[i].open;
      }
      sessionStorage.setItem(STORAGE_KEY, JSON.stringify({
        hash: hash,
        open: open,
        scroll: nav.scrollTop
      }));
    } catch (e) {}
  }

  var nav = getNav();
  if (!nav) return;

  nav.addEventListener('toggle', function (e) {
    if (e.target && e.target.hasAttribute('data-sd-idx')) {
      saveState();
    }
  }, true);

  document.addEventListener('visibilitychange', function () {
    if (document.visibilityState === 'hidden') saveState();
  });

  window.addEventListener('pagehide', saveState);
})();
