# Running your own SMTP (Postfix)

RelayEmail can send form submission notifications via your own SMTP server. No third-party email API required.

## Quick start

1. Install Postfix (or your preferred MTA):
   ```bash
   # Ubuntu/Debian
   sudo apt install postfix
   # Choose "Internet Site" and set your domain
   ```

2. Configure RelayEmail:
   ```bash
   SMTP_HOST=localhost
   SMTP_PORT=25
   SMTP_FROM=noreply@yourdomain.com
   ```

3. Set a notify email in the dashboard for each site.

## Deliverability tips

Running your own SMTP is not hard, but **deliverability** depends on:

- **PTR (reverse DNS)** – Your server must have a PTR record pointing to your hostname. Many VPS providers let you set this.
- **SPF** – Add a TXT record: `v=spf1 ip4:YOUR_IP ~all`
- **DKIM** – Sign outgoing mail (optional but recommended)
- **DMARC** – Policy record (optional)

Without these, Gmail/Outlook may spam-filter or reject.

## Port 25

Many ISPs and cloud providers block outbound port 25. Use:

- **Port 587** (submission) with SMTP auth – most providers allow this
- **Relay** – Send via your ISP’s relay or a relay service

Example with auth on 587:

```bash
SMTP_HOST=smtp.yourdomain.com
SMTP_PORT=587
SMTP_USER=your-user
SMTP_PASS=your-password
SMTP_FROM=noreply@yourdomain.com
```

## Local testing

For local dev without a real mail server:

- Use [MailHog](https://github.com/mailhog/MailHog) or [Mailpit](https://github.com/axllent/mailpit) – they catch mail on port 1025
- Or leave SMTP unset – submissions still go to the dashboard, just no email

---

# Inbound email forwarding

RelayEmail can **receive** emails at customer domains and forward them to the owner's inbox (e.g. contact@mybusiness.com → you@gmail.com).

## Server setup

1. **SMTP** must be configured (see above) – used to forward received mail.
2. **MX_HOST** – The hostname customers add to their MX records, e.g. `mx.relayemail.com`.
3. **INBOUND_SMTP_PORT** – Port to listen on (default 25). Must be open for inbound connections.

```bash
SMTP_HOST=mail.yourdomain.com
SMTP_FROM=noreply@yourdomain.com
MX_HOST=mx.yourdomain.com
INBOUND_SMTP_PORT=25
```

## Customer setup

1. Add the domain to the site's allowed domains (for form submission).
2. In the dashboard, add a forward address: `contact@example.com` → `you@gmail.com`.
3. Add an MX record for their domain:
   ```
   MX  10  mx.yourdomain.com
   ```

Mail sent to `contact@example.com` will be received by RelayEmail and forwarded to `you@gmail.com`.

## Port 25

Inbound port 25 is often blocked by cloud providers. Options:

- Request port 25 unblock from your provider (DigitalOcean, etc.).
- Run the inbound server on a host that allows port 25 (e.g. a dedicated mail relay).

---

# Email campaigns (Campaign Monitor–style)

RelayEmail can send **marketing emails** to subscriber lists: create lists, add subscribers (or import from form submissions), compose campaigns, and send.

## Requirements

- **SMTP** configured (see above) – used to send campaign emails.
- **TRACKING_BASE_URL** (optional) – e.g. `https://relayemail.com`. Enables open tracking (1×1 pixel). Must match your public API URL so tracking requests hit your server.

## Flow

1. Create a subscriber list per site.
2. Add subscribers manually or **Import from submissions** (pulls email/name from form submissions).
3. Create a campaign: subject, HTML body, from name/email, select list.
4. Send – emails go to all subscribed contacts.

## Tracking

When `TRACKING_BASE_URL` is set, campaigns include an open-tracking pixel. When a recipient opens the email, a request is recorded. Stats (sent, opened, clicked) are available via the API.
