/* =========================================================
   cookie3d.js — Three.js hero cookie
   ========================================================= */
import * as THREE from "three";
import { RoomEnvironment } from "three/addons/environments/RoomEnvironment.js";

const prefersReduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

/* ---------- procedural cookie texture ---------- */
function makeCookieTexture() {
  const size = 512;
  const c = document.createElement("canvas");
  c.width = c.height = size;
  const ctx = c.getContext("2d");

  // golden base
  const g = ctx.createRadialGradient(size * 0.4, size * 0.35, 30, size / 2, size / 2, size * 0.62);
  g.addColorStop(0, "#eec084");
  g.addColorStop(0.6, "#dda561");
  g.addColorStop(1, "#c98a45");
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, size, size);

  // toasted speckles + cracks
  for (let i = 0; i < 2600; i++) {
    const x = Math.random() * size;
    const y = Math.random() * size;
    const r = Math.random() * 2.4 + 0.3;
    const shade = Math.random();
    ctx.fillStyle =
      shade > 0.6
        ? `rgba(150,92,40,${Math.random() * 0.28})`
        : `rgba(255,232,196,${Math.random() * 0.22})`;
    ctx.beginPath();
    ctx.arc(x, y, r, 0, Math.PI * 2);
    ctx.fill();
  }

  // cracks
  ctx.strokeStyle = "rgba(130,78,32,0.35)";
  for (let i = 0; i < 14; i++) {
    ctx.lineWidth = Math.random() * 1.6 + 0.4;
    ctx.beginPath();
    let x = Math.random() * size;
    let y = Math.random() * size;
    ctx.moveTo(x, y);
    for (let j = 0; j < 5; j++) {
      x += (Math.random() - 0.5) * 70;
      y += (Math.random() - 0.5) * 70;
      ctx.lineTo(x, y);
    }
    ctx.stroke();
  }

  const tex = new THREE.CanvasTexture(c);
  tex.colorSpace = THREE.SRGBColorSpace;
  tex.anisotropy = 8;
  return tex;
}

/* ---------- wavy cookie outline ---------- */
function cookieShape(radius = 1) {
  const shape = new THREE.Shape();
  const pts = 220;
  const bumps = 15;
  for (let i = 0; i <= pts; i++) {
    const a = (i / pts) * Math.PI * 2;
    const r =
      radius *
      (1 +
        0.055 * Math.sin(a * bumps) +
        0.03 * Math.sin(a * 6 + 1.3) +
        0.018 * Math.sin(a * 27 + 0.7));
    const x = Math.cos(a) * r;
    const y = Math.sin(a) * r;
    i === 0 ? shape.moveTo(x, y) : shape.lineTo(x, y);
  }
  return shape;
}

