(function () {
  'use strict';
  function flash(btn, text) {
    var original = btn.textContent;
    btn.textContent = text;
    btn.classList.add('copied');
    setTimeout(function () {
      btn.textContent = original;
      btn.classList.remove('copied');
    }, 1500);
  }
  document.addEventListener('click', function (e) {
    var btn = e.target.closest && e.target.closest('.copy-btn');
    if (!btn) return;
    var code = btn.getAttribute('data-code') || '';
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(code).then(
        function () { flash(btn, 'Copied'); },
        function () { flash(btn, 'Failed'); }
      );
      return;
    }
    var ta = document.createElement('textarea');
    ta.value = code;
    ta.setAttribute('readonly', '');
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand('copy');
      flash(btn, 'Copied');
    } catch (_) {
      flash(btn, 'Failed');
    }
    document.body.removeChild(ta);
  });
})();
