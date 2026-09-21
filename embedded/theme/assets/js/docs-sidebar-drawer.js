(function () {
  'use strict';

  var BREAKPOINT = 1024;

  var toggle = document.getElementById('sarde-menu-toggle');
  var sidebar = document.getElementById('sarde-sidebar');
  var backdrop = document.getElementById('sarde-sidebar-backdrop');
  var mainFrame = document.querySelector('.sarde-main-frame');
  var header = document.querySelector('.sarde-header');
  var mobileToc = document.getElementById('sarde-mobile-toc');
  var skipLink = document.querySelector('.sarde-skip-link');

  if (!toggle || !sidebar) return;

  var isOpen = false;
  var hideTimer = null;
  var mq = window.matchMedia('(min-width: ' + BREAKPOINT + 'px)');

  // Runs after the close fade; a no-op if the drawer was reopened meanwhile.
  function hideBackdrop() {
    if (isOpen || !backdrop) return;
    backdrop.style.display = 'none';
  }

  function setInert(flag) {
    var els = [mainFrame, header, mobileToc, skipLink];
    for (var i = 0; i < els.length; i++) {
      if (els[i]) {
        if (flag) els[i].setAttribute('inert', '');
        else els[i].removeAttribute('inert');
      }
    }
  }

  function open() {
    if (isOpen) return;
    isOpen = true;

    toggle.setAttribute('aria-expanded', 'true');
    sidebar.classList.add('is-open');
    sidebar.removeAttribute('inert');
    document.body.classList.add('sarde-sidebar-open');

    if (backdrop) {
      clearTimeout(hideTimer);
      backdrop.style.display = 'block';
      backdrop.offsetHeight;
      backdrop.classList.add('is-visible');
    }

    setInert(true);

    var firstLink = sidebar.querySelector('a[href]');
    if (firstLink) {
      setTimeout(function () { firstLink.focus(); }, 50);
    }
  }

  function close() {
    if (!isOpen) return;
    isOpen = false;
    setInert(false);

    toggle.setAttribute('aria-expanded', 'false');
    sidebar.classList.remove('is-open');
    if (!mq.matches) {
      sidebar.setAttribute('inert', '');
    }
    document.body.classList.remove('sarde-sidebar-open');

    if (backdrop) {
      backdrop.classList.remove('is-visible');
      // Fallback for when the fade never fires transitionend.
      clearTimeout(hideTimer);
      hideTimer = setTimeout(hideBackdrop, 350);
    }

    toggle.focus();
  }

  // Idempotent: returns the page to a closed, interactive state from any
  // starting point. Never moves focus.
  function reset() {
    isOpen = false;
    clearTimeout(hideTimer);
    setInert(false);
    toggle.setAttribute('aria-expanded', 'false');
    sidebar.classList.remove('is-open');
    if (mq.matches) {
      sidebar.removeAttribute('inert');
    } else {
      sidebar.setAttribute('inert', '');
    }
    document.body.classList.remove('sarde-sidebar-open');
    if (backdrop) {
      backdrop.classList.remove('is-visible');
      backdrop.style.display = 'none';
    }
  }

  // Initialize
  reset();
  if (backdrop) {
    backdrop.addEventListener('transitionend', hideBackdrop);
  }

  // A page restored from the back/forward cache with the drawer open would
  // come back with the header and content still inert.
  window.addEventListener('pageshow', function (e) {
    if (e.persisted) reset();
  });

  // Hamburger toggle
  toggle.addEventListener('click', function () {
    if (isOpen) close(); else open();
  });

  // Backdrop click
  if (backdrop) {
    backdrop.addEventListener('click', close);
  }

  // Escape key
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape' && isOpen) {
      close();
    }
  });

  // Close on navigation link click
  sidebar.addEventListener('click', function (e) {
    var link = e.target.closest('a[href]');
    if (link && link.getAttribute('href') && link.getAttribute('href').charAt(0) !== '#') {
      close();
    }
  });

  // Resize: reset when crossing to desktop
  function onBreakpoint(e) {
    if (e.matches || !isOpen) reset();
  }

  if (mq.addEventListener) {
    mq.addEventListener('change', onBreakpoint);
  } else {
    mq.addListener(onBreakpoint);
  }
})();
