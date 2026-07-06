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
  var mq = window.matchMedia('(min-width: ' + BREAKPOINT + 'px)');

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
    sidebar.removeAttribute('aria-hidden');
    document.body.classList.add('sarde-sidebar-open');

    if (backdrop) {
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
      sidebar.setAttribute('aria-hidden', 'true');
    }
    document.body.classList.remove('sarde-sidebar-open');

    if (backdrop) {
      backdrop.classList.remove('is-visible');
      var hidden = false;
      function hide() {
        if (hidden || isOpen) return;
        hidden = true;
        backdrop.style.display = 'none';
      }
      backdrop.addEventListener('transitionend', function handler() {
        hide();
        backdrop.removeEventListener('transitionend', handler);
      });
      setTimeout(hide, 350);
    }

    toggle.focus();
  }

  function reset() {
    isOpen = false;
    setInert(false);
    toggle.setAttribute('aria-expanded', 'false');
    sidebar.classList.remove('is-open');
    sidebar.removeAttribute('aria-hidden');
    document.body.classList.remove('sarde-sidebar-open');
    if (backdrop) {
      backdrop.classList.remove('is-visible');
      backdrop.style.display = 'none';
    }
  }

  // Initialize
  if (!mq.matches) {
    sidebar.setAttribute('aria-hidden', 'true');
  }
  if (backdrop) {
    backdrop.style.display = 'none';
  }

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
    if (e.matches) {
      reset();
    } else if (!isOpen) {
      sidebar.setAttribute('aria-hidden', 'true');
    }
  }

  if (mq.addEventListener) {
    mq.addEventListener('change', onBreakpoint);
  } else {
    mq.addListener(onBreakpoint);
  }
})();
