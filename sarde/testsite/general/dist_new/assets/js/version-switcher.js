document.addEventListener('click', (e) => {
  const trigger = e.target.closest('[data-sarde-version-switcher-trigger]');
  if (trigger) {
    const menu = trigger.nextElementSibling;
    const expanded = trigger.getAttribute('aria-expanded') === 'true';
    trigger.setAttribute('aria-expanded', !expanded);
    menu.hidden = expanded;
    return;
  }
  document.querySelectorAll('[data-sarde-version-switcher-trigger][aria-expanded="true"]').forEach(t => {
    t.setAttribute('aria-expanded', 'false');
    t.nextElementSibling.hidden = true;
  });
});

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    document.querySelectorAll('[data-sarde-version-switcher-trigger][aria-expanded="true"]').forEach(t => {
      t.setAttribute('aria-expanded', 'false');
      t.nextElementSibling.hidden = true;
      t.focus();
    });
  }
});
