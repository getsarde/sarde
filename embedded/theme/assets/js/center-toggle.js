(function () {
  'use strict';

  var STORAGE_KEY = 'sd-docs-centered';
  var BREAKPOINT = 1280;
  var CLASS_CENTERED = 'sd-docs-centered';

  var btn = document.getElementById('sarde-center-toggle');
  if (!btn) return;

  var mq = window.matchMedia('(min-width: ' + BREAKPOINT + 'px)');

  function isCentered() {
    return document.documentElement.classList.contains(CLASS_CENTERED);
  }

  function setAria(centered) {
    btn.setAttribute('aria-label', centered ? 'Expand content width' : 'Center content');
  }

  function center() {
    document.documentElement.classList.add(CLASS_CENTERED);
    setAria(true);
    try { localStorage.setItem(STORAGE_KEY, '1'); } catch (e) {}
  }

  function expand() {
    document.documentElement.classList.remove(CLASS_CENTERED);
    setAria(false);
    try { localStorage.removeItem(STORAGE_KEY); } catch (e) {}
  }

  function toggle() {
    if (isCentered()) {
      expand();
    } else {
      center();
    }
  }

  setAria(isCentered());

  btn.addEventListener('click', toggle);

  function onBreakpoint(e) {
    if (!e.matches) {
      document.documentElement.classList.remove(CLASS_CENTERED);
    } else {
      try {
        if (localStorage.getItem(STORAGE_KEY) === '1') {
          document.documentElement.classList.add(CLASS_CENTERED);
          setAria(true);
        }
      } catch (e) {}
    }
  }

  if (mq.addEventListener) {
    mq.addEventListener('change', onBreakpoint);
  } else {
    mq.addListener(onBreakpoint);
  }
})();
