/* =========================================================
   main.js — smooth scroll, animations, interactions
   ========================================================= */
import { initCookie3D } from "./cookie3d.js";

const prefersReduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
const isTouch = window.matchMedia("(hover: none)").matches;

gsap.registerPlugin(ScrollTrigger);

/* ---------------- smooth scrolling ---------------- */
let lenis = null;
if (!prefersReduced && typeof Lenis !== "undefined") {
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
  const tl = gsap.timeline();
  tl.to(".preloader__inner", { y: -20, opacity: 0, duration: 0.5, ease: "power2.in" })
    .to(preloader, { yPercent: -100, duration: 0.85, ease: "power4.inOut" }, "-=0.1")
    .set(preloader, { display: "none" })
    .add(introHero, "-=0.45");
}
window.addEventListener("load", () => setTimeout(hidePreloader, 550));
setTimeout(() => { if (preloader.style.display !== "none") hidePreloader(); }, 4000);

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

// marquee — base drift + scroll velocity
const track = document.getElementById("marqueeTrack");
let marqueeX = 0;
const marqueeSpeed = { v: 0.6 };
gsap.ticker.add(() => {
  if (!track) return;
  marqueeX -= marqueeSpeed.v;
  const half = track.scrollWidth / 2;
  if (Math.abs(marqueeX) >= half) marqueeX = 0;
  track.style.transform = `translate3d(${marqueeX}px,0,0)`;
});
ScrollTrigger.create({
  trigger: ".marquee",
  start: "top bottom",
  end: "bottom top",
  onUpdate: (self) => {
    gsap.to(marqueeSpeed, { v: 0.6 + Math.abs(self.getVelocity()) / 900, duration: 0.4, overwrite: true });
  },
});

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
    modifiers: { x: (x) => `${parseFloat(x) % (rail.scrollWidth - window.innerWidth + 40 || 1)}px` },
  });
  ScrollTrigger.create({
    trigger: ".quotes", start: "top bottom", end: "bottom top",
    onEnter: () => railTween.play(),
    onLeave: () => railTween.pause(),
    onEnterBack: () => railTween.play(),
    onLeaveBack: () => railTween.pause(),
  });
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
    gsap.delayedCall(2, () => railTween.play());
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
let count = 0;
function addToCart(n = 1) {
  count += n;
  gsap.fromTo(cartCount, { scale: 1 }, { scale: 1.35, duration: 0.18, yoyo: true, repeat: 1, ease: "power2.out" });
  cartCount.textContent = count;
}
document.querySelectorAll("[data-add]").forEach((btn) => {
  btn.addEventListener("click", () => {
    addToCart();
    btn.classList.add("is-added");
    btn.textContent = "Added ✓";
    showToast(`${btn.dataset.name} added — $${(+btn.dataset.price).toFixed(2)}`);
    setTimeout(() => {
      btn.classList.remove("is-added");
      btn.textContent = "Add +";
    }, 1500);
  });
});

/* ---------------- box builder ---------------- */
const tray = document.getElementById("boxTray");
const boxCount = document.getElementById("boxCount");
const boxPrice = document.getElementById("boxPrice");
const boxAdd = document.getElementById("boxAdd");
const COOKIE_PRICE = 3.9;
const MAX = 6;
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
      slot.style.background = `radial-gradient(circle at 34% 30%, color-mix(in srgb, ${item.color} 78%, #ffffff), ${item.color} 78%)`;
      slot.title = item.name;
    } else {
      slot.classList.remove("is-filled");
      slot.style.background = "transparent";
      slot.title = "";
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
    const item = { name: fl.dataset.flavour, color: fl.dataset.color };
    picked.push(item);
    fl.classList.add("is-active");
    const slot = slots[picked.length - 1];
    gsap.fromTo(slot, { scale: 0.3 }, { scale: 1, duration: 0.6, ease: "elastic.out(1, 0.55)" });
    renderBox();
  });
});

boxAdd.addEventListener("click", () => {
  if (picked.length < MAX) return;
  addToCart(MAX);
  showToast(`Box of 6 added — $${(MAX * COOKIE_PRICE).toFixed(2)} 🎉`);
  picked = [];
  document.querySelectorAll(".flavour").forEach((f) => f.classList.remove("is-active"));
  renderBox();
  gsap.fromTo(tray, { scale: 0.97 }, { scale: 1, duration: 0.5, ease: "elastic.out(1, 0.6)" });
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
