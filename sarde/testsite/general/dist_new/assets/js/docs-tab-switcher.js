(function () {
  'use strict';

  function closeAll() {
    document.querySelectorAll('[data-sarde-tab-trigger][aria-expanded="true"]').forEach(function (trigger) {
      trigger.setAttribute('aria-expanded', 'false');
      var menu = trigger.nextElementSibling;
      if (menu) menu.setAttribute('hidden', '');
    });
  }

  document.addEventListener('click', function (e) {
    var trigger = e.target.closest('[data-sarde-tab-trigger]');
    if (trigger) {
      e.stopPropagation();
      var expanded = trigger.getAttribute('aria-expanded') === 'true';
      closeAll();
      if (!expanded) {
        trigger.setAttribute('aria-expanded', 'true');
        var menu = trigger.nextElementSibling;
        if (menu) menu.removeAttribute('hidden');
      }
      return;
    }
    if (!e.target.closest('[data-sarde-tab-menu]')) {
      closeAll();
    }
  });

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') closeAll();
  });
})();
