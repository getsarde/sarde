(function () {
  'use strict';
  var STORAGE_KEY = 'sarde-tabs';

  function getStore() {
    try { return JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}'); } catch (e) { return {}; }
  }

  function saveLabel(label) {
    var s = getStore();
    s[label] = 1;
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(s)); } catch (e) {}
  }

  function activateTab(container, btnSel, panelSel, label) {
    var buttons = container.querySelectorAll(btnSel);
    var found = false;
    buttons.forEach(function (b) {
      if (b.dataset.tabLabel === label) { found = true; }
    });
    if (!found) return;
    buttons.forEach(function (b) {
      var match = b.dataset.tabLabel === label;
      b.setAttribute('aria-selected', match ? 'true' : 'false');
      b.classList.toggle('is-active', match);
    });
    container.querySelectorAll(panelSel).forEach(function (p) {
      var match = p.dataset.tabLabel === label;
      p.classList.toggle('is-active', match);
      if (match) p.removeAttribute('hidden'); else p.setAttribute('hidden', '');
    });
  }

  function syncAll(label) {
    document.querySelectorAll('.sarde-tabs').forEach(function (c) {
      activateTab(c, '.sarde-tab-button', '.sarde-tab-panel', label);
    });

  }

  document.addEventListener('click', function (e) {
    if (!e.target.matches) return;
    if (e.target.matches('.sarde-tabs .sarde-tab-button')) {
      var label = e.target.dataset.tabLabel;
      if (label) {
        saveLabel(label);
        syncAll(label);
      }
    }
  });

  var stored = getStore();
  var labels = Object.keys(stored);
  for (var i = 0; i < labels.length; i++) {
    syncAll(labels[i]);
  }
})();
