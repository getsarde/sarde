(function () {
  var ws, retries = 0, maxRetry = 8000, startTime = Date.now();
  var overlay = null, banner = null;

  function connect() {
    var proto = location.protocol === "https:" ? "wss:" : "ws:";
    ws = new WebSocket(proto + "//" + location.host + "/ws");

    ws.onopen = function () { retries = 0; startTime = Date.now(); };

    ws.onmessage = function (e) {
      var msg;
      try { msg = JSON.parse(e.data); } catch (_) { return; }

      if (msg.type === "reload") {
        location.reload();
      } else if (msg.type === "css") {
        var links = document.querySelectorAll('link[rel="stylesheet"]');
        links.forEach(function (l) {
          var href = l.href.replace(/[?&]_lr=\d+/, "");
          l.href = href + (href.indexOf("?") > -1 ? "&" : "?") + "_lr=" + Date.now();
        });
      } else if (msg.type === "error") {
        showOverlay(msg);
      } else if (msg.type === "warning") {
        showBanner(msg.error || "Warning");
      }
    };

    ws.onclose = function () {
      if (Date.now() - startTime > 30000 && retries > 3) {
        location.reload();
        return;
      }
      var delay = Math.min(1000 * Math.pow(2, retries), maxRetry);
      retries++;
      setTimeout(connect, delay);
    };
  }

  function showOverlay(msg) {
    removeOverlay();
    overlay = document.createElement("div");
    overlay.id = "__lr_overlay";
    overlay.style.cssText =
      "position:fixed;inset:0;z-index:99999;background:rgba(0,0,0,0.85);" +
      "display:flex;align-items:center;justify-content:center;font-family:monospace;color:#fff;";

    var box = document.createElement("div");
    box.style.cssText =
      "background:#1e1e2e;border:1px solid #f38ba8;border-radius:8px;padding:24px;max-width:720px;" +
      "width:90%;max-height:80vh;overflow:auto;";

    var header = document.createElement("div");
    header.style.cssText = "display:flex;justify-content:space-between;align-items:center;margin-bottom:16px;";

    var title = document.createElement("span");
    title.style.cssText = "color:#f38ba8;font-size:16px;font-weight:bold;";
    title.textContent = (msg.file ? msg.file : "Build") + " Error";

    var close = document.createElement("button");
    close.textContent = "\u00d7";
    close.style.cssText =
      "background:none;border:none;color:#fff;font-size:24px;cursor:pointer;padding:0 4px;";
    close.onclick = removeOverlay;
    header.appendChild(title);
    header.appendChild(close);
    box.appendChild(header);

    if (msg.file) {
      var loc = document.createElement("div");
      loc.style.cssText = "color:#89b4fa;margin-bottom:12px;font-size:13px;";
      loc.textContent = msg.file + (msg.line ? ":" + msg.line : "") + (msg.col ? ":" + msg.col : "");
      box.appendChild(loc);
    }

    if (msg.frame) {
      var pre = document.createElement("pre");
      pre.style.cssText =
        "background:#181825;border-radius:4px;padding:12px;overflow-x:auto;font-size:13px;" +
        "line-height:1.5;margin:0 0 12px;color:#cdd6f4;";
      pre.textContent = msg.frame;
      box.appendChild(pre);
    }

    if (msg.error) {
      var err = document.createElement("div");
      err.style.cssText = "color:#f38ba8;font-size:13px;margin-bottom:12px;";
      err.textContent = msg.error;
      box.appendChild(err);
    }

    var foot = document.createElement("div");
    foot.style.cssText = "color:#6c7086;font-size:12px;";
    foot.textContent = "Waiting for file changes\u2026 (press Escape to dismiss)";
    box.appendChild(foot);

    overlay.appendChild(box);
    document.body.appendChild(overlay);
  }

  function removeOverlay() {
    if (overlay && overlay.parentNode) overlay.parentNode.removeChild(overlay);
    overlay = null;
  }

  function showBanner(text) {
    removeBanner();
    banner = document.createElement("div");
    banner.id = "__lr_banner";
    banner.style.cssText =
      "position:fixed;top:0;left:0;right:0;z-index:99998;background:#f9e2af;color:#1e1e2e;" +
      "padding:8px 16px;font-family:monospace;font-size:13px;cursor:pointer;text-align:center;";
    banner.textContent = "\u26a0 " + text;
    banner.onclick = removeBanner;
    document.body.appendChild(banner);
  }

  function removeBanner() {
    if (banner && banner.parentNode) banner.parentNode.removeChild(banner);
    banner = null;
  }

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") removeOverlay();
  });

  connect();
})();
