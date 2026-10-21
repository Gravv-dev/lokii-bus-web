/* =========================================================
   main.js — smooth scroll, animations, interactions
   ========================================================= */
import { initCookie3D } from "./cookie3d.js";
import { APIError, storefrontAPI } from "./api.js";
import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import Lenis from "lenis";

let initialized = false;

export function initPage() {
if (initialized) return;
initialized = true;

const prefersReduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
const isTouch = window.matchMedia("(hover: none)").matches;
let introStarted = false;

gsap.registerPlugin(ScrollTrigger);

/* ---------------- smooth scrolling ---------------- */
let lenis = null;
if (!prefersReduced) {
  lenis = new Lenis({ duration: 1.1, smoothWheel: true, lerp: 0.09 });
  lenis.on("scroll", ScrollTrigger.update);
  gsap.ticker.add((time) => lenis.raf(time * 1000));
  gsap.ticker.lagSmoothing(0);
}

// anchor links through Lenis
document.querySelectorAll('a[href^="#"]').forEach((a) => {
  a.addEventListener("click", (e) => {
    const id = a.getAttribute("href");
    if (id.length < 2) return;
    const el = document.querySelector(id);
    if (!el) return;
    e.preventDefault();
    closeMenu();
    if (lenis) lenis.scrollTo(el, { offset: -90, duration: 1.3 });
    else el.scrollIntoView({ behavior: "smooth" });
  });
});

/* ---------------- preloader ---------------- */
const preloader = document.getElementById("preloader");
function hidePreloader() {
  if (introStarted) return;
  introStarted = true;
  if (!preloader || preloader.style.display === "none") {
    introHero();
    return;
  }
  const tl = gsap.timeline({ onComplete: introHero });
  tl.to(".preloader__inner", { y: -20, opacity: 0, duration: 0.5, ease: "power2.in" })
    .to(preloader, { yPercent: -100, duration: 0.85, ease: "power4.inOut" }, "-=0.1")
    .set(preloader, { display: "none" });
}
if (document.readyState === "complete") {
  setTimeout(hidePreloader, 550);
} else {
  window.addEventListener("load", () => setTimeout(hidePreloader, 550), { once: true });
}
setTimeout(hidePreloader, 4000);

/* ---------------- hero intro ---------------- */
function introHero() {
  const tl = gsap.timeline({ defaults: { ease: "power4.out" } });
  tl.from(".nav__inner", { y: -40, opacity: 0, duration: 0.9 })
    .from(".hero__title .word", { yPercent: 118, duration: 1.1, stagger: 0.07 }, "-=0.6")
    .from(".hero__eyebrow", { y: 18, opacity: 0, duration: 0.7 }, "-=0.9")
    .from(".hero__sub, .hero__cta, .hero__stats", { y: 26, opacity: 0, duration: 0.8, stagger: 0.1 }, "-=0.8")
    .from(".hero__badge, .hero__scroll", { scale: 0.4, opacity: 0, duration: 0.8, stagger: 0.08 }, "-=0.6");

  // count up stats
  document.querySelectorAll("[data-count]").forEach((el) => {
    const target = +el.dataset.count;
    const obj = { v: 0 };
    gsap.to(obj, {
      v: target, duration: 1.6, ease: "power2.out", delay: 0.9,
      onUpdate: () => (el.textContent = Math.round(obj.v)),
    });
  });
}
if (prefersReduced) {
  gsap.set(".hero__title .word, .hero__eyebrow, .hero__sub, .hero__cta, .hero__stats", { clearProps: "all" });
  document.querySelectorAll("[data-count]").forEach((el) => (el.textContent = el.dataset.count));
}

/* ---------------- 3D cookie ---------------- */
const cookie = initCookie3D(document.getElementById("cookieCanvas"));

/* ---------------- scroll animations ---------------- */

// generic fades
gsap.utils.toArray("[data-fade]").forEach((el) => {
  if (el.closest(".hero")) return; // hero handled by intro
  gsap.from(el, {
    y: 34, opacity: 0, duration: 1, ease: "power3.out",
    scrollTrigger: { trigger: el, start: "top 88%" },
  });
});

// split titles into words
gsap.utils.toArray("[data-split]").forEach((el) => {
  const words = el.textContent.trim().split(/\s+/);
  el.innerHTML = words
    .map((w) => `<span class="line" style="display:inline-block;overflow:hidden;vertical-align:top"><span class="word" style="display:inline-block">${w}</span></span>`)
    .join(" ");
  gsap.from(el.querySelectorAll(".word"), {
    yPercent: 115, duration: 0.95, ease: "power4.out", stagger: 0.05,
    scrollTrigger: { trigger: el, start: "top 86%" },
  });
});

// hero: 3D cookie reacts to scroll
ScrollTrigger.create({
  trigger: ".hero",
  start: "top top",
  end: "bottom top",
  scrub: true,
  onUpdate: (self) => cookie && cookie.setScroll(self.progress),
});
gsap.to(".hero__content", {
  y: -70, opacity: 0.15, ease: "none",
  scrollTrigger: { trigger: ".hero", start: "top top", end: "bottom top", scrub: true },
});

// Animate the marquee only while it is visible.
const track = document.getElementById("marqueeTrack");
let marqueeX = 0;
const marqueeSpeed = { v: 0.6 };
if (track && !prefersReduced) {
  let half = track.scrollWidth / 2;
  const resizeMarquee = () => { half = track.scrollWidth / 2; };
  const animateMarquee = () => {
    marqueeX -= marqueeSpeed.v;
    if (Math.abs(marqueeX) >= half) marqueeX = 0;
    track.style.transform = `translate3d(${marqueeX}px,0,0)`;
  };
  window.addEventListener("resize", resizeMarquee, { passive: true });
  ScrollTrigger.create({
    trigger: ".marquee",
    start: "top bottom",
    end: "bottom top",
    onEnter: () => gsap.ticker.add(animateMarquee),
    onLeave: () => gsap.ticker.remove(animateMarquee),
    onEnterBack: () => gsap.ticker.add(animateMarquee),
    onLeaveBack: () => gsap.ticker.remove(animateMarquee),
    onUpdate: (self) => {
      gsap.to(marqueeSpeed, { v: 0.6 + Math.abs(self.getVelocity()) / 900, duration: 0.4, overwrite: true });
    },
  });
}

// parallax blocks
gsap.utils.toArray("[data-parallax]").forEach((el) => {
  gsap.fromTo(el, { y: 60 }, {
    y: -60, ease: "none",
    scrollTrigger: { trigger: el, start: "top bottom", end: "bottom top", scrub: true },
  });
});

// process steps + progress bar
gsap.utils.toArray("[data-step]").forEach((step, i) => {
  gsap.from(step, {
    y: 60, opacity: 0, duration: 0.9, ease: "power3.out",
    scrollTrigger: { trigger: step, start: "top 85%" },
  });
});
ScrollTrigger.create({
  trigger: ".process",
  start: "top 60%",
  end: "bottom 80%",
  scrub: 0.4,
  onUpdate: (self) => {
    const bar = document.getElementById("processBar");
    if (bar) bar.style.width = (self.progress * 100).toFixed(1) + "%";
  },
});

// card tilt (3D)
if (!isTouch && !prefersReduced) {
  document.querySelectorAll("[data-tilt]").forEach((card) => {
    const setX = gsap.quickTo(card, "rotationY", { duration: 0.6, ease: "power3.out" });
    const setY = gsap.quickTo(card, "rotationX", { duration: 0.6, ease: "power3.out" });
    const setR = gsap.quickTo(card, "y", { duration: 0.6, ease: "power3.out" });
    gsap.set(card, { transformPerspective: 900 });
    card.addEventListener("pointermove", (e) => {
      const r = card.getBoundingClientRect();
      const px = (e.clientX - r.left) / r.width - 0.5;
      const py = (e.clientY - r.top) / r.height - 0.5;
      setX(px * 14);
      setY(-py * 14);
      setR(-10);
    });
    card.addEventListener("pointerleave", () => { setX(0); setY(0); setR(0); });
  });
}

// quotes rail — auto scroll + drag
const rail = document.getElementById("quoteRail");
if (rail) {
  const railTween = gsap.to(rail, {
    x: () => -(rail.scrollWidth - window.innerWidth + 40),
    duration: 26, ease: "none", repeat: -1,
    paused: true,
    modifiers: { x: (x) => `${parseFloat(x) % (rail.scrollWidth - window.innerWidth + 40 || 1)}px` },
  });
  if (!prefersReduced) {
    ScrollTrigger.create({
      trigger: ".quotes", start: "top bottom", end: "bottom top",
      onEnter: () => railTween.play(),
      onLeave: () => railTween.pause(),
      onEnterBack: () => railTween.play(),
      onLeaveBack: () => railTween.pause(),
    });
  }
  // drag
  let down = false, startX = 0, startTx = 0;
  const getTx = () => gsap.getProperty(rail, "x");
  rail.addEventListener("pointerdown", (e) => {
    down = true; startX = e.clientX; startTx = getTx();
    rail.classList.add("is-drag"); railTween.pause();
  });
  window.addEventListener("pointermove", (e) => {
    if (!down) return;
    gsap.set(rail, { x: startTx + (e.clientX - startX) });
  });
  window.addEventListener("pointerup", () => {
    if (!down) return;
    down = false; rail.classList.remove("is-drag");
    if (!prefersReduced) gsap.delayedCall(2, () => railTween.play());
  });
}

// footer word mark drift
gsap.fromTo(".footer__word", { xPercent: 6 }, {
  xPercent: -6, ease: "none",
  scrollTrigger: { trigger: ".footer", start: "top bottom", end: "bottom bottom", scrub: true },
});

/* ---------------- nav ---------------- */
const nav = document.getElementById("nav");
const onScrollNav = () => nav.classList.toggle("is-stuck", window.scrollY > 40);
onScrollNav();
window.addEventListener("scroll", onScrollNav, { passive: true });

const burger = document.getElementById("burger");
const navLinks = document.getElementById("navLinks");
function closeMenu() {
  burger.classList.remove("is-open");
  navLinks.classList.remove("is-open");
}
burger.addEventListener("click", () => {
  burger.classList.toggle("is-open");
  navLinks.classList.toggle("is-open");
});

/* ---------------- custom cursor ---------------- */
if (!isTouch) {
  const dot = document.getElementById("cursor");
  const ring = document.getElementById("cursorRing");
  const dx = gsap.quickTo(dot, "x", { duration: 0.08, ease: "power2.out" });
  const dy = gsap.quickTo(dot, "y", { duration: 0.08, ease: "power2.out" });
  const rx = gsap.quickTo(ring, "x", { duration: 0.35, ease: "power3.out" });
  const ry = gsap.quickTo(ring, "y", { duration: 0.35, ease: "power3.out" });
  window.addEventListener("pointermove", (e) => {
    dx(e.clientX); dy(e.clientY); rx(e.clientX); ry(e.clientY);
  });
  document.addEventListener("pointerover", (e) => {
    ring.classList.toggle("is-hover", !!e.target.closest("[data-cursor='hover']"));
  });
}

/* ---------------- toast ---------------- */
const toast = document.getElementById("toast");
const toastText = document.getElementById("toastText");
let toastTimer;
function showToast(msg) {
  toastText.textContent = msg;
  gsap.killTweensOf(toast);
  gsap.timeline()
    .to(toast, { y: 0, yPercent: 0, xPercent: -50, duration: 0.55, ease: "back.out(1.8)", startAt: { yPercent: 140 } })
    .to(toast, { yPercent: 160, duration: 0.5, ease: "power3.in" }, "+=2.2");
  clearTimeout(toastTimer);
  if (cookie) cookie.burst();
}

/* ---------------- cart ---------------- */
const cartCount = document.getElementById("cartCount");
const shopGrid = document.getElementById("shopGrid");
const shopMessage = document.getElementById("shopMessage");
const commerceDialog = document.getElementById("commerceDialog");
const commerceMessage = document.getElementById("commerceMessage");
const authPanel = document.getElementById("authPanel");
const cartPanel = document.getElementById("cartPanel");
const authForm = document.getElementById("authForm");
const nameField = document.getElementById("nameField");
const authSubmit = document.getElementById("authSubmit");
const authModeToggle = document.getElementById("authModeToggle");
const cartContents = document.getElementById("cartContents");
const cartTotal = document.getElementById("cartTotal");
const orderContents = document.getElementById("orderContents");
const checkoutBtn = document.getElementById("checkoutBtn");
const accountBtn = document.getElementById("accountBtn");
let currentUser = null;
let registering = false;

function updateCartCount(count) {
  cartCount.textContent = count;
  gsap.fromTo(cartCount, { scale: 1 }, { scale: 1.25, duration: 0.18, yoyo: true, repeat: 1, ease: "power2.out" });
}

function showCommerceMessage(message = "") {
  commerceMessage.textContent = message;
}

function showAuth(message = "") {
  authPanel.hidden = false;
  cartPanel.hidden = true;
  showCommerceMessage(message);
  if (!commerceDialog.open) commerceDialog.showModal();
}

function showCartPanel() {
  authPanel.hidden = true;
  cartPanel.hidden = false;
  showCommerceMessage("");
  if (!commerceDialog.open) commerceDialog.showModal();
  refreshCart().catch((error) => showCommerceMessage(error.message));
}

function setUser(user) {
  currentUser = user;
  accountBtn.textContent = user ? user.name.split(/\s+/)[0] : "Sign in";
  accountBtn.setAttribute("aria-label", user ? `Signed in as ${user.name}` : "Sign in");
}

function money(value) {
  return `$${Number(value).toFixed(2)}`;
}

function showProductError(message) {
  shopGrid.replaceChildren();
  shopGrid.setAttribute("aria-busy", "false");
  shopMessage.textContent = message;
}

function renderProducts(products) {
  shopGrid.replaceChildren();
  if (products.length === 0) {
    showProductError("No cookies are available right now. Please check back soon.");
    return;
  }
  for (const product of products) {
    const card = document.createElement("article");
    card.className = "card";
    card.dataset.tilt = "";
    card.dataset.cursor = "hover";

    const art = document.createElement("div");
    art.className = "card__art";
    const image = document.createElement("img");
    image.src = product.image_url || "/images/shop-classic-chip.webp";
    image.alt = product.name;
    image.width = 720;
    image.height = 480;
    image.loading = "lazy";
    image.decoding = "async";
    art.append(image);

    const body = document.createElement("div");
    body.className = "card__body";
    const row = document.createElement("div");
    row.className = "card__row";
    const title = document.createElement("h3");
    title.textContent = product.name;
    const price = document.createElement("span");
    price.className = "card__price";
    price.textContent = money(product.price);
    row.append(title, price);

    const description = document.createElement("p");
    description.textContent = product.description;
    const foot = document.createElement("div");
    foot.className = "card__foot";
    const tag = document.createElement("span");
    tag.className = "tag";
    tag.textContent = product.inventory > 0 ? "Fresh from the oven" : "Sold out";
    const button = document.createElement("button");
    button.type = "button";
    button.className = "card__add";
    button.dataset.add = "";
    button.dataset.productId = product.id;
    button.dataset.name = product.name;
    button.textContent = product.inventory > 0 ? "Add +" : "Sold out";
    button.disabled = product.inventory < 1;
    foot.append(tag, button);
    body.append(row, description, foot);
    card.append(art, body);
    shopGrid.append(card);
  }
  shopGrid.setAttribute("aria-busy", "false");
  shopMessage.textContent = "";
}

async function loadProducts() {
  shopGrid.setAttribute("aria-busy", "true");
  shopGrid.replaceChildren();
  try {
    const result = await storefrontAPI.products();
    renderProducts(result.data || []);
  } catch (error) {
    showProductError(error.message);
  }
}

async function refreshCart() {
  const cart = await storefrontAPI.cart();
  updateCartCount(cart.items.reduce((total, item) => total + item.quantity, 0));
  cartContents.replaceChildren();
  orderContents.hidden = true;
  cartContents.hidden = false;
  cartTotal.hidden = false;
  checkoutBtn.hidden = false;
  if (cart.items.length === 0) {
    const empty = document.createElement("p");
    empty.className = "commerce-empty";
    empty.textContent = "Your cart is empty. Find something delicious in the shop.";
    cartContents.append(empty);
  }
  for (const item of cart.items) {
    const row = document.createElement("div");
    row.className = "commerce-item";
    const details = document.createElement("div");
    details.className = "commerce-item__details";
    const name = document.createElement("p");
    name.className = "commerce-item__name";
    name.textContent = item.name;
    const price = document.createElement("p");
    price.className = "commerce-item__price";
    price.textContent = `${money(item.unit_price)} each · ${money(item.subtotal)}`;
    details.append(name, price);

    const actions = document.createElement("div");
    actions.className = "commerce-item__actions";
    const decrease = document.createElement("button");
    decrease.type = "button";
    decrease.textContent = "−";
    decrease.setAttribute("aria-label", `Remove one ${item.name}`);
    decrease.disabled = item.quantity <= 1;
    decrease.addEventListener("click", () => changeCartItem(item.id, item.quantity - 1));
    const quantity = document.createElement("span");
    quantity.textContent = item.quantity;
    const increase = document.createElement("button");
    increase.type = "button";
    increase.textContent = "+";
    increase.setAttribute("aria-label", `Add one ${item.name}`);
    increase.addEventListener("click", () => changeCartItem(item.id, item.quantity + 1));
    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "commerce-item__remove";
    remove.textContent = "Remove";
    remove.addEventListener("click", () => removeCartItem(item.id));
    actions.append(decrease, quantity, increase, remove);
    row.append(details, actions);
    cartContents.append(row);
  }
  cartTotal.textContent = `Total: ${money(cart.total)}`;
  checkoutBtn.disabled = cart.items.length === 0;
}

async function changeCartItem(itemID, quantity) {
  try {
    await storefrontAPI.updateCartItem(itemID, quantity);
    await refreshCart();
  } catch (error) {
    showCommerceMessage(error.message);
  }
}

async function removeCartItem(itemID) {
  try {
    await storefrontAPI.deleteCartItem(itemID);
    await refreshCart();
  } catch (error) {
    showCommerceMessage(error.message);
  }
}

async function renderOrders() {
  try {
    const orders = await storefrontAPI.orders();
    orderContents.replaceChildren();
    orderContents.hidden = false;
    cartContents.hidden = true;
    cartTotal.hidden = true;
    checkoutBtn.hidden = true;
    if (orders.length === 0) {
      const empty = document.createElement("p");
      empty.className = "commerce-empty";
      empty.textContent = "You haven't placed an order yet.";
      orderContents.append(empty);
      return;
    }
    for (const order of orders) {
      const row = document.createElement("div");
      row.className = "commerce-order";
      const summary = document.createElement("span");
      summary.textContent = `Order ${order.id.slice(0, 8)} · ${order.status}`;
      const total = document.createElement("strong");
      total.textContent = money(order.total);
      row.append(summary, total);
      orderContents.append(row);
    }
  } catch (error) {
    showCommerceMessage(error.message);
  }
}

document.getElementById("commerceClose").addEventListener("click", () => commerceDialog.close());
accountBtn.addEventListener("click", () => currentUser ? showCartPanel() : showAuth());
document.getElementById("cartBtn").addEventListener("click", () => {
  if (!currentUser) {
    showAuth("Sign in or create an account to view your cart.");
    return;
  }
  showCartPanel();
});
authModeToggle.addEventListener("click", () => {
  registering = !registering;
  nameField.hidden = !registering;
  authForm.elements.password.autocomplete = registering ? "new-password" : "current-password";
  authForm.elements.name.required = registering;
  authSubmit.textContent = registering ? "Create account" : "Sign in";
  authModeToggle.textContent = registering ? "I already have an account" : "Create an account";
  document.getElementById("commerceTitle").textContent = registering ? "Join the crumb club" : "Your account";
  showCommerceMessage("");
});
authForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  authSubmit.disabled = true;
  const formData = new FormData(authForm);
  try {
    const user = registering
      ? await storefrontAPI.register(formData.get("name").trim(), formData.get("email").trim(), formData.get("password"))
      : await storefrontAPI.login(formData.get("email").trim(), formData.get("password"));
    setUser(user);
    authForm.reset();
    registering = false;
    nameField.hidden = true;
    authForm.elements.name.required = false;
    authModeToggle.textContent = "Create an account";
    document.getElementById("commerceTitle").textContent = "Your cart";
    await refreshCart();
    cartPanel.hidden = false;
    authPanel.hidden = true;
    showCommerceMessage("");
  } catch (error) {
    showCommerceMessage(error.message);
  } finally {
    authSubmit.disabled = false;
  }
});
document.querySelectorAll("[data-commerce-tab]").forEach((tab) => {
  tab.addEventListener("click", () => {
    document.querySelectorAll("[data-commerce-tab]").forEach((button) => button.classList.remove("is-active"));
    tab.classList.add("is-active");
    if (tab.dataset.commerceTab === "orders") renderOrders();
    else refreshCart().catch((error) => showCommerceMessage(error.message));
  });
});
document.getElementById("signOutBtn").addEventListener("click", async () => {
  try {
    await storefrontAPI.logout();
    setUser(null);
    updateCartCount(0);
    commerceDialog.close();
  } catch (error) {
    showCommerceMessage(error.message);
  }
});
checkoutBtn.addEventListener("click", async () => {
  checkoutBtn.disabled = true;
  try {
    const order = await storefrontAPI.createOrder();
    updateCartCount(0);
    await refreshCart();
    showCommerceMessage(`Order ${order.id.slice(0, 8)} was placed and is ${order.status}.`);
  } catch (error) {
    showCommerceMessage(error.message);
  } finally {
    checkoutBtn.disabled = false;
  }
});
shopGrid.addEventListener("click", async (event) => {
  const button = event.target.closest("[data-add]");
  if (!button || button.disabled) return;
  if (!button.dataset.productId) {
    showToast("This product is not available from the store service.");
    return;
  }
  if (!currentUser) {
    showAuth("Sign in or create an account to add cookies to your cart.");
    return;
  }
  button.disabled = true;
  try {
    const cart = await storefrontAPI.addCartItem(button.dataset.productId);
    updateCartCount(cart.items.reduce((total, item) => total + item.quantity, 0));
    button.classList.add("is-added");
    button.textContent = "Added ✓";
    showToast(`${button.dataset.name} added to your cart.`);
    window.setTimeout(() => {
      button.classList.remove("is-added");
      button.textContent = "Add +";
      button.disabled = false;
    }, 1500);
  } catch (error) {
    showToast(error.message);
    button.disabled = false;
  }
});

