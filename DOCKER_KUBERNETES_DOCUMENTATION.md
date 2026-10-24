# Docker and Kubernetes Deployment Guide

This guide explains how to deploy the Crumb & Co. application with Docker Compose or Kubernetes. Commands, technical identifiers, and configuration examples are kept in copyable form.

The application has three parts:

| Component | Technology | Port |
| --- | --- | ---: |
| Frontend storefront | Next.js | `3000` |
| Backend API | Go | `8080` |
| Database | PostgreSQL | `5432` |

The normal request path is:

```text
User's browser -> Frontend -> Backend API -> PostgreSQL
```

The browser runs the frontend's JavaScript and sends API requests
itself. Therefore, the frontend must be configured with an API URL that the
user's browser can reach. A Docker or Kubernetes internal service name is
usually not reachable from a user's browser.

## 1. Project files

```text
.
├── Dockerfile                    # Next.js frontend image
├── docker-compose.yml            # Docker Compose services
├── .env.example                  # Docker Compose settings example
├── backend/
│   ├── Dockerfile                # Go API image
│   └── internal/
│       ├── config/               # DATABASE_URL and CORS configuration
│       └── database/             # PostgreSQL connection and migrations
└── K8s/
    ├── Backend.yml               # Backend Deployment
    ├── Frontend.yml              # Frontend Deployment
    ├── backend-service.yml       # Internal API Service
    ├── frontend-service.yml      # Frontend NodePort Service
    └── Postgres.yml              # PostgreSQL Service and StatefulSet
```

These are the project files used in this guide. The Kubernetes
YAML files are kept in `K8s/`; apply those files rather than copying examples
from this document into separate manifests.

## 2. Docker and Docker Compose

### 2.1 Docker networking and service names

Docker Compose creates a private network for the services in
`docker-compose.yml`. Containers on that network can find each other by
Compose service name. A container's `localhost` means that same container; it
does not mean another container or the host computer.

| Service | Compose DNS name | Container port | Host port |
| --- | --- | ---: | ---: |
| Frontend | `frontend` | `3000` | `3000` |
| Backend API | `api` | `8080` | `API_PORT` (default `8080`) |
| PostgreSQL | `db` | `5432` | Not published |

The API connects to PostgreSQL at `db:5432`. The browser connects
to the API at `http://localhost:8080` in local development because Compose
publishes the API port on the host. Do not put `http://api:8080` in the
browser-facing frontend setting: the user's browser cannot normally resolve
the Compose-only name `api`.

### 2.2 Environment variables and database connection

Start from the supplied example file. It provides the variables
required by `docker-compose.yml`. Replace the example password before using
the application beyond local testing.

```bash
cp .env.example .env
```

Edit `.env`. The username, database, and password in
`DATABASE_URL` must match the PostgreSQL settings. Use `db` as the database
hostname because that is the Compose service name.

```dotenv
POSTGRES_DB=crumbco
POSTGRES_USER=crumbco
POSTGRES_PASSWORD=replace-with-a-strong-local-password
DATABASE_URL=postgres://crumbco:replace-with-a-strong-local-password@db:5432/crumbco?sslmode=disable

API_PORT=8080
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
CORS_ALLOWED_ORIGINS=http://localhost:3000
COOKIE_SECURE=false
COOKIE_DOMAIN=
IMAGE_TAG=latest
```

If the password contains URL-reserved characters such as `@`,
`:`, `/`, or `#`, URL-encode those characters inside `DATABASE_URL`. Keep the
actual PostgreSQL password and the URL-encoded password equivalent.

Key variables:

- `POSTGRES_DB`, `POSTGRES_USER`, and `POSTGRES_PASSWORD` are used
  by the PostgreSQL image when it initializes an empty data directory.
- `DATABASE_URL` tells the API which database host, port, name,
  and credentials to use.
- `CORS_ALLOWED_ORIGINS` must exactly match the website origin,
  including scheme and port, such as `http://localhost:3000`.
- `NEXT_PUBLIC_API_BASE_URL` is used by browser JavaScript and is
  baked into the frontend at build time.
