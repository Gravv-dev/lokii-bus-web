# Crumb & Co.

A small-batch cookie bakery landing page built with Next.js. The site includes a
product shop, a six-cookie box builder, a photo gallery, and animated visual
effects.

## Requirements

- Node.js 20.9 or later
- npm

## Getting started

Install dependencies:

```bash
npm install
```

Run the local development server:

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

## Available commands

| Command | Description |
| --- | --- |
| `npm run dev` | Start the Next.js development server |
| `npm run build` | Create an optimized production build |
| `npm start` | Serve the production build (run `npm run build` first) |

## Project structure

```text
app/
  globals.css       Imports the site's global stylesheet
  interactions.js   Loads browser-only page interactions
  layout.js         Root layout, fonts, and page metadata
  page.js           Next.js home route
css/
  style.css         Layout, components, responsive styles, and animations
js/
  main.js           GSAP animations, navigation, cart, and box builder
  cookie3d.js       Three.js cookie scene in the hero
public/
  images/           Local optimized WebP images used by the site
index.html          Source markup for the landing page
```

The home route reads the `<body>` markup from `index.html` and renders it on the
server. Keep page content in that file; `app/page.js` removes embedded script
tags because Next.js loads the client interactions from `app/interactions.js`.
Global document metadata and font links are configured in `app/layout.js`.

## Main features

- Responsive navigation with in-page links.
- Shop cards with product images and a demo cart counter.
- Box builder with six photo-filled tray slots. Choose flavours to replace the
  previews, then add the completed box to the cart.
- Kitchen photo gallery.
- GSAP scroll and entrance animations, Lenis smooth scrolling, and a Three.js
  hero scene. The 3D scene pauses when it is off-screen or the tab is hidden.
- Reduced-motion support for visitors who prefer less animation.

The cart and newsletter interactions are client-side demos; they are not
connected to a payment provider, subscription service, or backend.

## Images

Place site images in `public/images/` and reference them with a root-relative URL,
for example `/images/cookies-story.webp`. Prefer compressed WebP assets, provide
meaningful `alt` text and intrinsic `width` and `height`, and use lazy loading
for images below the fold.

Shop product images are mapped in the `data-image` attributes on the flavour
buttons in `index.html`. The six tray preview images are listed in
`previewImages` in `js/main.js`; keep these filenames in sync with files in
`public/images/`.

## Production

Build and run the production server:

```bash
npm run build
npm start
```

The application can be deployed to a Node.js host that supports Next.js.
Generated `.next/` output and installed `node_modules/` dependencies should not
be committed.
