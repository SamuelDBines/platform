# RelayEmail Deployment

This guide covers deploying RelayEmail to production.

## Prerequisites

- Go 1.21+
- Firebase project (Auth + optional Firestore for persistence)
- Stripe account (for Pro billing)
- A host (Cloud Run, Fly.io, Railway, or any VPS)

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `ADDR` | No | Listen address (default `:11000`) |
| `AUTH_MODE` | Yes | `firebase` for production |
| `FIREBASE_PROJECT_ID` | Yes | Your Firebase project ID |
| `GOOGLE_APPLICATION_CREDENTIALS` | Yes* | Path to Firebase service account JSON |
| `STRIPE_SECRET_KEY` | For Pro | Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | For Pro | Stripe webhook signing secret |
| `STRIPE_PRICE_ID_PRO` | For Pro | Stripe Price ID for Pro plan |
| `STRIPE_SUCCESS_URL` | For Pro | Redirect after checkout |
| `STRIPE_CANCEL_URL` | For Pro | Redirect on checkout cancel |

\* On Cloud Run, use Workload Identity; on other hosts, set the path to your service account file.

## Build

```bash
make build
```

Binaries are written to `./bin/`:
- `relayemail-server` — full stack (marketing, app, API)
- `relayemail-api` — API only

## Deploy to Cloud Run

1. Build a container:

```dockerfile
FROM gcr.io/distroless/static-debian12
COPY bin/relayemail-server /relayemail
EXPOSE 8080
ENTRYPOINT ["/relayemail"]
```

2. Set `ADDR=:8080` (Cloud Run expects port 8080).

3. Attach a service account with:
   - Firebase Auth Admin
   - Firestore (if using persistent store)

4. Add your production domain to Firebase Auth authorized domains.

5. Configure Stripe webhook to point to `https://your-domain.run.app/v1/billing/webhook`.

## Deploy to Fly.io

1. Create `fly.toml`:

```toml
app = "relayemail"

[build]
  image = "your-registry/relayemail:latest"

[env]
  ADDR = ":8080"
  AUTH_MODE = "firebase"
  FIREBASE_PROJECT_ID = "your-project"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 0
  processes = ["app"]
```

2. Set secrets:

```bash
fly secrets set FIREBASE_PROJECT_ID=your-project
fly secrets set GOOGLE_APPLICATION_CREDENTIALS_JSON="$(cat sa.json)"
```

## Domain Allowlist

For each site, add allowed domains in the app or via API. The submit endpoint checks the `Origin` header against this list. Add:

- Your marketing site (e.g. `relayemail.example.com`)
- Customer sites that embed your form (e.g. `customer.com`)

## Firebase Auth

1. Enable Auth providers (Email/Password, Google, GitHub).
2. Add authorized domains: your Cloud Run/Fly domain, `localhost` for dev.
3. Ensure `GOOGLE_APPLICATION_CREDENTIALS` is set or use Workload Identity on Cloud Run.

## Stripe

1. Create a Product and Price for the Pro plan.
2. Set `STRIPE_PRICE_ID_PRO` to the Price ID.
3. Create a webhook endpoint for `checkout.session.completed` and `customer.subscription.*`.
4. Set `STRIPE_WEBHOOK_SECRET` from the webhook's signing secret.