- Use `COOKIE_SECURE=false` for local HTTP and `true` when serving
  the application over HTTPS.

### 2.3 Build and run with Docker Compose

Check that Compose can read the configuration and all required
variables before starting containers.

```bash
docker compose config -q
```

Build the frontend and API images, then start PostgreSQL, the API,
and the frontend in the background.

```bash
docker compose up -d --build
```

Compose waits for the database health check before starting the
API, and for the API health check before starting the frontend. The API applies
database migrations during startup.

Open the storefront at `http://localhost:3000`. Test the
database-backed API health endpoint at `http://localhost:8080/health`.

### 2.4 Build and push Docker Hub images

Sign in to Docker Hub so Docker is allowed to push images to your
account.

```bash
docker login
```

Compose uses these image names. `IMAGE_TAG` selects the version
tag; the default is `latest`.

```text
garry12132/cookies-web-image:<tag>
garry12132/cookies-api-image:<tag>
```

Set a version tag, then build the images. The frontend API URL is
provided as a build argument; use the URL that the user's browser will be able
to reach.

```bash
export IMAGE_TAG=v1
export NEXT_PUBLIC_API_BASE_URL=https://api.example.com
docker compose build
```

Push the built frontend and API images to Docker Hub.

```bash
docker compose push frontend api
```

The Compose image names differ from the image names currently in
`K8s/Backend.yml` and `K8s/Frontend.yml`. Kubernetes must use the exact image
repository and tag that you pushed. Either update the Kubernetes `image:`
fields, or tag and push images using the names already in those manifests.

For example, tag and push the Compose images under the current
Kubernetes names:

```bash
docker tag garry12132/cookies-web-image:v1 \
  garry12132/production-level-application-frontend:v1
docker tag garry12132/cookies-api-image:v1 \
  garry12132/production-level-application-backend:v1

docker push garry12132/production-level-application-frontend:v1
docker push garry12132/production-level-application-backend:v1
```

If using `:v1`, also update the corresponding Kubernetes
`image:` fields to include `:v1`. Versioned tags make it clear which build is
running and avoid ambiguity from a changing `latest` tag.

### 2.5 Docker status, logs, and container access

Check whether each container is running and whether its health
check passed.

```bash
docker compose ps
```

Follow all service logs or narrow the output to one service to
find startup and connection errors.

```bash
docker compose logs -f --tail=100
docker compose logs -f --tail=100 api
docker compose logs -f --tail=100 db
docker compose logs -f --tail=100 frontend
```

Open a shell in PostgreSQL or the frontend container when
investigating. The database command runs `psql` with the credentials set in
the container.

```bash
docker compose exec db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
docker compose exec frontend sh
```

The API image is distroless and has no shell. Therefore,
`docker compose exec api sh` is expected to fail. Use API logs and the health
endpoint instead.

```bash
curl -i http://localhost:8080/health
docker compose logs --tail=100 api
```

Stop containers but keep the database volume with `down`. Add
`-v` only when you intentionally want to delete the local database data.

```bash
docker compose down
docker compose down -v
```

### 2.6 Docker request flow

1. The user's browser opens `http://localhost:3000`; Docker
   forwards the host port to the frontend container.
2. Frontend JavaScript calls the configured
   `NEXT_PUBLIC_API_BASE_URL`, normally `http://localhost:8080` for local use.
3. Docker forwards the API request from the host to the API
   container on port `8080`.
4. The API resolves `db` on the private Compose network and
   connects to PostgreSQL at `db:5432` using `DATABASE_URL`.
5. PostgreSQL stores data in the `postgres_data` named volume.

Only the frontend and API ports are published to the host in the
supplied Compose configuration. PostgreSQL remains private to the Compose
network.

## 3. Kubernetes deployment

### 3.1 Kubernetes architecture

- A **Namespace** is a logical boundary for related resources.
  These manifests expect `three-tier-application`.
- A **Pod** runs one or more containers. Pod IPs may change when
  Kubernetes replaces a Pod.
- A **Deployment** maintains interchangeable frontend or backend
  Pods and manages updates and replica counts.
- A **StatefulSet** provides stable identity and persistent
  storage for PostgreSQL.