loadProducts();
storefrontAPI.me().then((user) => {
  setUser(user);
  return storefrontAPI.cart();
}).then((cart) => {
  updateCartCount(cart.items.reduce((total, item) => total + item.quantity, 0));
}).catch((error) => {
  if (!(error instanceof APIError) || error.status !== 401) {
    console.error("Could not restore the customer session:", error);
  }
});

/* ---------------- box builder ---------------- */
const tray = document.getElementById("boxTray");
const boxCount = document.getElementById("boxCount");
const boxPrice = document.getElementById("boxPrice");
const boxAdd = document.getElementById("boxAdd");
const COOKIE_PRICE = 3.9;
const MAX = 6;
const previewImages = [
  "shop-classic-chip.webp",
  "shop-brown-butter.webp",
  "shop-raspberry-white.webp",
  "shop-double-fudge.webp",
  "cookie-flavour-trio.webp",
  "cookies-and-milk.webp",
];
let picked = [];

for (let i = 0; i < MAX; i++) {
  const slot = document.createElement("div");
  slot.className = "box__slot";
  slot.textContent = i + 1;
  tray.appendChild(slot);
}
const slots = [...tray.children];

function renderBox() {
  slots.forEach((slot, i) => {
    const item = picked[i];
    if (item) {
      slot.classList.add("is-filled");
      slot.classList.remove("is-preview");
      slot.style.backgroundImage = `url("/images/${item.image}")`;
      slot.title = item.name;
      slot.setAttribute("aria-label", `Cookie ${i + 1}: ${item.name}`);
    } else {
      slot.classList.remove("is-filled");
      slot.classList.add("is-preview");
      slot.style.backgroundImage = `linear-gradient(rgba(255, 246, 234, 0.2), rgba(255, 246, 234, 0.2)), url("/images/${previewImages[i]}")`;
      slot.title = "";
      slot.setAttribute("aria-label", `Empty cookie slot ${i + 1}, preview image`);
    }
  });
  boxCount.textContent = picked.length;
  boxPrice.textContent = "$" + (picked.length * COOKIE_PRICE).toFixed(2);
  boxAdd.disabled = picked.length < MAX;
  boxAdd.textContent = picked.length < MAX ? `Pick ${MAX - picked.length} more` : "Add box to cart";
}

