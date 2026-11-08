import { chromium } from "playwright";
import { mkdirSync, writeFileSync } from "node:fs";

const BASE = process.env.BASE_URL || "http://localhost:5174";
const OUT = "docs/verification/explore/after";
mkdirSync(OUT, { recursive: true });

const browser = await chromium.launch({ channel: "chrome", headless: true });
const summary = { sizes: [], gallery: {}, slideshow: {}, footer: {}, console: [] };

const sizes = [
  ["1440", 1440, 900],
  ["1024", 1024, 768],
  ["768", 768, 1024],
  ["390", 390, 844],
  ["320", 320, 568],
];

for (const [label, width, height] of sizes) {
  const page = await browser.newPage({ viewport: { width, height } });
  const logs = [];
  page.on("console", (msg) => {
    if (["error", "warning"].includes(msg.type())) logs.push(`${msg.type()}: ${msg.text()}`);
  });
  page.on("pageerror", (err) => logs.push(`pageerror: ${err.message}`));

  await page.goto(`${BASE}/explore`, { waitUntil: "networkidle" });
  await page.evaluate(() => document.fonts?.ready);
  await page.waitForTimeout(700);

  await page.screenshot({ path: `${OUT}/${label}-top.png` });

  await page.locator(".explore-story__media").scrollIntoViewIfNeeded();
  await page.waitForTimeout(900);
  await page.screenshot({ path: `${OUT}/${label}-story.png` });

  await page.locator("#gallery").scrollIntoViewIfNeeded();
  await page.waitForTimeout(900);
  await page.screenshot({ path: `${OUT}/${label}-gallery.png` });

  await page.locator(".explore-footer").scrollIntoViewIfNeeded();
  await page.waitForTimeout(1600);
  await page.screenshot({ path: `${OUT}/${label}-footer.png` });

  const metrics = await page.evaluate(() => {
    const ring = document.querySelector(".explore-carousel");
    const cards = [
      ...document.querySelectorAll("[data-ring-card]"),
      ...document.querySelectorAll(".explore-mobile-card__media"),
      ...document.querySelectorAll(".explore-static-card__media"),
    ];
    const slide = document.querySelector(".explore-slideshow__media");
    const footer = document.querySelector(".explore-footer");
    return {
      overflow: document.documentElement.scrollWidth > window.innerWidth,
      scrollWidth: document.documentElement.scrollWidth,
      innerWidth: window.innerWidth,
      ring: ring ? { ...ring.dataset, perspective: getComputedStyle(ring).perspective } : null,
      cards: cards.map((el) => {
        const image = el.querySelector("img");
        const r = el.getBoundingClientRect();
        return {
          layout: [el.offsetWidth, el.offsetHeight],
          ratio: +(el.offsetWidth / el.offsetHeight).toFixed(3),
          fit: image ? getComputedStyle(image).objectFit : null,
          position: image ? getComputedStyle(image).objectPosition : null,
        };
      }),
      slideshow: slide
        ? {
            box: [Math.round(slide.getBoundingClientRect().width), Math.round(slide.getBoundingClientRect().height)],
            label: document.querySelector(".explore-media-caption span")?.textContent,
          }
        : null,
      footer: footer
        ? {
            height: Math.round(footer.getBoundingClientRect().height),
            lines: document.querySelectorAll(".explore-footer__contour-lines path").length,
          }
        : null,
    };
  });

  summary.sizes.push({ label, width, height, ...metrics });
  summary.console.push(...logs.map((line) => `${label} ${line}`));
  await page.close();
}

/* Gallery scrub sweep at 1440. */
{
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  await page.goto(`${BASE}/explore`, { waitUntil: "networkidle" });
  await page.waitForTimeout(700);
  const info = await page.evaluate(() => {
    const el = document.querySelector("#gallery");
    return { top: el.getBoundingClientRect().top + window.scrollY, h: window.innerHeight };
  });
  const distance = info.h * 2.75;
  const sweep = [];
  for (const [name, progress] of [
    ["start", 0],
    ["q25", 0.25],
    ["mid", 0.5],
    ["q75", 0.75],
    ["end", 0.985],
  ]) {
    await page.evaluate(({ y }) => window.scrollTo(0, y), { y: info.top + distance * progress });
    await page.waitForTimeout(900);
    await page.screenshot({ path: `${OUT}/gallery-sweep-${name}.png` });
    sweep.push(
      await page.evaluate(() => {
        const ring = document.querySelector(".explore-carousel__ring");
        const cards = [...document.querySelectorAll("[data-ring-card]")];
        const scene = document.querySelector(".explore-carousel").getBoundingClientRect();
        return {
          active: document.querySelector(".explore-gallery__counter strong")?.textContent,
          ringTransform: getComputedStyle(ring).transform,
          visibleCardsInside: cards.filter((card) => {
            const r = card.getBoundingClientRect();
            return r.left >= scene.left - 1 && r.right <= scene.right + 1;
          }).length,
        };
      }),
    );
  }
  summary.gallery = { data: await page.evaluate(() => ({ ...document.querySelector(".explore-carousel").dataset })), sweep };
  await page.close();
}