- A **Service** provides a stable network name and routes traffic
  to Pods matching its label selector.
- A **ConfigMap** stores non-secret settings. A **Secret** supplies
  sensitive settings such as database credentials.

Kubernetes Secret data is not automatically protected just
because it is stored as a Secret. Restrict access, enable cluster encryption
where available, and use a secret manager for production.

### 3.2 Kubernetes networking and DNS

| Service | In-namespace DNS name | Port | Type |
| --- | --- | ---: | --- |
| Frontend | `frontend-service` | `3000` | NodePort |
| Backend | `backend-service` | `8080` | ClusterIP |
| PostgreSQL | `postgres-service` | `5432` | ClusterIP |

The backend uses `postgres-service:5432` to connect to the
database. It must not use `localhost:5432`.

The frontend's API URL is used by browser JavaScript. An external
browser cannot normally resolve the internal name `backend-service`. For a
public deployment, build the frontend with a public API URL, usually an HTTPS
Ingress or load-balancer URL. The API ConfigMap must allow the exact public
frontend origin.

### 3.3 Create namespace, Secret, and ConfigMap

The supplied manifests specify a namespace but do not create it.
Create it first; otherwise Kubernetes cannot create namespaced resources there.

```bash
kubectl create namespace three-tier-application
```

Create the PostgreSQL Secret. Choose a strong password and use
the same database, username, and password in the API connection string.

```bash
kubectl -n three-tier-application create secret generic postgres-secret \
  --from-literal=POSTGRES_DB=crumbco \
  --from-literal=POSTGRES_USER=crumbco \
  --from-literal=POSTGRES_PASSWORD='<replace-with-a-strong-password>'
```

Create the backend Secret. The hostname must be the Kubernetes
database Service name, `postgres-service`. URL-encode reserved characters in
the password.

```bash
kubectl -n three-tier-application create secret generic backend-secret \
  --from-literal=DATABASE_URL='postgres://crumbco:<url-encoded-password>@postgres-service:5432/crumbco?sslmode=disable'
```

Create the backend ConfigMap with the exact public site origin.
For local port forwarding, use `http://localhost:3000` instead.

```bash
kubectl -n three-tier-application create configmap backend-config \
  --from-literal=CORS_ALLOWED_ORIGINS='https://www.example.com'
```

The `Backend.yml` Deployment refers to the key `DATABASE_URL` in
`backend-secret` and the key `CORS_ALLOWED_ORIGINS` in `backend-config`. The
`Postgres.yml` StatefulSet refers to `POSTGRES_DB`, `POSTGRES_USER`, and
`POSTGRES_PASSWORD` in `postgres-secret`. Create these resources before their
Pods start. Do not commit real credentials in YAML or source control.

### 3.4 Confirm images and frontend API URL

The current Kubernetes manifests use these image names without an
explicit tag, so Kubernetes uses `latest`:

```text
garry12132/production-level-application-backend
garry12132/production-level-application-frontend
```

Confirm the images exist at the exact names and tags in the
manifests. The Compose image names differ. Prefer a versioned tag and update
the manifest to match it. Also build the frontend with a browser-accessible
`NEXT_PUBLIC_API_BASE_URL`; this value is compiled into the frontend image.

### 3.5 Deploy PostgreSQL, backend, and frontend

Apply PostgreSQL first. It creates an internal Service and a
StatefulSet with persistent storage.

```bash
kubectl apply -f K8s/Postgres.yml
kubectl -n three-tier-application rollout status statefulset/postgres
```

Check that the database Pod is ready and its PersistentVolumeClaim
is bound before starting the backend.

```bash
kubectl -n three-tier-application get pods
kubectl -n three-tier-application get pvc
```

Apply each Service and its corresponding Deployment. Services
provide stable network names while Deployments maintain the application Pods.

```bash
kubectl apply -f K8s/backend-service.yml
kubectl apply -f K8s/Backend.yml
kubectl apply -f K8s/frontend-service.yml
kubectl apply -f K8s/Frontend.yml
```

Wait for the application Deployments to finish rolling out.

