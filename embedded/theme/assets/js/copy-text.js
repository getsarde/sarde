document.addEventListener('click', function (e) {
    var btn = e.target.closest('.sarde-copy-text__btn');
    if (!btn) return;

    var widget = btn.closest('.sarde-copy-text');
    if (!widget) return;

    var text = widget.getAttribute('data-copy-text');
    if (!text) return;

    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(function () {
            widget.classList.add('is-copied');
            setTimeout(function () {
                widget.classList.remove('is-copied');
            }, 1500);
        });
    }
});
