(function () {
  var selectors = '.sarde-content-panel h1[id], .sarde-content-panel h2[id], .sarde-content-panel h3[id], .sarde-content-panel h4[id]';
  var headings = document.querySelectorAll(selectors);
  var svgNS = 'http://www.w3.org/2000/svg';

  headings.forEach(function (h) {
    var a = document.createElement('a');
    a.className = 'sarde-heading-anchor';
    a.href = '#' + h.id;
    a.setAttribute('aria-label', 'Link to section: ' + h.textContent.trim());

    var svg = document.createElementNS(svgNS, 'svg');
    svg.setAttribute('width', '16');
    svg.setAttribute('height', '16');
    svg.setAttribute('viewBox', '0 0 24 24');
    svg.setAttribute('fill', 'none');
    svg.setAttribute('stroke', 'currentColor');
    svg.setAttribute('stroke-width', '2');
    svg.setAttribute('stroke-linecap', 'round');
    svg.setAttribute('stroke-linejoin', 'round');
    svg.setAttribute('aria-hidden', 'true');

    var path1 = document.createElementNS(svgNS, 'path');
    path1.setAttribute('d', 'M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71');
    var path2 = document.createElementNS(svgNS, 'path');
    path2.setAttribute('d', 'M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71');

    svg.appendChild(path1);
    svg.appendChild(path2);
    a.appendChild(svg);
    h.appendChild(a);
  });
})();