```bash
kubectl -n three-tier-application rollout status deployment/backend-deployment
kubectl -n three-tier-application rollout status deployment/frontend-deployment
```

The API connects to PostgreSQL and runs migrations at startup.
Its `/health` endpoint checks database-backed health.

### 3.6 Access the application and port forwarding

Find the NodePort Kubernetes assigned to the frontend Service.
Open that port on a node address reachable from your computer.

```bash
kubectl -n three-tier-application get service frontend-service
```

For local development, port-forward the frontend Service and
open `http://localhost:3000`.

```bash
kubectl -n three-tier-application port-forward service/frontend-service 3000:3000
```

Forward the backend separately to test its health endpoint. This
command only makes the API available from the computer running `kubectl`.

```bash
kubectl -n three-tier-application port-forward service/backend-service 8080:8080
curl -i http://localhost:8080/health
```

Port forwarding is for development and troubleshooting, not a
public production entry point. Production usually needs an HTTPS Ingress or
load balancer, public DNS, the correct frontend build URL, matching CORS, and
secure cookies.

### 3.7 Kubernetes YAML relationships

The repository's complete YAML configurations are the five files
in `K8s/`. The following snippets show the important wiring. Keep the same
names and keys in the actual files and in the Secret/ConfigMap creation
commands.

The Deployments must use the Kubernetes Deployment API version
and exact resource kind shown below.

```yaml
apiVersion: apps/v1
kind: Deployment
```

These backend environment variables connect the API to its
configuration. A missing Secret, ConfigMap, or key prevents the container
from starting.

```yaml
env:
  - name: HTTP_ADDRESS
    value: ":8080"
  - name: DATABASE_URL
    valueFrom:
      secretKeyRef:
        name: backend-secret
        key: DATABASE_URL
  - name: CORS_ALLOWED_ORIGINS
    valueFrom:
      configMapKeyRef:
        name: backend-config
        key: CORS_ALLOWED_ORIGINS
```

The backend Service selects Pods with `app: backend` and sends
traffic to port `8080`; PostgreSQL does the same for `app: postgres` and port
`5432`. The frontend Service selects `app: frontend` Pods on port `3000`.
Deployment selectors must match Pod template labels, and Service selectors
must match the labels of the Pods they serve.

### 3.8 Kubernetes status, logs, exec, and deletion

List the workloads, Services, Pods, and persistent volume claims
to see what Kubernetes created and whether storage is available.

```bash
kubectl -n three-tier-application get deployments
kubectl -n three-tier-application get statefulsets
kubectl -n three-tier-application get pods -o wide
kubectl -n three-tier-application get services
kubectl -n three-tier-application get pvc
```

Describe a failing Pod and inspect recent namespace events to
find scheduling, image, configuration, or storage errors.

```bash
kubectl -n three-tier-application describe pod <pod-name>
kubectl -n three-tier-application get events --sort-by=.metadata.creationTimestamp
```

Read workload logs. Add `-f` to follow new log lines.

```bash
kubectl -n three-tier-application logs deployment/backend-deployment
kubectl -n three-tier-application logs deployment/frontend-deployment
kubectl -n three-tier-application logs statefulset/postgres
```

The PostgreSQL image has a shell, so you can open one for
database troubleshooting. The API image is distroless and has no shell; use
its logs and health endpoint instead.

```bash
kubectl -n three-tier-application exec -it statefulset/postgres -- sh
```

Ask Kubernetes to validate manifests. Server-side dry run needs a
reachable cluster.

```bash
kubectl apply --dry-run=client -f K8s/
kubectl apply --dry-run=server -f K8s/
```

Delete the application resources with the same manifest files.
Deleting a database StatefulSet is not the same as deleting the PVC; inspect
the PVC and only remove it if losing the stored database data is intended.

```bash
kubectl delete -f K8s/Frontend.yml
kubectl delete -f K8s/frontend-service.yml
kubectl delete -f K8s/Backend.yml
kubectl delete -f K8s/backend-service.yml
kubectl delete -f K8s/Postgres.yml

kubectl -n three-tier-application get pvc
kubectl -n three-tier-application delete pvc <postgres-pvc-name>
```

