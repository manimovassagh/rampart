// admin.js — External script for Rampart admin console.
// Replaces all inline onclick handlers to comply with strict CSP.

(function () {
  "use strict";

  // Handle confirm-action buttons: <button data-confirm="Are you sure?">
  document.addEventListener("click", function (e) {
    var btn = e.target.closest("[data-confirm]");
    if (btn) {
      if (!confirm(btn.getAttribute("data-confirm"))) {
        e.preventDefault();
      }
    }
  });

  // Handle auto-submit checkboxes: <input data-auto-submit="true">
  document.addEventListener("change", function (e) {
    var el = e.target.closest("[data-auto-submit]");
    if (el) {
      var form = el.closest("form");
      if (form) form.submit();
    }
  });

  // Handle dismiss buttons: <button data-dismiss>
  document.addEventListener("click", function (e) {
    var btn = e.target.closest("[data-dismiss]");
    if (btn) {
      var target = btn.closest("[data-dismissable]");
      if (target) {
        target.remove();
      }
    }
  });

  // OIDC page: copy from sibling [data-copy] element
  document.addEventListener("click", function (e) {
    var btn = e.target.closest("[data-copy-sibling]");
    if (!btn) return;
    var text = btn.parentElement.querySelector("[data-copy]");
    if (!text) return;
    navigator.clipboard.writeText(text.textContent.trim()).then(function () {
      var icon = btn.querySelector(".copy-icon");
      var check = btn.querySelector(".check-icon");
      if (icon) icon.classList.add("hidden");
      if (check) check.classList.remove("hidden");
      setTimeout(function () {
        if (icon) icon.classList.remove("hidden");
        if (check) check.classList.add("hidden");
      }, 1500);
    });
  });

  // OIDC page: copy full JSON
  document.addEventListener("click", function (e) {
    var btn = e.target.closest("[data-copy-json]");
    if (!btn) return;
    var el = document.getElementById("oidc-json-data");
    if (!el) return;
    navigator.clipboard.writeText(el.textContent).then(function () {
      var label = document.getElementById("copy-json-label");
      if (label) {
        label.textContent = "Copied!";
        setTimeout(function () {
          label.textContent = "Copy Full JSON";
        }, 1500);
      }
    });
  });
})();
