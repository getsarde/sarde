(function () {
  var modal = document.getElementById("sarde-search-modal");
  var input = document.getElementById("sarde-search-input");
  var results = document.getElementById("sarde-search-results");
  var empty = document.getElementById("sarde-search-empty");
  var triggers = document.querySelectorAll("[data-search-trigger]");
  if (!modal || !input || !results) return;

  var dbPromise = null;
  var _base = (window.__BASE_PATH__ || "/").replace(/\/$/, "");
  function loadIndex() {
    if (dbPromise) return dbPromise;
    dbPromise = Promise.all([
      import(_base + "/assets/vendor/orama/orama.esm.js"),
      fetch(_base + "/search-index.json").then(function (r) { return r.json(); })
    ]).then(function (arr) {
      var orama = arr[0], docs = arr[1];
      var db = orama.create({
        schema: { id: "string", title: "string", url: "string", content: "string", description: "string", section: "string", tags: "string[]", version: "enum", lang: "enum" }
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
  input.addEventListener("input", function () {
    clearTimeout(debounceId);
    var term = input.value.trim();
    debounceId = setTimeout(function () { runSearch(term); }, 120);
  });

  var searchVersion = modal.getAttribute("data-search-version") || "";
  var searchLang = modal.getAttribute("data-search-lang") || "";

  function runSearch(term) {
    if (!term) {
      clearChildren(results);
      if (empty) empty.hidden = true;
      return;
    }
    loadIndex().then(function (ctx) {
      var opts = { term: term, properties: ["title", "content", "description", "section", "tags"], limit: 20 };
      if (searchVersion) {
        opts.where = { version: { eq: searchVersion } };
      }
      if (searchLang) {
        opts.where = opts.where || {};
        opts.where.lang = { eq: searchLang };
      }
      return ctx.orama.search(ctx.db, opts);
    }).then(function (res) {
      render(res && res.hits ? res.hits : []);
    }).catch(function () {
      clearChildren(results);
      if (empty) empty.hidden = false;
    });
  }

  function render(hits) {
    clearChildren(results);
    if (!hits.length) {
      if (empty) empty.hidden = false;
      return;
    }
    if (empty) empty.hidden = true;
    hits.forEach(function (h) {
      var d = h.document || h;
      var li = document.createElement("li");
      var a = document.createElement("a");
      a.className = "sarde-search-result";
      a.href = d.url || "#";
      if (d.section) {
        var sec = document.createElement("div");
        sec.className = "sarde-search-result-breadcrumb";
        sec.textContent = d.section;
        a.appendChild(sec);
      }
      var title = document.createElement("div");
      title.className = "sarde-search-result-title";
      title.textContent = d.title || d.url || "";
      a.appendChild(title);
      if (d.description) {
        var p = document.createElement("div");
        p.className = "sarde-search-result-excerpt";
        p.textContent = d.description;
        a.appendChild(p);
      }
      li.appendChild(a);
      results.appendChild(li);
    });
  }

  function clearChildren(node) {
    while (node.firstChild) node.removeChild(node.firstChild);
  }
})();
