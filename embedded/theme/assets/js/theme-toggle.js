(function () {
  var STORAGE_KEY = 'sd-theme';

  function getTheme() {
    return localStorage.getItem(STORAGE_KEY) || 'system';
  }

  function isDark(theme) {
    return theme === 'dark' || (theme === 'system' && matchMedia('(prefers-color-scheme: dark)').matches);
  }

  function applyTheme(theme) {
    var root = document.documentElement;
    root.classList.add('sd-no-transition');
    root.setAttribute('data-theme', isDark(theme) ? 'dark' : 'light');
    requestAnimationFrame(function () {
      requestAnimationFrame(function () {
        root.classList.remove('sd-no-transition');
      });
    });
  }

  function syncToggles(theme) {
    document.querySelectorAll('[data-sarde-theme-toggle]').forEach(function (toggle) {
      var indicator = toggle.querySelector('[data-sarde-theme-indicator]');
      toggle.querySelectorAll('.sarde-theme-toggle-btn').forEach(function (btn) {
        var active = btn.getAttribute('data-theme') === theme;
        btn.classList.toggle('selected', active);
        btn.setAttribute('aria-pressed', active);
      });
      if (indicator) {
        indicator.classList.toggle('pos-system', theme === 'system');
        indicator.classList.toggle('pos-dark', theme === 'dark');
      }
    });
  }

  function setTheme(theme) {
    localStorage.setItem(STORAGE_KEY, theme);
    applyTheme(theme);
    syncToggles(theme);
  }

  document.addEventListener('click', function (e) {
    var btn = e.target.closest('.sarde-theme-toggle-btn');
    if (btn) setTheme(btn.getAttribute('data-theme'));
  });

  matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function () {
    applyTheme(getTheme());
  });

  var theme = getTheme();
  applyTheme(theme);
  syncToggles(theme);
})();
