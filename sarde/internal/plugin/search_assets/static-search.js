(function () {
  var modal = document.getElementById("sarde-search-modal");
  var input = document.getElementById("sarde-search-input");
  var results = document.getElementById("sarde-search-results");
  var empty = document.getElementById("sarde-search-empty");
  var emptyTips = document.getElementById("sarde-search-empty-tips");
  var initial = document.getElementById("sarde-search-initial");
  var filtersEl = document.getElementById("sarde-search-filters");
  var triggers = document.querySelectorAll("[data-search-trigger]");
  if (!modal || !input || !results) return;

  var dbPromise = null;
  var _base = (window.__BASE_PATH__ || "/").replace(/\/$/, "");
  var selectedIndex = -1;
  var currentHits = [];
  var currentTotal = 0;
  var currentOffset = 0;
  var isLoading = false;
  var activeSection = "";
  var availableSections = [];
  var filtersRendered = false;
  var PAGE_SIZE = 30;
  var RECENT_KEY = "sarde-recent-searches";
  var MAX_RECENTS = 5;

  var SEARCH_ICON = '<svg viewBox="0 0 24 24"><g fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21l-4.3-4.3"/></g></svg>';
  var HISTORY_ICON = '<svg viewBox="0 0 24 24"><g fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2"><path d="M3 12a9 9 0 1 0 9-9a9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5m4-1v5l4 2"/></g></svg>';
  var PAGE_ICON = '<svg viewBox="0 0 24 24"><g fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2"><path d="M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z"/><path d="M14 2v5a1 1 0 0 0 1 1h5"/></g></svg>';
  var HASH_ICON = '<svg viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 9h16M4 15h16M10 3L8 21m8-18l-2 18"/></svg>';
  var FOLDER_ICON = '<svg viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/></svg>';
  var ARROW_ICON = '<svg viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m9 18l6-6l-6-6"/></svg>';
  var CHEVRON_SEP = '<svg class="sarde-breadcrumb-sep" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m9 18l6-6l-6-6"/></svg>';

  // --- Recent searches (localStorage) ---

  function getRecentSearches() {
    try {
      var data = JSON.parse(localStorage.getItem(RECENT_KEY));
      return Array.isArray(data) ? data.slice(0, MAX_RECENTS) : [];
    } catch (e) { return []; }
  }

  function saveRecentSearch(term) {
    try {
      var recents = getRecentSearches().filter(function (r) { return r !== term; });
      recents.unshift(term);
      localStorage.setItem(RECENT_KEY, JSON.stringify(recents.slice(0, MAX_RECENTS)));
    } catch (e) {}
  }

  function clearRecentSearches() {
    try { localStorage.removeItem(RECENT_KEY); } catch (e) {}
    renderInitialState();
  }

  function renderInitialState() {
    if (!initial) return;
    clearChildren(initial);
    var recents = getRecentSearches();
    if (recents.length) {
      var header = document.createElement("div");
      header.className = "sarde-search-recent-header";
      var label = document.createElement("span");
      label.textContent = "Recent";
      header.appendChild(label);
      var clearBtn = document.createElement("button");
      clearBtn.type = "button";
      clearBtn.className = "sarde-search-recent-clear";
      clearBtn.textContent = "Clear";
      clearBtn.addEventListener("click", function () { clearRecentSearches(); });
      header.appendChild(clearBtn);
      initial.appendChild(header);

      recents.forEach(function (term) {
        var btn = document.createElement("button");
        btn.type = "button";
        btn.className = "sarde-search-recent-item";
        btn.innerHTML = HISTORY_ICON + "<span>" + escapeHTML(term) + "</span>";
        btn.addEventListener("click", function () {
          input.value = term;
          lastTerm = term;
          runSearch(term);
        });
        initial.appendChild(btn);
      });
    } else {
      var fallback = document.createElement("div");
      fallback.className = "sarde-search-initial-fallback";
      fallback.innerHTML = SEARCH_ICON + "<p>Type to start searching</p>";
      initial.appendChild(fallback);
    }
  }

  // --- Index loading ---

  function loadIndex() {
    if (dbPromise) return dbPromise;
    dbPromise = Promise.all([
      import(_base + "/assets/vendor/orama/orama.esm.js"),
      fetch(_base + "/search-index." + (searchLang || "en") + ".json").then(function (r) { return r.json(); })
    ]).then(function (arr) {
      var orama = arr[0], docs = arr[1];
      var db = orama.create({
        schema: { id: "string", title: "string", url: "string", content: "string", description: "string", section: "enum", tags: "string[]", version: "enum", breadcrumb: "string", kind: "enum", anchor: "string" }
      });
      orama.insertMultiple(db, docs);

      var sectionSet = {};
      docs.forEach(function (d) { if (d.section) sectionSet[d.section] = true; });
      availableSections = Object.keys(sectionSet).sort();
      renderFilterChips();

      return { orama: orama, db: db };
    });
    return dbPromise;
  }

  // --- Section filter chips ---

  function renderFilterChips() {
    if (!filtersEl || filtersRendered || !availableSections.length) return;
    filtersRendered = true;
    clearChildren(filtersEl);

    var allChip = document.createElement("button");
    allChip.type = "button";
    allChip.className = "sarde-search-filter-chip is-active";
    allChip.setAttribute("data-section", "");
    allChip.textContent = "All";
    allChip.addEventListener("click", function () { setActiveSection(""); });
    filtersEl.appendChild(allChip);

    availableSections.forEach(function (sec) {
      var chip = document.createElement("button");
      chip.type = "button";
      chip.className = "sarde-search-filter-chip";
      chip.setAttribute("data-section", sec);
      chip.textContent = sec.charAt(0).toUpperCase() + sec.slice(1);
      chip.addEventListener("click", function () { setActiveSection(sec); });
      filtersEl.appendChild(chip);
    });

    filtersEl.hidden = false;
  }

  function setActiveSection(section) {
    activeSection = section;
    if (filtersEl) {
      filtersEl.querySelectorAll(".sarde-search-filter-chip").forEach(function (chip) {
        chip.classList.toggle("is-active", chip.getAttribute("data-section") === section);
      });
    }
    var term = input.value.trim();
    if (term) {
      lastTerm = term;
      runSearch(term);
    }
  }

  // --- Open / Close ---

  function open() {
    modal.hidden = false;
    modal.removeAttribute("hidden");
    input.focus();
    renderInitialState();
    if (initial) initial.hidden = false;
    results.hidden = true;
    hideEmpty();
    loadIndex();
  }

  function close() {
    modal.hidden = true;
    modal.setAttribute("hidden", "");
    input.value = "";
    clearChildren(results);
    results.hidden = true;
    hideEmpty();
    if (initial) initial.hidden = false;
    selectedIndex = -1;
    currentHits = [];
    currentTotal = 0;
    currentOffset = 0;
    isLoading = false;
    activeSection = "";
    if (filtersEl) {
      filtersEl.querySelectorAll(".sarde-search-filter-chip").forEach(function (chip) {
        chip.classList.toggle("is-active", chip.getAttribute("data-section") === "");
      });
    }
  }

  // --- Event listeners ---

  triggers.forEach(function (t) {
    t.addEventListener("click", function (e) { e.preventDefault(); open(); });
  });

  document.querySelectorAll("[data-search-close]").forEach(function (el) {
    el.addEventListener("click", function (e) { e.preventDefault(); close(); });
  });

  document.addEventListener("keydown", function (e) {
    if ((e.ctrlKey || e.metaKey) && e.key === "k") {
      e.preventDefault();
      open();
    } else if (e.key === "Escape" && !modal.hidden) {
      close();
    }
  });

  results.addEventListener("scroll", function () {
    if (isLoading || currentOffset >= currentTotal) return;
    var scrollBottom = results.scrollHeight - results.scrollTop - results.clientHeight;
    if (scrollBottom < 200) {
      loadMore();
    }
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
        if (selectedIndex >= currentHits.length - 3 && currentOffset < currentTotal && !isLoading) {
          loadMore();
        }
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

  // --- Search ---

  function buildSearchOpts(term, offset) {
    var opts = { term: term, properties: ["title", "content", "description", "tags"], limit: PAGE_SIZE, offset: offset, tolerance: 1 };
    var where = {};
    if (searchVersion) where.version = { eq: searchVersion };
    if (activeSection) where.section = { eq: activeSection };
    if (Object.keys(where).length) opts.where = where;
    return opts;
  }

  function showMinLengthMessage() {
    if (!initial) return;
    clearChildren(initial);
    var msg = document.createElement("div");
    msg.className = "sarde-search-initial-fallback";
    msg.innerHTML = SEARCH_ICON + "<p>Type at least 2 characters</p>";
    initial.appendChild(msg);
    initial.hidden = false;
  }

  function runSearch(term) {
    if (!term) {
      clearChildren(results);
      results.hidden = true;
      hideEmpty();
      renderInitialState();
      if (initial) initial.hidden = false;
      selectedIndex = -1;
      currentHits = [];
      currentTotal = 0;
      currentOffset = 0;
      return;
    }
    if (term.length < 2) {
      clearChildren(results);
      results.hidden = true;
      hideEmpty();
      showMinLengthMessage();
      selectedIndex = -1;
      currentHits = [];
      currentTotal = 0;
      currentOffset = 0;
      return;
    }
    if (initial) initial.hidden = true;
    currentOffset = 0;
    clearChildren(results);
    results.hidden = false;
    showSpinner(results);
    loadIndex().then(function (ctx) {
      return ctx.orama.search(ctx.db, buildSearchOpts(term, 0));
    }).then(function (res) {
      var hits = res && res.hits ? res.hits : [];
      currentTotal = res && res.count ? res.count : hits.length;
      currentHits = hits;
      currentOffset = hits.length;
      selectedIndex = -1;
      if (hits.length) saveRecentSearch(term);
      render(currentHits, term, currentTotal);
    }).catch(function () {
      clearChildren(results);
      currentHits = [];
      currentTotal = 0;
      currentOffset = 0;
      selectedIndex = -1;
      showEmpty();
    });
  }

  function loadMore() {
    var term = lastTerm;
    if (!term || currentOffset >= currentTotal || isLoading) return;
    isLoading = true;
    showSpinner(results);
    loadIndex().then(function (ctx) {
      return ctx.orama.search(ctx.db, buildSearchOpts(term, currentOffset));
    }).then(function (res) {
      removeSpinner();
      var hits = res && res.hits ? res.hits : [];
      currentHits = currentHits.concat(hits);
      currentOffset += hits.length;
      appendResults(hits, term);
      isLoading = false;
    }).catch(function () {
      removeSpinner();
      isLoading = false;
    });
  }

  // --- Empty state ---

  function cleanupEmptyAction() {
    if (!empty) return;
    var existing = empty.querySelector(".sarde-search-empty-action");
    if (existing) existing.remove();
  }

  function showEmpty() {
    if (!empty) return;
    results.hidden = true;
    cleanupEmptyAction();
    if (activeSection && emptyTips) {
      var action = document.createElement("button");
      action.type = "button";
      action.className = "sarde-search-empty-action";
      action.textContent = "Try searching in all sections";
      action.addEventListener("click", function () { setActiveSection(""); });
      emptyTips.parentNode.insertBefore(action, emptyTips.nextSibling);
    }
    empty.hidden = false;
  }

  function hideEmpty() {
    if (!empty) return;
    cleanupEmptyAction();
    empty.hidden = true;
  }

  // --- Rendering ---

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

  function render(hits, term, totalCount) {
    clearChildren(results);
    if (!hits.length) {
      results.hidden = true;
      showEmpty();
      return;
    }
    hideEmpty();
    results.hidden = false;

    var header = document.createElement("li");
    header.className = "sarde-search-results-header";
    header.textContent = totalCount + " result" + (totalCount !== 1 ? "s" : "") + " for '" + term + "'";
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
      groupHeader.setAttribute("data-search-group", key);
      groupHeader.innerHTML = '<span class="sarde-search-group-icon">' + FOLDER_ICON + "</span>" + escapeHTML(key);
      results.appendChild(groupHeader);

      groups[key].forEach(function (d) {
        results.appendChild(buildResultItem(d, term));
      });
    });
  }

  function buildResultItem(d, term) {
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
      var parts = bc.split(" > ");
      var bcHTML = parts.map(function (p) { return highlight(p, term); }).join(CHEVRON_SEP);
      breadcrumb.innerHTML = bcHTML;
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
    return li;
  }

  function appendResults(hits, term) {
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
      var existing = null;
      results.querySelectorAll(".sarde-search-group-header").forEach(function (el) {
        if (el.getAttribute("data-search-group") === key) existing = el;
      });
      if (!existing) {
        var groupHeader = document.createElement("li");
        groupHeader.className = "sarde-search-group-header";
        groupHeader.setAttribute("data-search-group", key);
        groupHeader.innerHTML = '<span class="sarde-search-group-icon">' + FOLDER_ICON + "</span>" + escapeHTML(key);
        results.appendChild(groupHeader);
      }
      groups[key].forEach(function (d) {
        results.appendChild(buildResultItem(d, term));
      });
    });
  }

  // --- Utilities ---

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

  function showSpinner(container) {
    var li = document.createElement("li");
    li.className = "sarde-search-loading";
    li.innerHTML = '<div class="sarde-search-spinner"></div>';
    container.appendChild(li);
    return li;
  }

  function removeSpinner() {
    var el = results.querySelector(".sarde-search-loading");
    if (el) el.remove();
  }

  function clearChildren(node) {
    while (node.firstChild) node.removeChild(node.firstChild);
  }
})();
