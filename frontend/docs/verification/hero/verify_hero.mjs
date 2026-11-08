/**
 * Hero layout + motion verification.
 *
 * Usage (preview server must already be running):
 *   NODE_PATH=<playwright node_modules> node docs/verification/hero/verify_hero.mjs [outdir]
 */
import { createRequire } from "node:module";
import { mkdirSync, writeFileSync } from "node:fs";

const require = createRequire(import.meta.url);
const { chromium } = require("playwright");

const OUT = process.argv[2] || "docs/verification/hero";
const URL = process.env.HERO_URL || "http://127.0.0.1:4173/";

const VIEWPORTS = [
  { name: "desktop-1440x900", width: 1440, height: 900 },
  { name: "laptop-1366x768", width: 1366, height: 768 },
  { name: "tablet-768x1024", width: 768, height: 1024 },
  { name: "mobile-390x844", width: 390, height: 844 },
  { name: "narrow-320x640", width: 320, height: 640 },
  { name: "narrow-320x568", width: 320, height: 568 },
];

mkdirSync(OUT, { recursive: true });

const measure = () => {
  const rect = (sel) => {
    const el = document.querySelector(sel);
    if (!el) return null;
    const b = el.getBoundingClientRect();
    return {
      top: Math.round(b.top),
      bottom: Math.round(b.bottom),
      left: Math.round(b.left),
      right: Math.round(b.right),
      height: Math.round(b.height),
    };
  };

  return {
    vh: window.innerHeight,
    vw: window.innerWidth,
    scrollWidth: document.documentElement.scrollWidth,
    hero: (() => {
      const b = document.querySelector(".hero").getBoundingClientRect();
      return {
        top: Math.round(b.top),
        bottom: Math.round(b.bottom),
        height: Math.round(b.height),
      };
    })(),
    nav: rect("header nav"),
    eyebrow: rect(".hero__eyebrow"),
    title: rect("#hero-title"),
    body: rect(".hero__body"),
    actions: rect(".hero__actions"),
    chips: rect(".hero__chips"),
    stage: rect(".hero__stage-inner"),
    footer: rect(".hero__footer"),
    ctas: [...document.querySelectorAll(".hero__cta")].map((el) => {
      const b = el.getBoundingClientRect();
      return {
        label: el.textContent.trim(),
        href: el.getAttribute("href"),
        top: Math.round(b.top),
        bottom: Math.round(b.bottom),
      };
    }),
    stageCanvas: (() => {
      const c = document.querySelector(".tux-stage canvas");
      if (!c) return null;
      const b = c.getBoundingClientRect();
      return {
        top: Math.round(b.top),
        bottom: Math.round(b.bottom),
        width: Math.round(b.width),
        height: Math.round(b.height),
      };
    })(),
  };
};

const browser = await chromium.launch({
  executablePath: "/usr/bin/google-chrome-stable",
  headless: true,
  args: ["--enable-unsafe-swiftshader"],
});

const results = [];

for (const vp of VIEWPORTS) {
  const page = await browser.newPage({
    viewport: { width: vp.width, height: vp.height },
    deviceScaleFactor: 1,
  });
  const errors = [];
  page.on("console", (m) => {
    if (m.type() === "error") errors.push(m.text().slice(0, 200));
  });

  // The intro overlay runs its full choreography on first paint; under
  // SwiftShader that can take ~15s, so allow generously before measuring.
  await page.goto(URL, { waitUntil: "domcontentloaded" });
  await page.waitForFunction(
    () => !document.querySelector('#root > [role="presentation"]'),
    null,
    { timeout: 40000 },
  );
  await page.waitForTimeout(1200);

  const m = await page.evaluate(measure);
  const checks = {
    heroWithinViewport: m.hero.bottom <= m.vh + 1,
    stageVisible: m.stage ? m.stage.bottom <= m.vh : false,
    ctasVisible: m.ctas.every((c) => c.bottom <= m.vh && c.top >= 0),
    noHorizontalOverflow: m.scrollWidth <= m.vw,
    navGap: m.eyebrow && m.nav ? m.eyebrow.top - m.nav.bottom : null,
    modelFillsStage:
      m.stageCanvas && m.stage
        ? Math.round((m.stageCanvas.height / m.stage.height) * 100)
        : null,
  };

  await page.screenshot({ path: `${OUT}/${vp.name}.png` });

  // Scroll motion: one scrub ScrollTrigger should move the stage up and scale
  // it down without moving the page horizontally.
  const before = await page.evaluate(() => {
    const el = document.querySelector(".hero__stage");
    const r = el.getBoundingClientRect();
    return { top: Math.round(r.top), transform: getComputedStyle(el).transform };
  });
  await page.evaluate(() => window.scrollTo(0, window.innerHeight * 0.45));
  await page.waitForTimeout(700);
  const after = await page.evaluate(() => {
    const el = document.querySelector(".hero__stage");
    const r = el.getBoundingClientRect();
    return {
      top: Math.round(r.top),
      transform: getComputedStyle(el).transform,
      scrollY: Math.round(window.scrollY),
      scrollWidth: document.documentElement.scrollWidth,
    };
  });
  await page.screenshot({ path: `${OUT}/${vp.name}-scrolled.png` });
  await page.evaluate(() => window.scrollTo(0, 0));

  results.push({ viewport: vp, checks, metrics: m, scroll: { before, after }, errors });
  await page.close();
}

// Reduced motion: no scroll-linked transform, content immediately visible.
const rmPage = await browser.newPage({
  viewport: { width: 1440, height: 900 },
  reducedMotion: "reduce",
});
await rmPage.goto(URL, { waitUntil: "domcontentloaded" });
await rmPage.waitForTimeout(1500);
const rmBefore = await rmPage.evaluate(() => ({
  visible: getComputedStyle(document.querySelector(".hero__body")).opacity,
  stage: getComputedStyle(document.querySelector(".hero__stage")).transform,
}));
await rmPage.evaluate(() => window.scrollTo(0, window.innerHeight * 0.45));
await rmPage.waitForTimeout(600);
const rmAfter = await rmPage.evaluate(() => ({
  stage: getComputedStyle(document.querySelector(".hero__stage")).transform,
}));
await rmPage.screenshot({ path: `${OUT}/reduced-motion-1440x900.png` });
await rmPage.close();

await browser.close();

const report = {
  generatedAt: new Date().toISOString(),
  url: URL,
  results,
  reducedMotion: { before: rmBefore, after: rmAfter },
};
writeFileSync(`${OUT}/summary.json`, `${JSON.stringify(report, null, 2)}\n`);

for (const r of results) {
  const { viewport: vp, checks, metrics: m, errors } = r;
  console.log(
    `${vp.name}  heroBottom=${m.hero.bottom}/${m.vh} stageBottom=${m.stage?.bottom} ` +
      `eyebrowTop=${m.eyebrow.top} navGap=${checks.navGap} fills=${checks.modelFillsStage}% ` +
      `fits=${checks.heroWithinViewport} stage=${checks.stageVisible} ctas=${checks.ctasVisible} ` +
      `overflow=${!checks.noHorizontalOverflow} errors=${errors.length}`,
  );
}
console.log(`reduced-motion stage transform: ${rmBefore.stage} -> ${rmAfter.stage}`);
