(function () {
  // Injected by `sarde dev --debug-client`. Logs what the page does to the
  // browser console and, through the dev server, to the terminal, to
  // diagnose a page that stops responding: link clicks and whether a script
  // cancelled them, bfcache restores, errors, long tasks, inert and dialog
  // state, and live reload messages.
  var endpoint = "/__sarde/client-log";

  function send(kind, detail) {
    detail = detail || "";
    try { console.debug("[sarde debug] " + kind + " " + detail); } catch (_) {}
    var body = JSON.stringify({ kind: kind, path: location.pathname, detail: detail });
    try {
      if (!(navigator.sendBeacon && navigator.sendBeacon(endpoint, body))) {
        fetch(endpoint, { method: "POST", body: body, keepalive: true });
      }
    } catch (_) {}
  }

  function state() {
    var parts = [
      "inert=" + document.querySelectorAll("[inert]").length,
      "open-dialogs=" + document.querySelectorAll("dialog[open]").length
    ];
    if (document.querySelector("#sarde-search-modal.open")) parts.push("search-modal-open");
    return parts.join(" ");
  }

  // A bubbling listener on window runs after the page's own click handlers,
  // so defaultPrevented tells whether one of them cancelled the navigation.
  window.addEventListener("click", function (e) {
    var a = e.target && e.target.closest ? e.target.closest("a[href]") : null;
    if (!a) return;
    send("link-click", a.getAttribute("href") + " defaultPrevented=" + e.defaultPrevented + " " + state());
  });
  window.addEventListener("pageshow", function (e) { send("pageshow", "persisted=" + e.persisted + " " + state()); });
  window.addEventListener("pagehide", function (e) { send("pagehide", "persisted=" + e.persisted); });
  window.addEventListener("error", function (e) {
    send("error", (e.message || "") + " at " + (e.filename || "") + ":" + (e.lineno || 0));
  });
  window.addEventListener("unhandledrejection", function (e) { send("unhandled-rejection", String(e.reason)); });
  window.addEventListener("sarde:livereload", function (e) { send("livereload", e.detail && e.detail.type); });

  if (window.PerformanceObserver) {
    try {
      new PerformanceObserver(function (list) {
        list.getEntries().forEach(function (entry) {
          send("long-task", Math.round(entry.duration) + "ms");
        });
      }).observe({ type: "longtask", buffered: true });
    } catch (_) {}
  }
})();
