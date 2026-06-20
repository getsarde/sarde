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
        if (msg.changedAt) sessionStorage.setItem("__lr_changed_at", msg.changedAt);
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
        console.warn("[sarde]", msg.error || "Warning");
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

    // Backdrop
    overlay = document.createElement("div");
    overlay.id = "__lr_overlay";
    overlay.style.cssText =
      "position:fixed;inset:0;z-index:99999;background:rgba(0,0,0,0.66);" +
      "overflow-y:auto;padding:40px 16px 16px;" +
      "font-family:'SFMono-Regular',Consolas,'Liberation Mono',Menlo,monospace;";
    overlay.onclick = function (e) { if (e.target === overlay) removeOverlay(); };

    // Card
    var card = document.createElement("div");
    card.style.cssText =
      "background:#252525;border-radius:8px;overflow:hidden;width:92vw;max-width:960px;margin:0 auto;" +
      "max-height:85vh;overflow-y:auto;box-shadow:0 20px 60px rgba(0,0,0,0.5);" +
      "border-top:4px solid #ff5555;";

    var inner = document.createElement("div");
    inner.style.cssText = "padding:24px 28px;position:relative;";

    // Actions (top-right)
    var actions = document.createElement("div");
    actions.style.cssText = "position:absolute;top:20px;right:24px;display:flex;align-items:center;gap:8px;";

    var copySvg = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>';
    var copy = document.createElement("button");
    copy.innerHTML = copySvg + " Copy";
    copy.style.cssText =
      "background:#333;border:1px solid #555;color:#ccc;font-size:12px;font-family:inherit;" +
      "cursor:pointer;padding:5px 10px;border-radius:4px;display:flex;align-items:center;gap:5px;";
    copy.onmouseenter = function () { copy.style.background = "#444"; };
    copy.onmouseleave = function () { copy.style.background = "#333"; };
    copy.onclick = function () {
      var text = "";
      if (msg.file) text += msg.file + (msg.line ? ":" + msg.line : "") + (msg.col ? ":" + msg.col : "") + "\n";
      if (msg.error) text += msg.error + "\n";
      if (msg.frame) text += "\n" + msg.frame;
      navigator.clipboard.writeText(text.trim()).then(function () {
        copy.innerHTML = copySvg + " Copied!";
        setTimeout(function () { copy.innerHTML = copySvg + " Copy"; }, 1500);
      });
    };

    var close = document.createElement("button");
    close.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>';
    close.style.cssText =
      "background:none;border:none;color:#888;cursor:pointer;padding:4px;display:flex;";
    close.onmouseenter = function () { close.style.color = "#ccc"; };
    close.onmouseleave = function () { close.style.color = "#888"; };
    close.onclick = removeOverlay;

    actions.appendChild(copy);
    actions.appendChild(close);
    inner.appendChild(actions);

    // Error header: [Build Error] message
    var errHeader = document.createElement("div");
    errHeader.style.cssText =
      "font-size:16px;font-weight:700;line-height:1.5;margin-bottom:16px;" +
      "padding-right:140px;word-break:break-word;color:#e8e8e8;";
    var label = document.createElement("span");
    label.style.cssText = "color:#e2b340;";
    label.textContent = "[Build Error] ";
    errHeader.appendChild(label);
    if (msg.error) {
      errHeader.appendChild(document.createTextNode(msg.error));
    }
    inner.appendChild(errHeader);

    // File location
    if (msg.file) {
      var loc = document.createElement("div");
      loc.style.cssText = "color:#56b6c2;font-size:14px;margin-bottom:16px;";
      loc.textContent = msg.file + (msg.line ? ":" + msg.line : "") + (msg.col ? ":" + msg.col : "");
      inner.appendChild(loc);
    }

    // Code frame
    if (msg.frame) {
      var frameBox = document.createElement("div");
      frameBox.style.cssText =
        "background:#1a1a1a;border-radius:6px;padding:14px 0;overflow-x:auto;margin-bottom:16px;";

      var lines = msg.frame.split("\n");
      for (var i = 0; i < lines.length; i++) {
        if (!lines[i] && i === lines.length - 1) continue;
        var line = document.createElement("div");
        var isErrorLine = lines[i].indexOf("> ") === 0;
        line.style.cssText = "padding:1px 18px;font-size:13px;line-height:1.7;white-space:pre;" +
          (isErrorLine
            ? "background:rgba(255,85,85,0.1);color:#ff8888;"
            : "color:#999;");
        line.textContent = lines[i];
        frameBox.appendChild(line);
      }
      inner.appendChild(frameBox);
    }

    // Footer
    var foot = document.createElement("div");
    foot.style.cssText = "color:#666;font-size:13px;line-height:1.6;border-top:1px dashed #333;padding-top:14px;margin-top:4px;";
    foot.textContent = "Click outside or fix the code to dismiss.";
    inner.appendChild(foot);

    card.appendChild(inner);
    overlay.appendChild(card);
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

  function showReloadTiming() {
    var t = sessionStorage.getItem("__lr_changed_at");
    if (!t) return;
    sessionStorage.removeItem("__lr_changed_at");
    var ms = Date.now() - parseInt(t, 10);
    var el = document.createElement("div");
    el.style.cssText =
      "position:fixed;bottom:12px;right:12px;z-index:99997;background:#1e1e2e;color:#a6e3a1;" +
      "padding:6px 12px;border-radius:6px;font-family:monospace;font-size:13px;" +
      "opacity:1;transition:opacity 0.5s;pointer-events:none;border:1px solid #313244;";
    el.textContent = "↻ " + ms + "ms";
    document.body.appendChild(el);
    setTimeout(function () { el.style.opacity = "0"; }, 3000);
    setTimeout(function () { if (el.parentNode) el.parentNode.removeChild(el); }, 3500);
  }

  showReloadTiming();
  connect();
})();
