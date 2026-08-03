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

// Close when keyboard focus leaves the trigger/menu entirely (Tab-away),
// so aria-expanded never lies while focus is elsewhere on the page.
document.addEventListener('focusout', (e) => {
  document.querySelectorAll('[data-sarde-version-switcher-trigger][aria-expanded="true"]').forEach(t => {
    const menu = t.nextElementSibling;
    const next = e.relatedTarget;
    if (next && (t.contains(next) || (menu && menu.contains(next)))) return;
    t.setAttribute('aria-expanded', 'false');
    if (menu) menu.hidden = true;
  });
});
