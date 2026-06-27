(function () {
  'use strict';

  var toggle = document.getElementById('sarde-nav-toggle');
  if (!toggle) return;

  var nav = document.getElementById(toggle.getAttribute('aria-controls'));
  if (!nav) return;

  function setOpen(open) {
    toggle.setAttribute('aria-expanded', open ? 'true' : 'false');
    nav.classList.toggle('is-open', open);
  }

  function isOpen() {
    return toggle.getAttribute('aria-expanded') === 'true';
  }

  // Toggle on button click.
  toggle.addEventListener('click', function (e) {
    e.stopPropagation();
    setOpen(!isOpen());
  });

  // Close when clicking outside the menu.
  document.addEventListener('click', function (e) {
    if (isOpen() && !nav.contains(e.target) && !toggle.contains(e.target)) {
      setOpen(false);
    }
  });

  // Close on Escape.
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape' && isOpen()) {
      setOpen(false);
      toggle.focus();
    }
  });

  // Close after following a link.
  nav.addEventListener('click', function (e) {
    if (e.target.closest('a[href]')) {
      setOpen(false);
    }
  });

  // Reset when crossing to the desktop breakpoint.
  var mq = window.matchMedia('(min-width: 1024px)');
  function onBreakpoint(e) {
    if (e.matches) setOpen(false);
  }
  if (mq.addEventListener) {
    mq.addEventListener('change', onBreakpoint);
  } else {
    mq.addListener(onBreakpoint);
  }
})();