/* Slideshow timing, transition, no-blank and pause. */
{
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  await page.goto(`${BASE}/explore`, { waitUntil: "networkidle" });
  await page.locator(".explore-story__media").scrollIntoViewIfNeeded();
  await page.waitForTimeout(500);

  const samples = [];
  const started = Date.now();
  while (Date.now() - started < 30000) {
    const sample = await page.evaluate(() => {
      const media = document.querySelector(".explore-slideshow__media");
      const layers = [...document.querySelectorAll(".explore-slideshow__layer")];
      const opacities = layers.map((layer) => +getComputedStyle(layer).opacity);
      return {
        label: document.querySelector(".explore-media-caption span")?.textContent,
        opacities,
        opacityMax: opacities.length ? Math.max(...opacities) : 1,
        box: [Math.round(media.getBoundingClientRect().width), Math.round(media.getBoundingClientRect().height)],
      };
    });
    samples.push({ t: Date.now() - started, ...sample });
    await page.waitForTimeout(100);
  }

  const changes = [];
  for (let i = 1; i < samples.length; i += 1) {
    if (samples[i].label !== samples[i - 1].label) {
      changes.push({ t: samples[i].t, label: samples[i].label });
    }
  }

  // Catch an in-progress transition for a screenshot.
  let transitionShot = false;
  for (let i = 0; i < 70; i += 1) {
    const state = await page.evaluate(() => {
      const layers = [...document.querySelectorAll(".explore-slideshow__layer")];
      return layers.map((layer) => ({
        opacity: +getComputedStyle(layer).opacity,
        clip: getComputedStyle(layer).clipPath,
      }));
    });
    if (state.some((layer) => layer.opacity > 0.45 && layer.opacity < 0.99 && layer.clip.includes("inset"))) {
      await page.screenshot({ path: `${OUT}/slideshow-transition.png` });
      transitionShot = true;
      break;
    }
    await page.waitForTimeout(100);
  }

  const labelsAfter = () =>
    page.evaluate(() => document.querySelector(".explore-media-caption span")?.textContent);

  // Wait until no crossfade is in flight before testing the offscreen pause.
  for (let i = 0; i < 80; i += 1) {
    const settled = await page.evaluate(() => {
      const layers = [...document.querySelectorAll(".explore-slideshow__layer")];
      const opacities = layers.map((layer) => +getComputedStyle(layer).opacity);
      return opacities.some((value) => value > 0.99) && opacities.some((value) => value < 0.2);
    });
    if (settled) break;
    await page.waitForTimeout(100);
  }

  const offscreenStart = await labelsAfter();
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.waitForTimeout(8000);
  const offscreenLabel = await labelsAfter();
  await page.locator(".explore-story__media").scrollIntoViewIfNeeded();
  await page.waitForTimeout(6000);
  const resumedLabel = await labelsAfter();

  summary.slideshow = {
    changes,
    uniqueLabels: [...new Set(samples.map((s) => s.label))],
    blankFrames: samples.filter((s) => s.opacityMax < 0.35).length,
    constantBox: new Set(samples.map((s) => JSON.stringify(s.box))).size === 1,
    box: samples[0].box,
    offscreenHold: { before: offscreenStart, after: offscreenLabel },
    offscreenPaused: offscreenStart === offscreenLabel,
    resumedChanged: resumedLabel !== offscreenLabel,
    resumedLabel,
    transitionShot,
  };
  await page.close();
}

