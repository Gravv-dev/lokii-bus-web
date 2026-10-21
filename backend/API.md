# Lokii Cookies API

Base URL: `http://localhost:8080/api/v1` in local development. JSON responses
use `{ "data": ... }`; errors use `{ "error": "..." }`. Product and order list
routes include pagination metadata. Product prices and order totals are decimal
currency values sourced from integer cents in PostgreSQL.

The browser session uses an `HttpOnly` `session` cookie and a readable
`csrf_token` cookie. Authenticated `POST`, `PATCH`, `PUT`, and `DELETE` requests
must include the same CSRF value in `X-CSRF-Token`. Send credentials with
cross-origin requests. Configure `CORS_ALLOWED_ORIGINS` explicitly.

## Health and catalog

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/health` | Database-backed readiness status |
| GET | `/api/v1/products?page=1&page_size=20&category_id=<uuid>&search=<text>` | Active product listing |
| GET | `/api/v1/products/:id` | Active product detail |
| GET | `/api/v1/categories` | Categories |

## Authentication and customer

| Method | Path | Access | Purpose |
| --- | --- | --- | --- |
| POST | `/api/v1/auth/register` | Public, rate limited | Register customer (`name`, `email`, `password`) |
| POST | `/api/v1/auth/login` | Public, rate limited | Create session (`email`, `password`) |
| POST | `/api/v1/auth/logout` | Customer/admin + CSRF | Revoke current session |
| GET | `/api/v1/auth/me` | Customer/admin | Current user |
| GET | `/api/v1/cart` | Customer/admin | Current user's cart |
| POST | `/api/v1/cart/items` | Customer/admin + CSRF | Add `{ "product_id": "<uuid>", "quantity": 1 }` |
| PATCH | `/api/v1/cart/items/:id` | Customer/admin + CSRF | Set `{ "quantity": 2 }` |
| DELETE | `/api/v1/cart/items/:id` | Customer/admin + CSRF | Remove own cart item |
| POST | `/api/v1/orders` | Customer/admin + CSRF | Create order from current cart |
| GET | `/api/v1/orders?page=1&page_size=20` | Customer/admin | Own order history |
| GET | `/api/v1/orders/:id` | Customer/admin | One of the current user's orders |

Registration only creates customer accounts. The server reads product prices
from the database; cart and order requests do not accept client-supplied prices.

## Administration

All routes below require a valid admin session and CSRF token for mutations.
Create the initial admin via the bootstrap environment variables described in
the main README.

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/api/v1/admin/products` | Create product (name, slug, description, image_url, price, inventory, optional category_id/active) |
| PUT | `/api/v1/admin/products/:id` | Replace product fields |
| DELETE | `/api/v1/admin/products/:id` | Deactivate product (soft delete) |
| POST | `/api/v1/admin/categories` | Create category (name, slug, description) |
| PUT | `/api/v1/admin/categories/:id` | Update category |
| DELETE | `/api/v1/admin/categories/:id` | Delete category; linked products have no category |
| PATCH | `/api/v1/admin/products/:id/inventory` | Set `{ "quantity": 12 }` |
| GET | `/api/v1/admin/orders?page=1&page_size=20` | View customer orders |
| PATCH | `/api/v1/admin/orders/:id/status` | Set pending/processing/shipped/completed/cancelled |
| GET | `/api/v1/admin/reports/sales` | Lifetime completed-order count and order value |

Product payload `price` is expressed in major currency units and is rounded to
cents by the server. Use a valid HTTPS image URL or a local `/images/...` URL.
