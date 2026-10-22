# Crumb & Co.

Crumb & Co. is a cookie storefront with a Next.js frontend, a Go/Gin API, and
PostgreSQL. Docker Compose builds and runs the complete application.

## Prerequisites

- Docker Engine with the Docker Compose v2 plugin
- A free TCP port on the host for `3000` (web) and `8080` (API)
- For image publishing: a Docker Hub account with permission to push the
  repositories listed in [Publishing images](#publishing-images)

Node.js/npm and Go are only needed when developing or testing outside Docker.
The Dockerfiles use Node.js 22 and Go 1.25 build stages.

## Quick start

Clone the repository and create a private local environment file:

```bash
cp .env.example .env
chmod 600 .env
```

Edit `.env` before starting. Replace the example database password with a
unique password, and use the exact same password in `POSTGRES_PASSWORD` and
`DATABASE_URL`. URL-encode reserved characters in the password when placing it
in `DATABASE_URL` (for example, `@` becomes `%40`). The values committed in
`.env.example` are development placeholders, not production credentials.

Start PostgreSQL, the API, and the frontend with one command:

```bash
docker compose up -d
```

Compose builds the two application images if they are not already present,
creates a persistent `postgres_data` volume, and waits for PostgreSQL before
starting the API. The API applies pending migrations before it listens. The
frontend waits for API readiness. On subsequent code changes, rebuild with:

```bash
docker compose up -d --build
```

Open [http://localhost:3000](http://localhost:3000). The API is available at
[http://localhost:8080](http://localhost:8080); `GET /health` returns HTTP 200
only when the API can reach PostgreSQL. PostgreSQL is not published on a host
port and is reachable only over Compose's private network.

Check the service state and logs:

```bash
docker compose ps
docker compose logs --tail=100 db api frontend
```

To stop the services without deleting database data:

```bash
docker compose down
```

Do not use `docker compose down -v` unless intentionally deleting the database
volume and all persistent application data.

## Environment configuration

Compose reads the untracked `.env` file in the project root. `.env` and other
`.env.*` files are excluded from Git; only example templates are tracked. Do
not commit real credentials or bake secrets into Docker images. In production,
inject secret values from the deployment platform's secret manager and restrict
access to the Compose host.

| Variable | Required | Purpose |
| --- | --- | --- |
| `POSTGRES_DB` | Yes | Database name initialized by the official PostgreSQL image. |
| `POSTGRES_USER` | Yes | Application database login created by PostgreSQL. |
| `POSTGRES_PASSWORD` | Yes | Password for that login; must match `DATABASE_URL`. |
| `DATABASE_URL` | Yes | API connection string. Compose hostname is `db`; include the same database, username, password, and `?sslmode=disable` for the private local network. |
| `CORS_ALLOWED_ORIGINS` | Yes | Comma-separated exact browser origins. For local Compose, `http://localhost:3000`. |
| `NEXT_PUBLIC_API_BASE_URL` | Yes | Browser-reachable API URL, default `http://localhost:8080`. This is baked into the frontend during image build; set it before building. Never set it to the Compose-only hostname `api`. |
| `API_PORT` | No | Host port mapped to API container port 8080; defaults to `8080`. |
| `COOKIE_SECURE` | No | Set `true` when serving over HTTPS in production. Example defaults to `false` for local HTTP. |
| `COOKIE_DOMAIN` | No | Optional cookie domain; leave empty unless production routing requires it. |
| `BOOTSTRAP_ADMIN_EMAIL` | No | Email for optional first administrator creation. Set with the password below. |
| `BOOTSTRAP_ADMIN_PASSWORD` | No | Initial admin password (12–72 bytes). Set both bootstrap variables only for initial provisioning, then remove them from the environment. |
| `IMAGE_TAG` | No | Application image tag, defaults to `latest`; use an immutable version tag such as `v1.0.0` for releases. |

The example local database settings are:

| Setting | Development example |
| --- | --- |
| Database | `crumbco` |
| Username | `crumbco` |
| Password | `change-this-local-only` (replace before use) |
| API's database host | `db` inside Compose |
| Database port | `5432` inside Compose only |

These are not production credentials and must not be reused in a shared
environment. The database volume is initialized only when it is empty. Changing
the PostgreSQL variables later does not change credentials in an existing
volume; update the database role and API URL deliberately, or provision a new
database. Migrations run on API startup and are tracked in
`schema_migrations`; they do not reset existing data.

### Production settings

Before building a production frontend image, configure its public API address
and matching CORS origin, for example:

```dotenv
NEXT_PUBLIC_API_BASE_URL=https://api.example.com
CORS_ALLOWED_ORIGINS=https://www.example.com
COOKIE_SECURE=true
```

Use strong unique database/admin credentials, URL-encode special characters in
`DATABASE_URL`, and supply secrets through your production secret manager
rather than a committed `.env` file. Terminate HTTPS at a trusted reverse
proxy/load balancer and restrict direct network access to the API as required.
The frontend `NEXT_PUBLIC_*` value is public and immutable for a built image;
build a new image to change it. The example Compose file publishes the API on
the host so the browser can reach it; use firewall rules or a reverse proxy in
production to control exposure. Do not expose PostgreSQL publicly.

## Database operations

Connect to the database from the Compose network:

```bash
docker compose exec db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

If the shell does not have those variables, use the values from your `.env`
file in the command. Verify the API/database connection with:

```bash
curl --fail http://localhost:8080/health
```

Back up the database before upgrades or migrations:

```bash
docker compose exec -T db sh -c 'pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB"' > backup.sql
```

The backup contains application data and must be protected like a secret.
Restore only into an intentionally selected database, after taking a separate
backup of its current contents.

## First administrator and catalog

Set both `BOOTSTRAP_ADMIN_EMAIL` and `BOOTSTRAP_ADMIN_PASSWORD` before the
first API startup to create the initial administrator. The password must be
12–72 bytes. Bootstrap does not change the credentials or role of an existing
account. Remove these variables from `.env` and recreate the API after first
provisioning:

```bash
docker compose up -d --force-recreate api
```

Products and categories are intentionally not seeded with fake data. Create
them through the protected admin API described in
[backend/API.md](./backend/API.md). Public registration creates customer
accounts only. The storefront shows an empty/unavailable catalog state until
products are added.

## Docker images and publishing

Compose builds two separate application images:

- Frontend: `garry12132/cookies-web-image:<tag>`
- API: `garry12132/cookies-api-image:<tag>`

The frontend repository name is the requested
[Docker Hub repository](https://hub.docker.com/r/garry12132/cookies-web-image).
The API uses a separate repository because a frontend container and a Go API
container are distinct services and cannot both be represented by one image
tag. Create both repositories in Docker Hub and ensure the authenticated
account can push to them.

After setting `IMAGE_TAG` in `.env` (for example `v1.0.0`), build and publish:

```bash
docker login
docker compose build
docker push "garry12132/cookies-web-image:${IMAGE_TAG}"
docker push "garry12132/cookies-api-image:${IMAGE_TAG}"
```

Use a unique immutable release tag; publish `latest` only if it is part of
your release policy. Do not pass database or administrator credentials as
build arguments. Verify the published frontend at
<https://hub.docker.com/r/garry12132/cookies-web-image> and the API at
<https://hub.docker.com/r/garry12132/cookies-api-image>.

## Troubleshooting

- **Missing-variable or Compose interpolation error:** confirm `.env` exists,
  has no placeholder values, and matches `.env.example`'s variable names. Run
  `docker compose config -q` to validate the configuration without printing
  interpolated secret values.
- **Database remains unhealthy:** inspect `docker compose logs db`; check that
  `POSTGRES_DB`, `POSTGRES_USER`, and `POSTGRES_PASSWORD` match the credentials
  in the existing volume. PostgreSQL only applies initialization variables to
  a new data directory.
- **API exits or is unhealthy:** inspect `docker compose logs api`. Confirm
  `DATABASE_URL` uses hostname `db` (not `localhost`), the correct credentials,
  and database name. The API exits on connection or migration errors rather
  than silently starting without a database.
- **Frontend loads but API calls fail:** the browser needs to resolve
  `NEXT_PUBLIC_API_BASE_URL` from the user's machine. For local Compose use
  `http://localhost:8080`; for a hosted deployment use the public HTTPS API
  address. Ensure the exact frontend origin is included in
  `CORS_ALLOWED_ORIGINS`, then rebuild with `docker compose up -d --build`.
- **Port already in use:** change `API_PORT` for the API and update
  `NEXT_PUBLIC_API_BASE_URL` to match, or stop the process already using that
  port. The frontend currently publishes host port 3000.
- **Code changes do not appear:** run `docker compose up -d --build` to rebuild
  instead of relying on the production containers to mount source files.
- **Resetting local data:** `docker compose down -v` irreversibly removes the
  named database volume. Back up first and only run this when a reset is
  intended.

## Validation and local development

Run backend checks:

```bash
cd backend
go test ./...
go vet ./...
```

Run the frontend production build:

```bash
npm ci
npm run build
```

Docker deployment validation, once Docker is installed:

```bash
docker compose config -q
docker compose up -d --build
docker compose ps
curl --fail http://localhost:8080/health
curl --fail http://localhost:3000/
```

For frontend-only local development, run `npm run dev`. For the API outside
Compose, set `DATABASE_URL`, `CORS_ALLOWED_ORIGINS`, and optional cookie/admin
variables for a PostgreSQL instance reachable from the host (use `localhost`
as its host), then run `cd backend && go run ./cmd/api`. See
[backend/API.md](./backend/API.md) for HTTP routes.

## Project structure

```text
app/                 Next.js routes and client bootstrap
backend/cmd/api/     Go service entry point
backend/internal/    Configuration, database, HTTP API, and repository
backend/internal/database/migrations/
js/api.js            Centralized browser API client
js/main.js           Storefront interactions and API integration
index.html           Storefront markup
css/style.css        Storefront and commerce dialog styling
public/images/       Local visual assets
Dockerfile           Multi-stage Next.js production image
backend/Dockerfile    Multi-stage Go API image
docker-compose.yml   Frontend, API, and PostgreSQL services
```