/* Reduced motion. */
{
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, reducedMotion: "reduce" });
  await page.goto(`${BASE}/explore`, { waitUntil: "networkidle" });
  await page.locator(".explore-story__media").scrollIntoViewIfNeeded();
  await page.waitForTimeout(500);
  const first = await page.evaluate(() => document.querySelector(".explore-media-caption span")?.textContent);
  await page.waitForTimeout(5000);
  const later = await page.evaluate(() => document.querySelector(".explore-media-caption span")?.textContent);
  await page.locator("#gallery").scrollIntoViewIfNeeded();
  await page.waitForTimeout(400);
  await page.screenshot({ path: `${OUT}/reduced-gallery.png` });
  await page.locator(".explore-footer").scrollIntoViewIfNeeded();
  await page.waitForTimeout(800);
  await page.screenshot({ path: `${OUT}/reduced-footer.png` });
  const reduced = await page.evaluate((firstLabel) => {
    const card = document.querySelector(".explore-static-card__media");
    const path = document.querySelector(".explore-footer__contour-lines path");
    return {
      stories: firstLabel,
      staticGrid: !!card,
      cardRatio: card ? +(card.offsetWidth / card.offsetHeight).toFixed(3) : null,
      ring: !!document.querySelector(".explore-carousel"),
      slideshowLayers: document.querySelectorAll(".explore-slideshow__layer").length,
      firstPath: path?.getAttribute("d"),
      overflow: document.documentElement.scrollWidth > window.innerWidth,
      footerWordmark: getComputedStyle(document.querySelector("[data-footer-wordmark]")).transform,
      footerLegal: getComputedStyle(document.querySelector("[data-footer-legal]")).opacity,
    };
  }, first);
  await page.waitForTimeout(1500);
  reduced.firstPathStatic =
    reduced.firstPath ===
    (await page.evaluate(() => document.querySelector(".explore-footer__contour-lines path")?.getAttribute("d")));
  reduced.slideshowStatic = first === later;
  summary.reduced = reduced;
  await page.close();
}

/* Footer entrance, contour pause and frame-time sample. */
{
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  await page.goto(`${BASE}/explore`, { waitUntil: "networkidle" });
  await page.waitForTimeout(700);
  const top = await page.evaluate(
    () => document.querySelector(".explore-footer").getBoundingClientRect().top + window.scrollY,
  );
  await page.evaluate(({ y }) => window.scrollTo(0, y), { y: top - 850 });
  await page.waitForTimeout(500);
  await page.evaluate(({ y }) => window.scrollTo(0, y), { y: top - 660 });

  const entrance = [];
  for (let i = 0; i < 15; i += 1) {
    await page.waitForTimeout(120);
    entrance.push(
      await page.evaluate(() => {
        const q = (selector) => document.querySelector(selector);
        const style = (selector) => getComputedStyle(q(selector));
        const wordmark = q("[data-footer-wordmark]").getBoundingClientRect();
        const mask = q(".explore-footer__wordmark-mask").getBoundingClientRect();
        return {
          brand: +style("[data-footer-brand]").opacity,
          dividerScaleX: style("[data-footer-line]").transform,
          wordmarkTop: +(wordmark.top - mask.top).toFixed(1),
          nav: +style(".explore-footer__nav").opacity,
          statement: +style(".explore-footer__statement").opacity,
          bottom: +style(".explore-footer__back").opacity,
          legal: +style("[data-footer-legal]").opacity,
          contours: +style("[data-footer-contours]").opacity,
        };
      }),
    );
  }

  const contourPath = () =>
    page.evaluate(() =>
      document.querySelector(".explore-footer__contour-lines path").getAttribute("d"),
    );
  const pathA = await contourPath();
  await page.waitForTimeout(1500);
  const pathB = await contourPath();
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.waitForTimeout(600);
  const pathC = await contourPath();
  await page.waitForTimeout(1500);
  const pathD = await contourPath();
  await page.locator(".explore-footer").scrollIntoViewIfNeeded();
  await page.waitForTimeout(1000);
  const pathE = await contourPath();

  const frameTimes = await page.evaluate(
    () =>
      new Promise((resolve) => {
        const times = [];
        let last = performance.now();
        let count = 0;
        function frame(now) {
          times.push(now - last);
          last = now;
          if ((count += 1) < 120) requestAnimationFrame(frame);
          else resolve(times);
        }
        requestAnimationFrame(frame);
      }),
  );

  summary.footer = {
    entrance,
    motion: {
      movingWhileVisible: pathA !== pathB,
      pausedOffscreen: pathC === pathD,
      resumed: pathD !== pathE,
    },
    frameTimeMs: {
      average: +(frameTimes.reduce((a, b) => a + b, 0) / frameTimes.length).toFixed(2),
      worst: +Math.max(...frameTimes).toFixed(2),
    },
  };
  await page.close();
}

await browser.close();
summary.console = [...new Set(summary.console)];
writeFileSync(`${OUT}/summary.json`, JSON.stringify(summary, null, 2));
console.log(JSON.stringify(summary, null, 2));
