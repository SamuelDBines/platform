# Platform

This repo is now set up as the control plane for multiple products, not a single app scaffold.

## Structure

```text
platform/
  package.json                # workspace root and runner
  packages/
    components/              # shared Solid UI
    services/                # shared browser/service helpers
  relayemail.net/
    app/                     # Solid app
    theme/                   # domain styles
    public/                  # static assets
    docs/                    # ready for a docs submodule
  statuslater.co.uk/
    app/
    theme/
    public/
    docs/
  services/
    go/
      cmd/platform-api/       # backend entrypoint
```

## Commands

Run everything together from the repo root:

```bash
npm install
npm run dev
```

Useful focused commands:

```bash
npm run dev:web
npm run dev:relayemail
npm run dev:statuslater
npm run dev:api
npm run build
npm run check
```

## Why this shape

- Each domain owns its app, brand theme, public assets, and docs boundary.
- Shared UI and browser-side service helpers live in `packages/*`.
- Go stays the default backend runtime, but the platform runner can be extended later for Python, Dart, or additional workers.
- The root `dev` command starts the frontend workspace and the API together so you can manage the whole platform in one go.
