// Announcements Plugin — client-side dismissal tracking

const STORAGE_PREFIX = 'announcement-dismissed-';

document.querySelectorAll('.announcement-banner[data-announcement-id]').forEach(banner => {
    const id = banner.dataset.announcementId;
    if (localStorage.getItem(STORAGE_PREFIX + id)) {
        banner.classList.add('dismissed');
    }
});

document.addEventListener('click', (e) => {
    const btn = e.target.closest('.announcement-dismiss');
    if (!btn) return;

    const id = btn.dataset.announcementId;
    const banner = btn.closest('.announcement-banner');

    if (banner) {
        banner.classList.add('dismissed');
    }

    if (id) {
        localStorage.setItem(STORAGE_PREFIX + id, '1');
    }
});
