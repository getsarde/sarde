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
    if (id && !seen[id]) { seen[id] = true; headingIds.push(id); }
  });
  [].forEach.call(mobileLinks, function (link) {
    var id = (link.getAttribute('href') || '').replace('#', '');
    if (id && !seen[id]) { seen[id] = true; headingIds.push(id); }
  });

  if (headingIds.length === 0) return;

  function getNavOffset() {
    var header = document.querySelector('.sarde-header');
    var base = header ? header.offsetHeight : 64;
    var mobileTocH = (window.innerWidth < 1024 && mobileToc) ? 48 : 0;
    return base + mobileTocH + 8;
  }

  function setActive(id) {
    [].forEach.call(desktopLinks, function (link) {
      link.classList.toggle('active', link.getAttribute('href') === '#' + id);
    });

    var activeText = '';
    [].forEach.call(mobileLinks, function (link) {
      var isActive = link.getAttribute('href') === '#' + id;
      link.setAttribute('aria-current', isActive ? 'true' : 'false');
      if (isActive) activeText = link.textContent.trim();
    });
    if (currentSpan) currentSpan.textContent = activeText;
  }

  var observer = new IntersectionObserver(function (entries) {
    entries.forEach(function (entry) {
      if (entry.isIntersecting) {
        setActive(entry.target.id);
      }
    });
  }, { rootMargin: '-' + getNavOffset() + 'px 0px -60% 0px', threshold: 0 });

  headingIds.forEach(function (id) {
    var el = document.getElementById(id);
    if (el) observer.observe(el);
  });

  [].forEach.call(desktopLinks, function (link) {
    link.addEventListener('click', function (e) {
      e.preventDefault();
      var id = this.getAttribute('href').replace('#', '');
      var el = document.getElementById(id);
      if (el) {
        window.scrollTo({ top: el.offsetTop - getNavOffset(), behavior: 'smooth' });
      }
      history.replaceState(null, '', '#' + id);
    });
  });

  if (!mobileToc) return;

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

  var progressFill = mobileToc.querySelector('.sarde-mobile-toc-progress-fill');
  if (progressFill) {
    var circumference = 2 * Math.PI * 6;
    progressFill.style.strokeDasharray = circumference;
    progressFill.style.strokeDashoffset = circumference;
    var scrollRaf = null;
    window.addEventListener('scroll', function () {
      if (scrollRaf) return;
      scrollRaf = requestAnimationFrame(function () {
        scrollRaf = null;
        var scrollTop = window.scrollY;
        var docHeight = document.documentElement.scrollHeight - window.innerHeight;
        var progress = docHeight > 0 ? Math.min(scrollTop / docHeight, 1) : 0;
        progressFill.style.strokeDashoffset = circumference * (1 - progress);
      });
    }, { passive: true });
  }
})();
