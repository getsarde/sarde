(function () {
  'use strict';
  function activate(container, tabIdx, buttonSel, panelSel) {
    var buttons = container.querySelectorAll(buttonSel);
    var panels = container.querySelectorAll(panelSel);
    buttons.forEach(function (b) {
      var active = b.getAttribute('data-tab') === tabIdx;
      b.setAttribute('aria-selected', active ? 'true' : 'false');
      b.classList.toggle('active', active);
    });
    panels.forEach(function (p) {
      var active = p.getAttribute('data-tab') === tabIdx;
      p.classList.toggle('active', active);
      if (active) p.removeAttribute('hidden');
      else p.setAttribute('hidden', '');
    });
  }
  document.addEventListener('click', function (e) {
    if (!e.target.matches) return;
    if (e.target.matches('.tabs .tab-button')) {
      var container = e.target.closest('.tabs');
      activate(container, e.target.getAttribute('data-tab'), '.tab-button', '.tab-panel');
    } else if (e.target.matches('.code-group .code-group-tab')) {
      var cg = e.target.closest('.code-group');
      activate(cg, e.target.getAttribute('data-tab'), '.code-group-tab', '.code-group-panel');
    }
  });
})();
