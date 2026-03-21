# RelayEmail API Quickstart

## Overview

RelayEmail exposes a REST API for managing sites, domains, and receiving form submissions.

- **Base URL**: `https://your-relayemail-host.com/v1`
- **Auth**: Firebase ID token in `Authorization: Bearer <token>` header (or `dev:uid:email` for local dev)

## 1. Create a site

```bash
curl -X POST https://your-host/v1/sites \
  -H "Authorization: Bearer YOUR_FIREBASE_ID_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"My Website"}'
```

Response includes `id` and `publicKey`. Save the `publicKey` for form submissions.

## 2. Add allowed domains

Submissions are only accepted from allowed domains (checked via `Origin` header).

```bash
curl -X POST https://your-host/v1/sites/SITE_ID/domains \
  -H "Authorization: Bearer YOUR_FIREBASE_ID_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"domain":"example.com"}'
```

Add all domains that will host your form (e.g. `example.com`, `www.example.com`).

## 3. Embed the form

### Option A: Embed script (hosted by RelayEmail)

Load the embed from your RelayEmail server:

```html
<script
  src="https://your-relayemail-host.com/embed.js"
  data-key="YOUR_PUBLIC_KEY"
  data-theme="default"
  data-accent="#4f46e5"
></script>
<div data-relayemail-form></div>
```

The script injects a contact form that POSTs to `/v1/submit`.

### Option B: Custom form (fetch)

```html
<form id="contact-form">
  <input name="name" required />
  <input name="email" type="email" required />
  <textarea name="message" required></textarea>
  <button type="submit">Send</button>
</form>

<script>
  document.getElementById('contact-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.target;
    const res = await fetch('https://your-relayemail-host.com/v1/submit', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        publicKey: 'YOUR_PUBLIC_KEY',
        name: form.name.value,
        email: form.email.value,
        message: form.message.value,
      }),
    });
    if (res.ok) {
      alert('Thanks!');
      form.reset();
    } else {
      alert('Something went wrong.');
    }
  });
</script>
```

### Option C: Configurable form (static sites)

For static sites (GitHub Pages, etc.), use data attributes so you can configure the API URL and key without rebuilding:

```html
<div id="contact-form-wrapper"
  data-relayemail-api="https://your-relayemail-host.com"
  data-relayemail-key="YOUR_PUBLIC_KEY">
  <!-- Fallback: mailto link when not configured -->
  <p><a href="mailto:you@example.com">Email us</a></p>
  <form id="contact-form" style="display:none">...</form>
</div>
<script src="contact-form.js"></script>
```

Your `contact-form.js` reads `data-relayemail-api` and `data-relayemail-key`; when both are set, it shows the form and POSTs to `{api}/v1/submit`.

## 4. Submit payload

`POST /v1/submit` accepts JSON:

```json
{
  "publicKey": "pk_xxx",
  "name": "Jane",
  "email": "jane@example.com",
  "message": "Hello!"
}
```

You can also send arbitrary fields; they are stored as submitted. The `Origin` header must match an allowed domain for the site.

## 5. List sites (authenticated)

```bash
curl https://your-host/v1/sites \
  -H "Authorization: Bearer YOUR_FIREBASE_ID_TOKEN"
```

## 6. Get site submissions

Submissions are visible in the RelayEmail app at `/app/`. API endpoints for listing submissions may be added in a future release.