document.querySelectorAll(".flavour").forEach((fl) => {
  fl.addEventListener("click", () => {
    if (picked.length >= MAX) {
      showToast("Box is full! Add it or start over.");
      return;
    }
    const item = {
      name: fl.dataset.flavour,
      color: fl.dataset.color,
      image: fl.dataset.image,
    };
    picked.push(item);
    fl.classList.add("is-active");
    const slot = slots[picked.length - 1];
    gsap.fromTo(slot, { scale: 0.3 }, { scale: 1, duration: 0.6, ease: "elastic.out(1, 0.55)" });
    renderBox();
  });
});

boxAdd.addEventListener("click", () => {
  if (picked.length < MAX) return;
  showToast("Custom six-cookie boxes are not available in online checkout yet.");
});
renderBox();

/* ---------------- newsletter ---------------- */
const form = document.getElementById("ctaForm");
const ctaMsg = document.getElementById("ctaMsg");
form.addEventListener("submit", (e) => {
  e.preventDefault();
  const input = form.querySelector("input");
  if (!/^\S+@\S+\.\S+$/.test(input.value)) {
    ctaMsg.textContent = "Hmm, that email looks half-baked. Try again!";
    ctaMsg.style.color = "#a4472a";
    gsap.fromTo(form, { x: -8 }, { x: 0, duration: 0.5, ease: "elastic.out(1, 0.4)" });
    return;
  }
  ctaMsg.style.color = "#4d6b32";
  ctaMsg.textContent = "You're in! Check your inbox for $5 off 🍪";
  input.value = "";
  showToast("Welcome to the crumb club!");
});

/* ---------------- refresh on load ---------------- */
window.addEventListener("load", () => ScrollTrigger.refresh());
}
