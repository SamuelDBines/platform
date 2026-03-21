(function () {
  function escapeHtml(s) {
    var div = document.createElement("div");
    div.textContent = s;
    return div.innerHTML;
  }

  var script = document.currentScript;
  if (!script) return;

  var key = script.getAttribute("data-key") || "";
  var theme = script.getAttribute("data-theme") || "default";
  var accent = script.getAttribute("data-accent") || "#4f46e5";

  var apiBase = "";
  try {
    apiBase = new URL(script.src).origin;
  } catch (e) {
    apiBase = window.location.origin;
  }
  var submitUrl = apiBase + "/v1/submit";

  var container = document.querySelector("[data-relayemail-form]");
  if (!container) {
    container = document.createElement("div");
    container.setAttribute("data-relayemail-form", "");
    script.parentNode.insertBefore(container, script.nextSibling);
  }

  var formId = "relayemail-" + Math.random().toString(36).slice(2);
  var accentSafe = accent.replace(/[^a-fA-F0-9#]/g, "");
  if (!accentSafe) accentSafe = "#4f46e5";

  var themeClass = "relayemail-theme-" + (theme === "minimal" || theme === "dark" ? theme : "default");

  var html =
    '<form class="relayemail-form ' +
    themeClass +
    '" data-relayemail-public-key="' +
    escapeHtml(key) +
    '" id="' +
    formId +
    '" style="--relayemail-accent:' +
    accentSafe +
    '">' +
    "<style>" +
    ".relayemail-form{max-width:420px;padding:1.5rem;border-radius:0.75rem;border:1px solid #e2e8f0;" +
    "background:#fff;font-family:system-ui,-apple-system,sans-serif;box-shadow:0 12px 30px rgba(15,23,42,0.08)}" +
    ".relayemail-form h2{margin:0 0 0.75rem;font-size:1.1rem;font-weight:600;color:#0f172a}" +
    ".relayemail-form p{margin:0 0 1rem;font-size:0.9rem;color:#64748b}" +
    ".relayemail-form .relayemail-field{display:flex;flex-direction:column;gap:0.25rem;margin-bottom:0.9rem;font-size:0.9rem}" +
    ".relayemail-form label{font-weight:500;color:#0f172a}" +
    ".relayemail-form input,.relayemail-form textarea{padding:0.6rem 0.75rem;border-radius:0.5rem;border:1px solid #cbd5e1;" +
    "font:inherit;color:#0f172a;background:#f8fafc;outline:none;transition:border-color .12s,box-shadow .12s,background-color .12s}" +
    ".relayemail-form input:focus,.relayemail-form textarea:focus{border-color:var(--relayemail-accent,#4f46e5);background:#fff;" +
    "box-shadow:0 0 0 1px color-mix(in srgb,var(--relayemail-accent,#4f46e5) 40%,transparent)}" +
    ".relayemail-form textarea{min-height:120px;resize:vertical}" +
    ".relayemail-form button[type=submit]{display:inline-flex;align-items:center;justify-content:center;margin-top:0.25rem;padding:0.6rem 1.1rem;" +
    "border-radius:999px;border:none;background:var(--relayemail-accent,#4f46e5);color:#fff;font-size:0.9rem;font-weight:600;cursor:pointer;" +
    "transition:transform .08s,box-shadow .08s,filter .08s;box-shadow:0 10px 20px color-mix(in srgb,var(--relayemail-accent,#4f46e5) 30%,transparent)}" +
    ".relayemail-form button[type=submit]:hover{filter:brightness(1.05);transform:translateY(-1px);" +
    "box-shadow:0 14px 24px color-mix(in srgb,var(--relayemail-accent,#4f46e5) 35%,transparent)}" +
    ".relayemail-form button[type=submit]:disabled{opacity:0.7;cursor:default;box-shadow:none;transform:none}" +
    ".relayemail-form .relayemail-status{margin-top:0.6rem;font-size:0.85rem}" +
    ".relayemail-form .relayemail-status--success{color:#0f766e}" +
    ".relayemail-form .relayemail-status--error{color:#b91c1c}" +
    ".relayemail-form.relayemail-theme-minimal{box-shadow:0 1px 3px rgba(0,0,0,0.08);border-color:#e2e8f0}" +
    ".relayemail-form.relayemail-theme-dark{background:#1e293b;border-color:#334155}" +
    ".relayemail-form.relayemail-theme-dark h2,.relayemail-form.relayemail-theme-dark label{color:#f1f5f9}" +
    ".relayemail-form.relayemail-theme-dark p{color:#94a3b8}" +
    ".relayemail-form.relayemail-theme-dark input,.relayemail-form.relayemail-theme-dark textarea{" +
    "background:#0f172a;border-color:#334155;color:#f1f5f9}" +
    ".relayemail-form.relayemail-theme-dark input:focus,.relayemail-form.relayemail-theme-dark textarea:focus{background:#1e293b}" +
    "</style>" +
    "<h2>Contact us</h2>" +
    "<p>Drop us a line and we'll get back to you.</p>" +
    '<div class="relayemail-field"><label for="relay-name-' +
    formId +
    '">Name</label><input id="relay-name-' +
    formId +
    '" name="name" type="text" autocomplete="name" required></div>' +
    '<div class="relayemail-field"><label for="relay-email-' +
    formId +
    '">Email</label><input id="relay-email-' +
    formId +
    '" name="email" type="email" autocomplete="email" required></div>' +
    '<div class="relayemail-field"><label for="relay-message-' +
    formId +
    '">Message</label><textarea id="relay-message-' +
    formId +
    '" name="message" rows="4" required></textarea></div>' +
    '<button type="submit">Send message</button>' +
    '<div class="relayemail-status relayemail-status--success" style="display:none">Thanks – your message has been sent.</div>' +
    '<div class="relayemail-status relayemail-status--error" style="display:none">Sorry, something went wrong. Please try again.</div>' +
    "</form>";

  container.innerHTML = html;
  var form = container.querySelector("form");
  if (!form) return;

  var successEl = form.querySelector(".relayemail-status--success");
  var errorEl = form.querySelector(".relayemail-status--error");
  var submitBtn = form.querySelector('button[type="submit"]');

  function setStatus(kind, message) {
    if (successEl) successEl.style.display = "none";
    if (errorEl) errorEl.style.display = "none";
    var el = kind === "success" ? successEl : errorEl;
    if (el) {
      if (message) el.textContent = message;
      el.style.display = "block";
    }
  }

  form.addEventListener("submit", function (e) {
    e.preventDefault();
    if (!submitBtn) return;

    var formData = new FormData(form);
    var name = String(formData.get("name") || "").trim();
    var email = String(formData.get("email") || "").trim();
    var message = String(formData.get("message") || "").trim();
    var formKey = form.getAttribute("data-relayemail-public-key") || key;

    if (!name || !email || !message) {
      setStatus("error", "Please fill out all fields.");
      return;
    }

    submitBtn.disabled = true;
    setStatus("success", "");
    setStatus("error", "");

    fetch(submitUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        publicKey: formKey,
        name: name,
        email: email,
        message: message,
      }),
    })
      .then(function (res) {
        return res.json().catch(function () { return {}; }).then(function (data) {
          if (!res.ok) {
            var msg = "Sorry, something went wrong. Please try again.";
            if (data.error === "invalid_key") {
              msg = "Invalid form key. Check your snippet in the dashboard.";
            } else if (data.error === "domain_not_allowed") {
              msg = "This domain is not allowed. Add your domain in the project settings.";
            }
            setStatus("error", msg);
            return;
          }
          form.reset();
          setStatus("success", "Thanks – your message has been sent.");
        });
      })
      .catch(function () {
        setStatus("error", "Sorry, something went wrong. Please try again.");
      })
      .finally(function () {
        submitBtn.disabled = false;
      });
  });
})();
