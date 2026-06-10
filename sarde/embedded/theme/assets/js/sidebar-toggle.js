(function () {
  'use strict';

  var STORAGE_KEY = 'sd-sidebar';
  var BREAKPOINT = 1024;
  var CLASS_COLLAPSED = 'sd-sidebar-collapsed';

  var btn = document.getElementById('sarde-sidebar-toggle');
  if (!btn) return;

  var mq = window.matchMedia('(min-width: ' + BREAKPOINT + 'px)');

  function isCollapsed() {
    return document.documentElement.classList.contains(CLASS_COLLAPSED);
  }

  function setAria(collapsed) {
    btn.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
    btn.setAttribute('aria-label', collapsed ? 'Expand sidebar' : 'Collapse sidebar');
  }

  function collapse() {
    document.documentElement.classList.add(CLASS_COLLAPSED);
    setAria(true);
    try { localStorage.setItem(STORAGE_KEY, 'collapsed'); } catch (e) {}
  }

  function expand() {
    document.documentElement.classList.remove(CLASS_COLLAPSED);
    setAria(false);
    try { localStorage.setItem(STORAGE_KEY, 'expanded'); } catch (e) {}
  }

  function toggle() {
    if (isCollapsed()) {
      expand();
    } else {
      collapse();
    }
  }

  setAria(isCollapsed());

  btn.addEventListener('click', toggle);

  var sidebar = document.getElementById('sarde-sidebar');
  if (sidebar) {
    sidebar.addEventListener('click', function (e) {
      if (isCollapsed() && !btn.contains(e.target)) {
        expand();
      }
    });
  }

  function onBreakpoint(e) {
    if (!e.matches) {
      document.documentElement.classList.remove(CLASS_COLLAPSED);
    } else {
      try {
        if (localStorage.getItem(STORAGE_KEY) === 'collapsed') {
          document.documentElement.classList.add(CLASS_COLLAPSED);
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
