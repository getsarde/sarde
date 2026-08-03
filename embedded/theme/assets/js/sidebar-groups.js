(function () {
  'use strict';

  var STORAGE_KEY = 'sd-sb';

  function getNav() {
    return document.querySelector('.sarde-sidebar-nav');
  }

  function groupToggle(group) {
    return group.querySelector('button[aria-expanded]');
  }

  function saveState() {
    try {
      var nav = getNav();
      if (!nav) return;
      var hash = nav.getAttribute('data-sd-hash');
      if (!hash) return;
      var groups = nav.querySelectorAll('[data-sd-idx]');
      var open = [];
      for (var i = 0; i < groups.length; i++) {
        var idx = parseInt(groups[i].getAttribute('data-sd-idx'), 10);
        var btn = groupToggle(groups[i]);
        if (idx >= 0 && btn) open[idx] = btn.getAttribute('aria-expanded') === 'true';
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

  nav.addEventListener('click', function (e) {
    var btn = e.target.closest('button[aria-controls]');
    if (!btn || !nav.contains(btn)) return;
    var list = document.getElementById(btn.getAttribute('aria-controls'));
    if (!list) return;
    var expanded = btn.getAttribute('aria-expanded') === 'true';
    btn.setAttribute('aria-expanded', expanded ? 'false' : 'true');
    if (expanded) {
      list.setAttribute('hidden', '');
    } else {
      list.removeAttribute('hidden');
    }
    saveState();
  });

  document.addEventListener('visibilitychange', function () {
    if (document.visibilityState === 'hidden') saveState();
  });

  window.addEventListener('pagehide', saveState);
})();
