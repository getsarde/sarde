document.addEventListener('toggle', function (e) {
    var details = e.target;
    if (details.tagName !== 'DETAILS' || !details.open) return;

    var group = details.closest('.sarde-details-group');
    if (!group || group.hasAttribute('data-independent')) return;

    var siblings = group.querySelectorAll(':scope > details');
    for (var i = 0; i < siblings.length; i++) {
        if (siblings[i] !== details && siblings[i].open) {
            siblings[i].open = false;
        }
    }
}, true);
