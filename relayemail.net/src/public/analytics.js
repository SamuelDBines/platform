(function () {
  var script = document.currentScript;
  if (!script) return;

  var key = script.getAttribute("data-key") || "";
  if (!key) return;

  var privacy = script.getAttribute("data-privacy");
  if (privacy !== null && privacy !== "" && navigator.doNotTrack === "1") {
    return;
  }

  var apiBase = "";
  try {
    apiBase = new URL(script.src).origin;
  } catch (e) {
    apiBase = window.location.origin;
  }
  var analyticsUrl = apiBase + "/v1/analytics";

  function send(event) {
    var payload = {
      publicKey: key,
      event: event,
      origin: window.location.origin || "",
      referrer: document.referrer || ""
    };
    var body = JSON.stringify(payload);
    if (navigator.sendBeacon) {
      var blob = new Blob([body], { type: "application/json" });
      navigator.sendBeacon(analyticsUrl, blob);
    } else {
      fetch(analyticsUrl, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: body
      }).catch(function () {});
    }
  }

  send("page_view");

  var form = document.querySelector("[data-relayemail-form]");
  if (form) {
    if ("IntersectionObserver" in window) {
      var observer = new IntersectionObserver(
        function (entries) {
          entries.forEach(function (entry) {
            if (entry.isIntersecting) {
              send("form_impression");
              observer.disconnect();
            }
          });
        },
        { threshold: 0.1 }
      );
      observer.observe(form);
    } else {
      send("form_impression");
    }
  }
})();