Deleting the namespace removes all namespaced resources. Storage
behavior depends on the cluster's reclaim policy, so verify backups before
deleting production resources.

```bash
kubectl delete namespace three-tier-application
```

## 4. Complete application workflow

### 4.1 Docker Compose

1. The browser requests the frontend at `localhost:3000`.
2. Docker forwards the published host port to the frontend
   container.
3. Frontend JavaScript sends API requests to the
   browser-accessible `NEXT_PUBLIC_API_BASE_URL`.
4. Docker forwards the API port to the Go backend container.
5. The backend resolves `db` on the Compose network and connects
   to PostgreSQL using `DATABASE_URL`.
6. PostgreSQL reads or writes the data in its persistent volume,
   and the API response returns to the browser.

### 4.2 Kubernetes

1. The browser reaches the frontend through NodePort or, in
   production, an Ingress or load balancer.
2. Frontend JavaScript calls the public API URL compiled into the
   frontend image.
3. The public API route sends the request to the backend Service,
   which selects a ready backend Pod.
4. The backend uses the Secret's `DATABASE_URL` to reach
   `postgres-service:5432`.
5. PostgreSQL reads or writes data on its persistent volume, and
   the API response returns to the browser through the public API route.

Kubernetes internal DNS names connect workloads inside the
cluster. They do not automatically create public access for a user's browser.

## 5. Troubleshooting: causes and fixes

### 5.1 Compose reports a missing environment variable

**Cause:** The Compose file requires database settings,
`DATABASE_URL`, and `CORS_ALLOWED_ORIGINS`. Compose cannot start correctly
when required values are missing.

**Fix:** Create `.env` from the supplied example, fill required
values, and validate it before starting.

```bash
cp .env.example .env
docker compose config -q
docker compose up -d --build
```

### 5.2 API says `DATABASE_URL is required`

**Cause:** The Go API intentionally refuses to start without
`DATABASE_URL`.

**Fix:** Set it in `.env` for Compose or in `backend-secret` for
Kubernetes. Use `db` in Compose and `postgres-service` in Kubernetes.

### 5.3 API cannot connect to PostgreSQL

**Cause:** Common causes are a wrong hostname, mismatched username or
password, an invalid URL, or trying to use `localhost` from inside the API
container.

**Fix:** Use this URL shape and substitute values for the target
environment. URL-encode reserved password characters.

```text
postgres://<user>:<url-encoded-password>@<database-service>:5432/<database>?sslmode=disable
```

Use `db` for `<database-service>` in Compose and
`postgres-service` in Kubernetes. Ensure the database name, username, and
password match PostgreSQL's initialization settings.

### 5.4 Changed database password has no effect

**Cause:** PostgreSQL initialization variables only create the user
and database when the data directory is empty. Changing them does not rewrite
credentials in an existing volume.

**Fix:** Deliberately update the existing PostgreSQL role, or reset
the local volume only if its data can be deleted. `down -v` permanently
removes the Compose database volume.

```bash
docker compose down -v
```

### 5.5 API reports missing or invalid CORS

**Cause:** The API requires at least one explicit HTTP or HTTPS
origin. A wildcard, missing scheme, or URL path is not a valid origin.

**Fix:** Set the exact frontend origin, including scheme and port.
For local use it is `http://localhost:3000`; for production use the exact
HTTPS site origin.

Update a Kubernetes ConfigMap and restart the backend so its
environment reflects the new value.

```bash
kubectl -n three-tier-application create configmap backend-config \
  --from-literal=CORS_ALLOWED_ORIGINS='https://www.example.com' \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl -n three-tier-application rollout restart deployment/backend-deployment
```

### 5.6 Frontend loads but API requests fail

**Cause:** The API URL may be unreachable from the browser, CORS may
not allow the frontend origin, or the frontend image may contain an old
build-time `NEXT_PUBLIC_API_BASE_URL`.

**Fix:** Test the API URL from the user's browser network. Set the
correct public URL and rebuild the frontend. Do not use `http://api:8080` in
Compose frontend code or `http://backend-service:8080` for an external
Kubernetes browser.

