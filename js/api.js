const API_BASE_URL = (
  process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080"
).replace(/\/$/, "");

/**
 * @typedef {{ id: string, name: string, slug: string, description: string, image_url: string, price: number, inventory: number, category_id?: string }} Product
 * @typedef {{ id: string, email: string, name: string, role: "customer" | "admin" }} User
 * @typedef {{ id: string, product_id: string, name: string, image_url: string, unit_price: number, quantity: number, subtotal: number }} CartItem
 * @typedef {{ items: CartItem[], total: number }} Cart
 * @typedef {{ id: string, status: string, total: number, created_at: string }} Order
 * @typedef {{ page: number, page_size: number, total: number, total_pages?: number }} Pagination
 */

export class APIError extends Error {
  constructor(message, status) {
    super(message);
    this.name = "APIError";
    this.status = status;
  }
}

function csrfToken() {
  const cookie = document.cookie
    .split("; ")
    .find((entry) => entry.startsWith("csrf_token="));
  return cookie ? decodeURIComponent(cookie.slice("csrf_token=".length)) : "";
}

async function request(path, { method = "GET", body } = {}) {
  const headers = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (!["GET", "HEAD", "OPTIONS"].includes(method)) {
    const token = csrfToken();
    if (token) headers["X-CSRF-Token"] = token;
  }

  let response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method,
      headers,
      credentials: "include",
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new APIError("The store service is unavailable. Please try again shortly.", 0);
  }

  const result = response.status === 204 ? null : await response.json().catch(() => null);
  if (!response.ok) {
    throw new APIError(
      result?.error || `The request failed (${response.status}).`,
      response.status,
    );
  }
  return result;
}

const data = async (path, options) => (await request(path, options))?.data;

/** @type {{
 * products: () => Promise<{data: Product[], pagination: Pagination}>,
 * categories: () => Promise<Array<{id: string, name: string, slug: string, description: string}>>,
 * me: () => Promise<User>,
 * login: (email: string, password: string) => Promise<User>,
 * register: (name: string, email: string, password: string) => Promise<User>,
 * logout: () => Promise<{logged_out: boolean}>,
 * cart: () => Promise<Cart>,
 * addCartItem: (productID: string, quantity?: number) => Promise<Cart>,
 * updateCartItem: (itemID: string, quantity: number) => Promise<Cart>,
 * deleteCartItem: (itemID: string) => Promise<Cart>,
 * createOrder: () => Promise<Order>,
 * orders: () => Promise<Order[]>
 *}} */
export const storefrontAPI = {
  products: () => request("/api/v1/products?page=1&page_size=100"),
  categories: () => data("/api/v1/categories"),
  me: () => data("/api/v1/auth/me"),
  login: (email, password) =>
    data("/api/v1/auth/login", { method: "POST", body: { email, password } }),
  register: (name, email, password) =>
    data("/api/v1/auth/register", {
      method: "POST",
      body: { name, email, password },
    }),
  logout: () => data("/api/v1/auth/logout", { method: "POST", body: {} }),
  cart: () => data("/api/v1/cart"),
  addCartItem: (productID, quantity = 1) =>
    data("/api/v1/cart/items", {
      method: "POST",
      body: { product_id: productID, quantity },
    }),
  updateCartItem: (itemID, quantity) =>
    data(`/api/v1/cart/items/${encodeURIComponent(itemID)}`, {
      method: "PATCH",
      body: { quantity },
    }),
  deleteCartItem: (itemID) =>
    data(`/api/v1/cart/items/${encodeURIComponent(itemID)}`, {
      method: "DELETE",
      body: {},
    }),
  createOrder: () => data("/api/v1/orders", { method: "POST", body: {} }),
  orders: () => data("/api/v1/orders?page=1&page_size=50"),
};
