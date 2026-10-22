# Crumb & Co.

A small-batch cookie storefront built with Next.js and a Go/Gin API backed by
PostgreSQL. The original storefront design, animations, local images, and
six-cookie visual builder are preserved.

## Requirements

- Node.js 20.9 or later and npm
- Go 1.25 or later (for running the API without Docker)
- Docker Compose (for the included PostgreSQL, API, and storefront services)

## Start locally

1. Copy `.env.example` to `.env` and change the local database password. The
   example values are for local development only.
2. Start PostgreSQL, the API, and the storefront:

   ```bash
   docker compose up --build
   ```

   Open the storefront at [http://localhost:3000](http://localhost:3000). The
   API listens at `http://localhost:8080`, and PostgreSQL is only reachable on
   the private Compose network. The `postgres_data` volume is persistent.
   Migrations run automatically at API startup and are recorded in
   `schema_migrations`; they do not drop or reset existing data.
3. To run the storefront directly on your host instead, install and start it in
   another terminal:

   ```bash
   npm install
   npm run dev
   ```

   Set `NEXT_PUBLIC_API_BASE_URL` in `.env` when the API is hosted elsewhere.

For a Go-only API run, set `DATABASE_URL`, `CORS_ALLOWED_ORIGINS`, and the
optional cookie/admin environment variables for a PostgreSQL instance reachable
from the host (for example, use `localhost` as the database host), then:

```bash
cd backend
go run ./cmd/api
```

### Run with locally installed PostgreSQL (no Docker)

PostgreSQL can run directly on your computer; Docker is optional. Check that
the service is accepting connections:

```bash
pg_isready -h 127.0.0.1 -p 5432
```

If it is stopped on a system using PostgreSQL clusters, start the installed
cluster with `sudo pg_ctlcluster 16 main start` (use your installed version in
place of `16`). Do not delete or reset the cluster.

Create a dedicated development login and database only if those names are not
already in use. First inspect existing roles and databases:

```bash
sudo -u postgres psql -c '\du'
sudo -u postgres psql -l
```

If `lokii_app` and `lokii_cookies_dev` are unused, create them:

```bash
sudo -u postgres psql
```

At the `psql` prompt:

```sql
CREATE ROLE lokii_app LOGIN;
\password lokii_app
CREATE DATABASE lokii_cookies_dev OWNER lokii_app;
\q
```

Choose a local password at the `\password` prompts; do not paste it into chat
or commit it. Copy `.env.local.example` to `.env.local` and put that same
password in its `DATABASE_URL`. In one terminal, run the API with those values:

```bash
set -a
source .env.local
set +a
cd backend
go run ./cmd/api
```

In another terminal, start the website if it is not already running:

```bash
npm run dev
```

Check the API and database connection:

```bash
curl -i http://localhost:8080/health
```

An HTTP `200` response with `{"status":"ok"}` means the API can reach
PostgreSQL. Check `pg_isready` separately to confirm the PostgreSQL server
accepts connections. `GET /api/v1/products` returns an empty list until an
administrator adds real catalog products.

## First administrator and catalog

Set both `BOOTSTRAP_ADMIN_EMAIL` and `BOOTSTRAP_ADMIN_PASSWORD` before the
first API startup to create an administrator account. The password must be
12–72 bytes. Bootstrap does not change the role or password of an existing
account. Remove these variables after the initial account is created.

Products and categories are not seeded with fake data. Create them through the
protected admin API (documented in [backend/API.md](./backend/API.md)); public
registration can only create customer accounts. The storefront displays an
explicit unavailable/empty state if the API cannot supply products.

## Backend security and behavior

- Passwords use bcrypt; login issues a random opaque session cookie, with only
  its SHA-256 hash stored in PostgreSQL. Sessions expire after 14 days.
- Mutating authenticated requests require the session-bound CSRF cookie and
  matching `X-CSRF-Token` header.
- CORS allows only the explicitly configured origins. Set `COOKIE_SECURE=true`
  behind HTTPS in production.
- Authentication is rate-limited per client IP. SQL uses parameterized
  queries, and request JSON is size-limited and rejects unknown fields.
- Product prices and inventory come from PostgreSQL. Checkout locks inventory
  and creates an order/items and inventory decrement in one transaction.
- Orders are pending records only; payment processing, shipping addresses,
  refunds, and email delivery are not implemented.
- Custom six-cookie box selection remains a visual builder and is not accepted
  as a checkout item yet. Standard catalog products use the real cart/order API.

## Validation

```bash
cd backend
find cmd internal -type f -name '*.go' -exec gofmt -w {} +
go vet ./...
go test ./...
go build -o /tmp/lokii-api ./cmd/api
```

Run the web build with `npm run build`. Migration verification needs a reachable
PostgreSQL server; the API applies pending migrations on startup and reports
database connectivity through `GET /health`. Docker is not required to run Go
tests, but it is required for the provided Compose setup.

## Project structure

```text
app/                 Next.js routes and client bootstrap
backend/cmd/api/     Go service entry point
backend/internal/    Configuration, database, HTTP API, and repository
backend/internal/database/migrations/
js/api.js            Centralized browser API client
js/main.js           Storefront interactions and API integration
index.html           Server-rendered storefront markup
css/style.css        Storefront and commerce dialog styling
public/images/       Local visual assets
```
