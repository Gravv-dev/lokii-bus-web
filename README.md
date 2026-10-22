# Crumb & Co.

Crumb & Co. is a cookie storefront. It includes a Next.js website, a Go API,
and a PostgreSQL database. Docker Compose starts all three services together.

## What you need

- Docker Engine
- Docker Compose v2 (`docker compose`)

You do not need to install Node.js, Go, or PostgreSQL to run the project with
Docker.

## Run the whole project

1. Copy the example settings file:

   ```bash
   cp .env.example .env
   ```

2. Open `.env` and replace `POSTGRES_PASSWORD` with a strong password. Put the
   same password in `DATABASE_URL`. The example password is only for local
   development; do not use it on a shared or production server.

   `DATABASE_URL` must use `db` as the database host because that is the
   database service name inside Docker Compose. If your password contains
   special characters such as `@`, URL-encode them in `DATABASE_URL` (for
   example, `@` becomes `%40`).

3. Start the website, API, and database:

   ```bash
   docker compose up -d
   ```

   Compose builds the application images, starts PostgreSQL, waits until it is
   ready, and then starts the API and website. The API creates or updates the
   database tables when it starts.

4. Open the website at [http://localhost:3000](http://localhost:3000).
   The API health check is at [http://localhost:8080/health](http://localhost:8080/health).

Check whether services are running and view their logs:

```bash
docker compose ps
docker compose logs --tail=100
```

To stop the services and keep the database data:

```bash
docker compose down
```

The database is stored in the `postgres_data` Docker volume, so it remains
available when you stop and restart the services. To permanently delete the
local database and all its data, run `docker compose down -v`.

## Configuration

Docker Compose reads settings from `.env` in the project root. Start with
`.env.example`; the example is for local development.

| Setting | Purpose |
| --- | --- |
| `POSTGRES_DB` | Name of the database PostgreSQL creates. |
| `POSTGRES_USER` | Database login used by the API. |
| `POSTGRES_PASSWORD` | Password for that login. Choose your own value. |
| `DATABASE_URL` | API connection string. Use the same database, username, and password as above, with `db` as the host. |
| `CORS_ALLOWED_ORIGINS` | Website origin allowed to call the API. For local use, keep `http://localhost:3000`. |
| `NEXT_PUBLIC_API_BASE_URL` | API address used by the browser. For local use, keep `http://localhost:8080`. |
| `API_PORT` | Port on your computer used to reach the API. Defaults to `8080`. |
| `COOKIE_SECURE` | Use `false` for local HTTP. Set to `true` when the site uses HTTPS. |
| `COOKIE_DOMAIN` | Optional cookie domain. Leave empty for local use. |
| `BOOTSTRAP_ADMIN_EMAIL` | Optional email for creating the first administrator. Set this together with the password below. |
| `BOOTSTRAP_ADMIN_PASSWORD` | Optional first administrator password. It must be 12–72 bytes. |
| `IMAGE_TAG` | Docker image version. Defaults to `latest`. |

Example database settings:

```dotenv
POSTGRES_DB=crumbco
POSTGRES_USER=crumbco
POSTGRES_PASSWORD=use-a-strong-local-password
DATABASE_URL=postgres://crumbco:use-a-strong-local-password@db:5432/crumbco?sslmode=disable
```

Keep the password in `DATABASE_URL` identical to `POSTGRES_PASSWORD`. If you
change the database name or username, update both the PostgreSQL settings and
the connection string. PostgreSQL applies these settings when it first creates
the database volume; editing them later does not change an existing database
login.

The website's API address is included when its image is built. If you change
`NEXT_PUBLIC_API_BASE_URL`, rebuild the services:

```bash
docker compose up -d --build
```

## First administrator and products

To create an administrator when the API starts for the first time, set both
`BOOTSTRAP_ADMIN_EMAIL` and `BOOTSTRAP_ADMIN_PASSWORD` in `.env` before
starting the services. Use a strong password of 12–72 bytes. Bootstrap does
not change an existing account. After the administrator has been created,
remove both values from `.env` and restart the API:

```bash
docker compose up -d --force-recreate api
```

The database starts without product or category records. Add them through the
administrator API. Public registration creates customer accounts only. See
[the API guide](./backend/API.md) for routes and request examples.

## How the services fit together

| Service | What it does | Local address |
| --- | --- | --- |
| `frontend` | Serves the storefront website. | [http://localhost:3000](http://localhost:3000) |
| `api` | Handles products, accounts, carts, and orders. | [http://localhost:8080](http://localhost:8080) |
| `db` | Stores accounts, products, inventory, carts, and orders. | Only available to services inside Compose. |

The frontend calls the API from the visitor's browser. The API connects to
PostgreSQL using the internal hostname `db`. The API runs database migrations
at startup and reports healthy only when it can reach the database.

## Troubleshooting

- **Compose reports a missing setting:** Check that `.env` exists in the
  project root and that required values are filled in. Validate the Compose
  file with `docker compose config -q`.
- **The database or API does not start:** Run `docker compose logs db api`.
  Make sure `DATABASE_URL` uses `db` as the host and matches the database name,
  username, and password configured for PostgreSQL.
- **The website opens but cannot reach the API:** Check that
  `NEXT_PUBLIC_API_BASE_URL` is reachable from your browser and that its
  website origin is listed in `CORS_ALLOWED_ORIGINS`. Rebuild with
  `docker compose up -d --build` after changing the API address.
- **A port is already in use:** Change `API_PORT` to an available port and
  update `NEXT_PUBLIC_API_BASE_URL` to use that port. The website uses port
  `3000`.
- **Changes to existing database credentials have no effect:** PostgreSQL
  keeps the existing database and login in its volume. Update the database
  login deliberately, or remove the volume only if you intend to erase its
  data.

## Production notes

- Use unique, strong database and administrator passwords. Do not commit `.env`
  or put secrets in Docker build arguments.
- Set `COOKIE_SECURE=true` when serving the site over HTTPS.
- Set `NEXT_PUBLIC_API_BASE_URL` to the public HTTPS API address and
  `CORS_ALLOWED_ORIGINS` to the exact website origin, then build the frontend
  image.
- Keep PostgreSQL private. Use a trusted HTTPS reverse proxy or load balancer,
  and restrict access to the API as appropriate for your deployment.
- Provide production secrets through your platform's secret manager rather
  than a checked-in file.

## Development checks

Run backend tests:

```bash
cd backend
go test ./...
```

Build the frontend:

```bash
npm ci
npm run build
```

For more API details, see [backend/API.md](./backend/API.md).
