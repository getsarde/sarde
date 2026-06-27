(function () {
  'use strict';

  var desktopLinks = document.querySelectorAll('.sarde-toc-nav a');
  var mobileLinks = document.querySelectorAll('.sarde-mobile-toc-link');
  var mobileToc = document.getElementById('sarde-mobile-toc');
  var currentSpan = mobileToc ? mobileToc.querySelector('.sarde-mobile-toc-current') : null;

  var headingIds = [];
  var seen = {};
  [].forEach.call(desktopLinks, function (link) {
    var id = (link.getAttribute('href') || '').replace('#', '');
    if (id && id !== '_top' && !seen[id]) { seen[id] = true; headingIds.push(id); }
  });
  [].forEach.call(mobileLinks, function (link) {
    var id = (link.getAttribute('href') || '').replace('#', '');
    if (id && id !== '_top' && !seen[id]) { seen[id] = true; headingIds.push(id); }
  });

  if (headingIds.length === 0) return;

  var firstHeadingId = headingIds[0];
  var lastHeadingId = headingIds[headingIds.length - 1];

  // ── Narrow 53px detection strip just below the nav ────────────────
  function getStripRootMargin() {
    var header = document.querySelector('.sarde-header');
    var navHeight = header ? header.offsetHeight : 64;
    var mobileTocH = (window.innerWidth < 1280 && mobileToc) ? 48 : 0;
    var topOffset = navHeight + mobileTocH + 32;
    var bottomOffset = topOffset + 53 - window.innerHeight;
    return '-' + topOffset + 'px 0% ' + bottomOffset + 'px';
  }

  // ── Attribute a content element to its nearest preceding heading ───
  function getElementHeading(el) {
    var current = el;
    while (current) {
      if (/^H[2-6]$/.test(current.nodeName) && current.id) return current;
      var prev = current.previousElementSibling;
      while (prev) {
        if (/^H[2-6]$/.test(prev.nodeName) && prev.id) return prev;
        var last = prev.lastElementChild;
        while (last) {
          if (/^H[2-6]$/.test(last.nodeName) && last.id) return last;
          last = last.lastElementChild;
        }
        prev = prev.previousElementSibling;
      }
      current = current.parentElement;
    }
    return null;
  }

  // ── Set the active TOC link ───────────────────────────────────────
  function setActive(id) {
    [].forEach.call(desktopLinks, function (link) {
      var isActive = link.getAttribute('href') === '#' + id;
      link.classList.toggle('active', isActive);
      link.setAttribute('aria-current', isActive ? 'true' : 'false');
    });

    var activeText = '';
    [].forEach.call(mobileLinks, function (link) {
      var isActive = link.getAttribute('href') === '#' + id;
      link.setAttribute('aria-current', isActive ? 'true' : 'false');
      if (isActive) activeText = link.textContent.trim();
    });
    if (currentSpan) currentSpan.textContent = activeText;
  }

  // ── Collect all elements to observe ───────────────────────────────
  var elementsToObserve = [];

  function collectElements() {
    elementsToObserve = [];
    var content = document.querySelector('article.sarde-markdown-content');
    if (!content) {
      headingIds.forEach(function (id) {
        var el = document.getElementById(id);
        if (el) elementsToObserve.push(el);
      });
      return;
    }
    var children = content.children;
    for (var i = 0; i < children.length; i++) {
      elementsToObserve.push(children[i]);
    }
    var deepHeadings = content.querySelectorAll('h2[id], h3[id], h4[id], h5[id], h6[id]');
    [].forEach.call(deepHeadings, function (h) {
      if (h.parentElement !== content) elementsToObserve.push(h);
    });
  }

  // ── IntersectionObserver ──────────────────────────────────────────
  var observer = null;

  function observerCallback(entries) {
    for (var i = 0; i < entries.length; i++) {
      if (entries[i].isIntersecting) {
        var heading = getElementHeading(entries[i].target);
        if (heading && heading.id) {
          setActive(heading.id);
        } else {
          setActive('_top');
        }
        break;
      }
    }
  }

  function buildObserver() {
    if (observer) observer.disconnect();
    observer = new IntersectionObserver(observerCallback, {
      rootMargin: getStripRootMargin(),
      threshold: 0
    });
    elementsToObserve.forEach(function (el) { observer.observe(el); });
  }

  // ── Resize: rebuild observer (rootMargin depends on header height)
  var resizeTimer = null;
  window.addEventListener('resize', function () {
    if (resizeTimer) clearTimeout(resizeTimer);
    resizeTimer = setTimeout(buildObserver, 200);
  }, { passive: true });

  // ── Smooth scroll on TOC link click ───────────────────────────────
  function getNavOffset() {
    var header = document.querySelector('.sarde-header');
    var base = header ? header.offsetHeight : 64;
    var mobileTocH = (window.innerWidth < 1024 && mobileToc) ? 48 : 0;
    return base + mobileTocH + 8;
  }

  [].forEach.call(desktopLinks, function (link) {
    link.addEventListener('click', function (e) {
      e.preventDefault();
      var id = this.getAttribute('href').replace('#', '');
      if (id === '_top') {
        window.scrollTo({ top: 0, behavior: 'smooth' });
        history.replaceState(null, '', window.location.pathname);
        setActive('_top');
      } else {
        var el = document.getElementById(id);
        if (el) {
          window.scrollTo({ top: el.offsetTop - getNavOffset(), behavior: 'smooth' });
        }
        history.replaceState(null, '', '#' + id);
      }
    });
  });

  // ── Mobile TOC behavior ───────────────────────────────────────────
  if (mobileToc) {
    [].forEach.call(mobileLinks, function (link) {
      link.addEventListener('click', function () {
        setTimeout(function () { mobileToc.open = false; }, 0);
      });
    });

    document.addEventListener('click', function (e) {
      if (mobileToc.open && !mobileToc.contains(e.target)) {
        mobileToc.open = false;
      }
    });

    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape' && mobileToc.open) {
        mobileToc.open = false;
        var summary = mobileToc.querySelector('summary');
        if (summary) summary.focus();
      }
    });
  }

  // ── Circular progress indicator ───────────────────────────────────
  var progressFill = mobileToc ? mobileToc.querySelector('.sarde-mobile-toc-progress-fill') : null;
  var circumference = 2 * Math.PI * 6;
  if (progressFill) {
    progressFill.style.strokeDasharray = circumference;
    progressFill.style.strokeDashoffset = circumference;
  }

  var scrollRaf = null;
  window.addEventListener('scroll', function () {
    if (scrollRaf) return;
    scrollRaf = requestAnimationFrame(function () {
      scrollRaf = null;
      if (progressFill) {
        var scrollTop = window.scrollY;
        var docHeight = document.documentElement.scrollHeight - window.innerHeight;
        var progress = docHeight > 0 ? Math.min(scrollTop / docHeight, 1) : 0;
        progressFill.style.strokeDashoffset = circumference * (1 - progress);
      }
      var atBottom = (window.innerHeight + window.scrollY) >=
        (document.documentElement.scrollHeight - 50);
      if (atBottom) setActive(lastHeadingId);
    });
  }, { passive: true });

  // ── Init (deferred to avoid blocking first paint) ─────────────────
  function init() {
    setActive('_top');
    collectElements();
    buildObserver();
  }

  if ('requestIdleCallback' in window) {
    requestIdleCallback(init);
  } else {
    setTimeout(init, 16);
  }
})();