### 5.7 Kubernetes rejects the Deployment

**Cause encountered:** The frontend and backend manifests previously
used `apiVersion: v1` and lowercase `kind: deployment`. Deployments belong to
`apps/v1`, and Kubernetes resource kinds are case-sensitive.

**Fix:** Use the correct version and capitalization. The corrected
files are `K8s/Backend.yml` and `K8s/Frontend.yml`.

```yaml
apiVersion: apps/v1
kind: Deployment
```

Also make sure the Deployment selector matches its Pod labels.
Otherwise Kubernetes rejects the manifest or the workload will not select the
intended Pods.

### 5.8 Pod has `CreateContainerConfigError`

**Cause:** A referenced Secret or ConfigMap is missing, or the
required key is absent.

**Fix:** Inspect the Pod and confirm resource names and keys. Create
`postgres-secret`, `backend-secret`, and `backend-config` before their
corresponding Pods start.

```bash
kubectl -n three-tier-application describe pod <pod-name>
kubectl -n three-tier-application get secrets
kubectl -n three-tier-application get configmaps
```

### 5.9 Pod has `ImagePullBackOff` or `ErrImagePull`

**Cause:** The image name or tag is not available, a private
registry needs an image pull Secret, or Compose and Kubernetes image names do
not match.

**Fix:** Check the Pod events, push the exact image and tag used by
the manifest, update the manifest if needed, and reapply it.

```bash
kubectl -n three-tier-application describe pod <pod-name>
kubectl apply -f K8s/Backend.yml
kubectl apply -f K8s/Frontend.yml
```

### 5.10 PostgreSQL Pod is Pending

**Cause:** The cluster may not have a compatible StorageClass or
enough storage to bind the StatefulSet's `1Gi` PersistentVolumeClaim.

**Fix:** Inspect the PVC and available StorageClasses. Configure
compatible storage or change the requested size to match the cluster.

```bash
kubectl -n three-tier-application get pvc
kubectl -n three-tier-application describe pvc <postgres-pvc-name>
kubectl get storageclass
```

### 5.11 Service has no endpoints

**Cause:** The Service selector does not match Pod labels, or Pods
are not ready.

**Fix:** Compare Pod labels with Service selectors and resolve Pod
readiness errors.

```bash
kubectl -n three-tier-application get pods --show-labels
kubectl -n three-tier-application describe service backend-service
kubectl -n three-tier-application get endpoints backend-service
```

### 5.12 Backend starts before PostgreSQL is ready

**Cause:** Unlike the Compose setup, the current Kubernetes
manifests do not enforce startup order or wait for database readiness before
starting the API.

**Fix:** Apply PostgreSQL first, wait for the StatefulSet rollout,
then apply the backend. Inspect backend logs if Pods restart. For stronger
production resilience, add appropriate startup/readiness probes and database
reconnection handling.

```bash
kubectl -n three-tier-application rollout status statefulset/postgres
kubectl -n three-tier-application logs deployment/backend-deployment
kubectl -n three-tier-application get pods
```

### 5.13 `localhost` points to the wrong place

| Caller | Correct address |
| --- | --- |
| Compose API -> PostgreSQL | `db:5432` |
| Kubernetes API -> PostgreSQL | `postgres-service:5432` |
| Local user's browser -> Compose API | `http://localhost:8080` |
| Production user's browser -> API | Public API HTTPS URL |

`localhost` refers to the current container, Pod, or computer.
Choose the address based on which component is making the connection.

## 6. Production checklist

- Use strong, unique database and administrator passwords.
- Do not commit `.env` files or real Secret values to source
  control.
- Use persistent storage and maintain tested database backups.
  A persistent volume is not itself a backup.
- Use HTTPS for public frontend and API traffic; set
  `COOKIE_SECURE=true`.
- Set `CORS_ALLOWED_ORIGINS` to the exact frontend origin.
- Build the frontend with the correct public API URL and rebuild
  it whenever that URL changes.
- Use explicit image tags and keep Docker Hub image names and
  Kubernetes manifests consistent.
- Restrict database network access and permissions to Secrets.
