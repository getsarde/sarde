(function () {
  var tocLinks = document.querySelectorAll('.sarde-toc-nav a');
  if (tocLinks.length === 0) return;

  var headingIds = [];
  tocLinks.forEach(function (link) {
    var id = link.getAttribute('href');
    if (id) headingIds.push(id.replace('#', ''));
  });

  var observer = new IntersectionObserver(function (entries) {
    entries.forEach(function (entry) {
      if (entry.isIntersecting) {
        tocLinks.forEach(function (link) { link.classList.remove('is-active'); });
        var active = document.querySelector('.sarde-toc-nav a[href="#' + entry.target.id + '"]');
        if (active) active.classList.add('is-active');
      }
    });
  }, { rootMargin: '-80px 0px -60% 0px' });

  headingIds.forEach(function (id) {
    var el = document.getElementById(id);
    if (el) observer.observe(el);
  });

  tocLinks.forEach(function (link) {
    link.addEventListener('click', function (e) {
      e.preventDefault();
      var id = this.getAttribute('href').replace('#', '');
      var el = document.getElementById(id);
      if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
      history.replaceState(null, '', '#' + id);
    });
  });
})();
