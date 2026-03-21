(function () {
  var script = document.currentScript;
  if (!script) return;

  var key = script.getAttribute("data-key") || "";
  if (!key) return;

  var consoleMode = script.getAttribute("data-console") || "";
  var forwardLog = consoleMode.indexOf("log") >= 0;
  var forwardWarn = consoleMode.indexOf("warn") >= 0 || consoleMode === "1" || consoleMode === "true";
  var forwardError = consoleMode.indexOf("error") >= 0 || consoleMode === "1" || consoleMode === "true";

  var apiBase = "";
  try {
    apiBase = new URL(script.src).origin;
  } catch (e) {
    apiBase = window.location.origin;
  }
  var logsUrl = apiBase + "/v1/logs";

  function send(level, message, stack, extra) {
    var payload = {
      publicKey: key,
      level: level,
      message: message || "",
      stack: stack || "",
      url: window.location.href || "",
      referrer: document.referrer || "",
      extra: extra || ""
    };
    fetch(logsUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload)
    }).catch(function () {});
  }

  window.onerror = function (message, source, lineno, colno, error) {
    var stack = error && error.stack ? error.stack : "";
    send("error", String(message), stack, source ? source + ":" + lineno + ":" + colno : "");
    return false;
  };

  window.addEventListener("unhandledrejection", function (e) {
    var msg = e.reason;
    var stack = "";
    if (msg && typeof msg === "object") {
      stack = msg.stack || "";
      msg = msg.message || String(msg);
    } else {
      msg = String(msg);
    }
    send("error", msg, stack, "");
  });

  if (forwardLog || forwardWarn || forwardError) {
    var orig = {
      log: console.log,
      warn: console.warn,
      error: console.error
    };
    function wrap(level, fn) {
      return function () {
        fn.apply(console, arguments);
        var args = Array.prototype.slice.call(arguments);
        var msg = args.map(function (a) {
          return typeof a === "object" ? JSON.stringify(a) : String(a);
        }).join(" ");
        send(level, msg, "", "");
      };
    }
    if (forwardLog) console.log = wrap("log", orig.log);
    if (forwardWarn) console.warn = wrap("warn", orig.warn);
    if (forwardError) console.error = wrap("error", orig.error);
  }
})();