export function initCookie3D(canvas) {
  if (!canvas || prefersReduced) return null;

  const renderer = new THREE.WebGLRenderer({
    canvas,
    antialias: true,
    alpha: true,
    powerPreference: "high-performance",
  });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
  renderer.outputColorSpace = THREE.SRGBColorSpace;
  renderer.toneMapping = THREE.ACESFilmicToneMapping;
  renderer.toneMappingExposure = 1.05;
  renderer.shadowMap.enabled = true;
  renderer.shadowMap.type = THREE.PCFSoftShadowMap;

  const scene = new THREE.Scene();
  const camera = new THREE.PerspectiveCamera(38, 1, 0.1, 100);
  camera.position.set(0, 1.15, 6.2);
  camera.lookAt(0, 0, 0);

  // studio-ish environment for nice reflections
  const pmrem = new THREE.PMREMGenerator(renderer);
  scene.environment = pmrem.fromScene(new RoomEnvironment(), 0.06).texture;

  /* ---------- lights ---------- */
  scene.add(new THREE.HemisphereLight(0xfff0d8, 0x8a5a2c, 0.75));

  const key = new THREE.DirectionalLight(0xfff1dc, 2.1);
  key.position.set(3.4, 5.2, 4.2);
  key.castShadow = true;
  key.shadow.mapSize.set(1024, 1024);
  key.shadow.camera.near = 1;
  key.shadow.camera.far = 20;
  key.shadow.camera.left = key.shadow.camera.bottom = -5;
  key.shadow.camera.right = key.shadow.camera.top = 5;
  key.shadow.bias = -0.0008;
  scene.add(key);

  const rim = new THREE.PointLight(0xff9d5c, 18, 22);
  rim.position.set(-4, 1.4, -3.2);
  scene.add(rim);

  const fill = new THREE.PointLight(0xffe6c2, 8, 20);
  fill.position.set(-2.4, -1.6, 3.4);
  scene.add(fill);

  /* ---------- cookie ---------- */
  const group = new THREE.Group();
  scene.add(group);

  const tex = makeCookieTexture();
  const bodyGeo = new THREE.ExtrudeGeometry(cookieShape(1.35), {
    depth: 0.42,
    bevelEnabled: true,
    bevelThickness: 0.2,
    bevelSize: 0.17,
    bevelSegments: 8,
    curveSegments: 8,
  });
  bodyGeo.center();

  const bodyMat = new THREE.MeshStandardMaterial({
    map: tex,
    bumpMap: tex,
    bumpScale: 0.35,
    roughness: 0.82,
    metalness: 0.02,
    color: 0xffe9cd,
  });
  const body = new THREE.Mesh(bodyGeo, bodyMat);
  body.rotation.x = -Math.PI / 2; // lay flat, top facing up
  body.castShadow = true;
  body.receiveShadow = true;
  group.add(body);

  // chocolate chips
  const chipMat = new THREE.MeshPhysicalMaterial({
    color: 0x2b1608,
    roughness: 0.32,
    metalness: 0.05,
    clearcoat: 0.55,
    clearcoatRoughness: 0.35,
  });
  const chipGeo = new THREE.DodecahedronGeometry(1, 0);
  const chips = new THREE.InstancedMesh(chipGeo, chipMat, 16);
  chips.castShadow = true;
  const dummy = new THREE.Object3D();
  const placed = [];
  for (let i = 0; i < 16; i++) {
    let x, y, ok = false, tries = 0;
    while (!ok && tries < 60) {
      const a = Math.random() * Math.PI * 2;
      const d = Math.sqrt(Math.random()) * 1.02;
      x = Math.cos(a) * d;
      y = Math.sin(a) * d;
      ok = placed.every((p) => Math.hypot(p[0] - x, p[1] - y) > 0.3);
      tries++;
    }
    placed.push([x, y]);
    const s = 0.11 + Math.random() * 0.075;
    dummy.position.set(x, 0.4 + Math.random() * 0.02, y);
    dummy.rotation.set(Math.random() * 0.5, Math.random() * Math.PI, Math.random() * 0.5);
    dummy.scale.set(s, s * 0.6, s);
    dummy.updateMatrix();
    chips.setMatrixAt(i, dummy.matrix);
  }
  group.add(chips);

  // floating crumbs
  const crumbGeo = new THREE.TetrahedronGeometry(0.05);
  const crumbMat = new THREE.MeshStandardMaterial({ color: 0xcf9252, roughness: 0.9 });
  const crumbs = new THREE.InstancedMesh(crumbGeo, crumbMat, 26);
  const crumbData = [];
  for (let i = 0; i < 26; i++) {
    crumbData.push({
      r: 2.1 + Math.random() * 2.4,
      a: Math.random() * Math.PI * 2,
      y: (Math.random() - 0.5) * 3.4,
      sp: 0.1 + Math.random() * 0.28,
      s: 0.5 + Math.random() * 1.1,
      rot: Math.random() * Math.PI,
    });
  }
  scene.add(crumbs);

  // soft contact shadow
  const shadowMat = new THREE.ShadowMaterial({ opacity: 0.16 });
  const shadowPlane = new THREE.Mesh(new THREE.PlaneGeometry(14, 14), shadowMat);
  shadowPlane.rotation.x = -Math.PI / 2;
  shadowPlane.position.y = -1.35;
  shadowPlane.receiveShadow = true;
  scene.add(shadowPlane);

  /* ---------- responsive layout ---------- */
  const state = {
    pointer: { x: 0, y: 0 },
    spin: 0.0028,
    spinBoost: 0,
    squish: 0,
    scroll: 0,
    visible: true,
  };

  function layout() {
    const w = canvas.clientWidth || window.innerWidth;
    const h = canvas.clientHeight || window.innerHeight;
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    const small = w < 860;
    // push the cookie to the right on desktop, centre+up on mobile
    group.position.x = small ? 0 : Math.min(2.1, w * 0.0013);
    group.position.y = small ? 1.0 : -0.05;
    const s = small ? Math.min(1, w / 620) : 1;
    group.scale.setScalar(s);
    group.userData.baseScale = s;
    camera.updateProjectionMatrix();
    group.userData.layoutY = group.position.y;
  }
  layout();
  window.addEventListener("resize", layout);

  /* ---------- interaction ---------- */
  window.addEventListener("pointermove", (e) => {
    state.pointer.x = (e.clientX / window.innerWidth) * 2 - 1;
    state.pointer.y = (e.clientY / window.innerHeight) * 2 - 1;
  });

  const burst = () => {
    state.spinBoost = 0.12;
    state.squish = 1;
  };
  canvas.addEventListener("pointerdown", burst);
  window.__cookieBurst = burst;

  const io = new IntersectionObserver(
    ([entry]) => (state.visible = entry.isIntersecting),
    { threshold: 0.02 }
  );
  io.observe(canvas);

  /* ---------- render loop ---------- */
  const clock = new THREE.Clock();
  let raf;
  function tick() {
    raf = requestAnimationFrame(tick);
    if (!state.visible || document.hidden) return;

    const t = clock.getElapsedTime();
    const base = group.userData.baseScale || 1;

    // idle spin + user boost decay
    state.spinBoost *= 0.94;
    state.squish *= 0.88;

    const targetTiltY = state.pointer.x * 0.45;
    const targetTiltX = state.pointer.y * 0.28;

    group.rotation.y += state.spin + state.spinBoost;
    group.rotation.y += (targetTiltY - (group.rotation.y % (Math.PI * 2)) * 0.02) * 0.012;
    group.rotation.x += (targetTiltX - group.rotation.x) * 0.05;
    group.rotation.z += (-state.pointer.x * 0.12 - group.rotation.z) * 0.04;

    // gentle float + scroll drift
    const floatY = Math.sin(t * 1.1) * 0.09;
    const layoutY = group.userData.layoutY ?? 0;
    group.position.y = layoutY + floatY - state.scroll * 2.2;

    // squish
    const sq = 1 - state.squish * 0.14;
    group.scale.set(base * (2 - sq), base * sq, base * (2 - sq));

    // crumbs drift
    for (let i = 0; i < crumbData.length; i++) {
      const c = crumbData[i];
      c.a += c.sp * 0.006;
      dummy.position.set(
        Math.cos(c.a) * c.r + group.position.x * 0.4,
        c.y + Math.sin(t * 0.6 + i) * 0.25 - state.scroll * 1.6,
        Math.sin(c.a) * c.r - 1.5
      );
      dummy.rotation.set(c.rot + t * 0.4, c.rot, c.rot * 0.5);
      dummy.scale.setScalar(c.s);
      dummy.updateMatrix();
      crumbs.setMatrixAt(i, dummy.matrix);
    }
    crumbs.instanceMatrix.needsUpdate = true;

    renderer.render(scene, camera);
  }

  // stash layout y so float doesn't fight the responsive offset
  tick();

  return {
    setScroll(p) {
      state.scroll = p;
    },
    burst,
    destroy() {
      cancelAnimationFrame(raf);
      window.removeEventListener("resize", layout);
      renderer.dispose();
    },
  };
}
