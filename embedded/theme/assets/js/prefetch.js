(function () {
  'use strict';
  if (!('IntersectionObserver' in window)) return;
  var prefetched = new Set();
  function prefetch(href) {
    if (prefetched.has(href)) return;
    prefetched.add(href);
    var link = document.createElement('link');
    link.rel = 'prefetch';
    link.href = href;
    document.head.appendChild(link);
  }
  function sameOrigin(href) {
    try {
      var u = new URL(href, location.href);
      return u.origin === location.origin && u.pathname !== location.pathname;
    } catch (_) {
      return false;
    }
  }
  document.addEventListener('mouseover', function (e) {
    var a = e.target.closest && e.target.closest('a[href]');
    if (!a) return;
    var href = a.getAttribute('href');
    if (!href || href.startsWith('#')) return;
    if (sameOrigin(a.href)) prefetch(a.href);
  });
  var io = new IntersectionObserver(function (entries) {
    entries.forEach(function (entry) {
      if (!entry.isIntersecting) return;
      var a = entry.target;
      if (sameOrigin(a.href)) prefetch(a.href);
      io.unobserve(a);
    });
  }, { rootMargin: '200px' });
  document.querySelectorAll('a[href]').forEach(function (a) {
    var href = a.getAttribute('href');
    if (!href || href.startsWith('#') || href.startsWith('http')) return;
    io.observe(a);
  });
})();
