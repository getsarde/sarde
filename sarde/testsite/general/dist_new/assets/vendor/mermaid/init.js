(function () {
  function run() {
    if (typeof window.mermaid === "undefined") return;
    var dark = document.documentElement.getAttribute("data-theme") === "dark";
    window.mermaid.initialize({
      startOnLoad: true,
      theme: dark ? "dark" : "default",
      securityLevel: "strict"
    });
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", run);
  } else {
    run();
  }
})();
