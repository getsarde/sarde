(function () {
  var modal = document.getElementById("sarde-search-modal");
  var input = document.getElementById("sarde-search-input");
  var results = document.getElementById("sarde-search-results");
  var empty = document.getElementById("sarde-search-empty");
  var triggers = document.querySelectorAll("[data-search-trigger]");
  if (!modal || !input || !results) return;

  var dbPromise = null;
  var _base = (window.__BASE_PATH__ || "/").replace(/\/$/, "");
  var selectedIndex = -1;
  var currentHits = [];

  function loadIndex() {
    if (dbPromise) return dbPromise;
    dbPromise = Promise.all([
      import(_base + "/assets/vendor/orama/orama.esm.js"),
      fetch(_base + "/search-index." + (searchLang || "en") + ".json").then(function (r) { return r.json(); })
    ]).then(function (arr) {
      var orama = arr[0], docs = arr[1];
      var db = orama.create({
        schema: { id: "string", title: "string", url: "string", content: "string", description: "string", section: "string", tags: "string[]", version: "enum", breadcrumb: "string", kind: "enum", anchor: "string" }
      });
      orama.insertMultiple(db, docs);
      return { orama: orama, db: db };
    });
    return dbPromise;
  }

  function open() {
    modal.hidden = false;
    modal.removeAttribute("hidden");
    input.focus();
    loadIndex();
  }

  function close() {
    modal.hidden = true;
    modal.setAttribute("hidden", "");
    input.value = "";
    clearChildren(results);
    if (empty) empty.hidden = true;
    selectedIndex = -1;
    currentHits = [];
  }

  triggers.forEach(function (t) {
    t.addEventListener("click", function (e) { e.preventDefault(); open(); });
  });

  document.addEventListener("keydown", function (e) {
    if ((e.ctrlKey || e.metaKey) && e.key === "k") {
      e.preventDefault();
      open();
    } else if (e.key === "Escape" && !modal.hidden) {
      close();
    }
  });

  modal.addEventListener("click", function (e) {
    if (e.target === modal) close();
  });

  var debounceId = null;
  var lastTerm = "";
  input.addEventListener("input", function () {
    clearTimeout(debounceId);
    var term = input.value.trim();
    lastTerm = term;
    debounceId = setTimeout(function () { runSearch(term); }, 120);
  });

  input.addEventListener("keydown", function (e) {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      if (currentHits.length) {
        selectedIndex = Math.min(selectedIndex + 1, currentHits.length - 1);
        updateSelection();
      }
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      if (currentHits.length) {
        selectedIndex = Math.max(selectedIndex - 1, 0);
        updateSelection();
      }
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (selectedIndex >= 0 && selectedIndex < currentHits.length) {
        var links = results.querySelectorAll(".sarde-search-result");
        if (links[selectedIndex]) {
          var href = links[selectedIndex].href;
          close();
          window.location.href = href;
        }
      }
    }
  });

  var searchVersion = modal.getAttribute("data-search-version") || "";
  var searchLang = modal.getAttribute("data-search-lang") || "";

  function runSearch(term) {
    if (!term) {
      clearChildren(results);
      if (empty) empty.hidden = true;
      selectedIndex = -1;
      currentHits = [];
      return;
    }
    loadIndex().then(function (ctx) {
      var opts = { term: term, properties: ["title", "content", "description", "section", "tags"], limit: 30, tolerance: 1 };
      if (searchVersion) {
        opts.where = { version: { eq: searchVersion } };
      }
      return ctx.orama.search(ctx.db, opts);
    }).then(function (res) {
      var hits = res && res.hits ? res.hits : [];
      currentHits = hits;
      selectedIndex = -1;
      render(hits, term);
    }).catch(function () {
      clearChildren(results);
      currentHits = [];
      selectedIndex = -1;
      if (empty) empty.hidden = false;
    });
  }

  function escapeHTML(s) {
    var div = document.createElement("div");
    div.textContent = s;
    return div.innerHTML;
  }

  function highlight(text, term) {
    if (!term) return escapeHTML(text);
    var escaped = term.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    var re = new RegExp("(" + escaped + ")", "gi");
    return escapeHTML(text).replace(re, "<mark>$1</mark>");
  }

  function snippet(content, term, maxLen) {
    if (!content) return "";
    maxLen = maxLen || 120;
    var lower = content.toLowerCase();
    var tLower = term.toLowerCase();
    var idx = lower.indexOf(tLower);
    var text;
    if (idx === -1) {
      text = content.length > maxLen ? content.slice(0, maxLen) + "..." : content;
    } else {
      var start = Math.max(0, idx - Math.floor(maxLen / 2));
      var end = Math.min(content.length, start + maxLen);
      text = (start > 0 ? "..." : "") + content.slice(start, end) + (end < content.length ? "..." : "");
    }
    return highlight(text, term);
  }

  var PAGE_ICON = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>';
  var HASH_ICON = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" y1="9" x2="20" y2="9"/><line x1="4" y1="15" x2="20" y2="15"/><line x1="10" y1="3" x2="8" y2="21"/><line x1="16" y1="3" x2="14" y2="21"/></svg>';
  var FOLDER_ICON = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>';
  var ARROW_ICON = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>';

  function render(hits, term) {
    clearChildren(results);
    if (!hits.length) {
      if (empty) empty.hidden = false;
      return;
    }
    if (empty) empty.hidden = true;

    var header = document.createElement("li");
    header.className = "sarde-search-results-header";
    header.textContent = hits.length + " result" + (hits.length !== 1 ? "s" : "") + " for ‘" + term + "’";
    results.appendChild(header);

    var groups = {};
    var groupOrder = [];
    hits.forEach(function (h) {
      var d = h.document || h;
      var groupKey = d.section || "other";
      if (!groups[groupKey]) {
        groups[groupKey] = [];
        groupOrder.push(groupKey);
      }
      groups[groupKey].push(d);
    });

    groupOrder.forEach(function (key) {
      var groupHeader = document.createElement("li");
      groupHeader.className = "sarde-search-group-header";
      groupHeader.innerHTML = '<span class="sarde-search-group-icon">' + FOLDER_ICON + "</span>" + escapeHTML(key);
      results.appendChild(groupHeader);

      groups[key].forEach(function (d) {
        var li = document.createElement("li");
        var a = document.createElement("a");
        a.className = "sarde-search-result";
        a.href = d.url || "#";

        var icon = document.createElement("span");
        icon.className = "sarde-search-result-icon";
        icon.innerHTML = d.kind === "heading" ? HASH_ICON : PAGE_ICON;
        a.appendChild(icon);

        var contentDiv = document.createElement("div");
        contentDiv.className = "sarde-search-result-content";

        var title = document.createElement("div");
        title.className = "sarde-search-result-title";
        title.innerHTML = highlight(d.title || d.url || "", term);
        contentDiv.appendChild(title);

        var bc = d.breadcrumb || d.section || "";
        if (bc) {
          var breadcrumb = document.createElement("div");
          breadcrumb.className = "sarde-search-result-breadcrumb";
          breadcrumb.textContent = bc;
          contentDiv.appendChild(breadcrumb);
        }

        if (d.content || d.description) {
          var exc = document.createElement("div");
          exc.className = "sarde-search-result-excerpt";
          exc.innerHTML = d.content ? snippet(d.content, term) : highlight(d.description, term);
          contentDiv.appendChild(exc);
        }

        a.appendChild(contentDiv);

        var arrow = document.createElement("span");
        arrow.className = "sarde-search-result-arrow";
        arrow.innerHTML = ARROW_ICON;
        a.appendChild(arrow);

        li.appendChild(a);
        results.appendChild(li);
      });
    });
  }

  function updateSelection() {
    var links = results.querySelectorAll(".sarde-search-result");
    links.forEach(function (el, i) {
      if (i === selectedIndex) {
        el.classList.add("is-selected");
        el.scrollIntoView({ block: "nearest" });
      } else {
        el.classList.remove("is-selected");
      }
    });
  }

  function clearChildren(node) {
    while (node.firstChild) node.removeChild(node.firstChild);
  }
})();
