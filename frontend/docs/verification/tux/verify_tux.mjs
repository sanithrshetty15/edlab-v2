import { createRequire } from "node:module";
import fs from "node:fs";

// Resolved through NODE_PATH so the script can reuse a Playwright install that
// is not a dependency of the site itself.
const require = createRequire(import.meta.url);
const { chromium } = require("playwright");

const URL = "http://127.0.0.1:4173/";
const OUT = process.argv[2] || "/tmp/tux-verify";
fs.mkdirSync(OUT, { recursive: true });

const browser = await chromium.launch({
  executablePath: "/usr/bin/google-chrome-stable",
  headless: true,
  args: ["--enable-unsafe-swiftshader", "--hide-scrollbars"],
});

async function inspect(page) {
  return page.evaluate(() => {
    const canvas = document.querySelector(".tux-stage canvas");
    const stage = document.querySelector(".tux-stage__canvas");
    const rect = canvas?.getBoundingClientRect();
    const gl = canvas?.getContext("webgl2") || canvas?.getContext("webgl");
    return {
      hasCanvas: Boolean(canvas),
      canvasBacking: canvas ? [canvas.width, canvas.height] : null,
      canvasCss: rect ? [Math.round(rect.width), Math.round(rect.height)] : null,
      glLost: gl ? gl.isContextLost() : null,
      artworkOverlay: Boolean(document.querySelector(".tux-stage__fallback--overlay")),
      artworkFallback: Boolean(document.querySelector(".tux-stage__fallback")),
      stageRect: stage
        ? Object.fromEntries(
            ["x", "y", "width", "height"].map((k) => [k, Math.round(stage.getBoundingClientRect()[k])]),
          )
        : null,
      docScrollWidth: document.documentElement.scrollWidth,
      docClientWidth: document.documentElement.clientWidth,
    };
  });
}

async function run(name, viewport, opts = {}) {
  const context = await browser.newContext({
    viewport,
    deviceScaleFactor: 1,
    reducedMotion: opts.reducedMotion || "no-preference",
  });
  const page = await context.newPage();
  const logs = [];
  page.on("console", (m) => {
    if (["error", "warning"].includes(m.type())) logs.push(`${m.type()}: ${m.text()}`);
  });
  page.on("pageerror", (e) => logs.push(`pageerror: ${e.message}`));
  page.on("requestfailed", (r) => logs.push(`requestfailed: ${r.url()} ${r.failure()?.errorText}`));

  if (opts.abortModel) {
    await page.route("**/models/tux.glb", (route) => route.abort());
  }

  await page.goto(URL, { waitUntil: "networkidle" });
  await page
    .waitForFunction(() => !document.querySelector('#root > [role="presentation"]'), null, {
      timeout: 15000,
    })
    .catch(() => logs.push("note: intro overlay still present after 15s"));

  let modelReady = false;
  try {
    await page.waitForFunction(
      () =>
        document.querySelector(".tux-stage canvas") &&
        !document.querySelector(".tux-stage__fallback--overlay"),
      null,
      { timeout: 25000 },
    );
    modelReady = true;
  } catch {
    /* handled below */
  }

  await page.waitForTimeout(opts.abortModel ? 3000 : 1500);

  const info = await inspect(page);
  const stage = page.locator(".tux-stage__canvas");
  await page.screenshot({ path: `${OUT}/${name}-viewport.png` });
  if (opts.heroShot) {
    await page.locator('[data-hero-stage]').screenshot({ path: `${OUT}/${name}-hero-stage.png` });
  }
  if (!opts.abortModel && (await stage.count())) {
    await stage.screenshot({ path: `${OUT}/${name}-tux-stage.png` });
  }

  console.log(`\n=== ${name} (${viewport.width}x${viewport.height}) ===`);
  console.log("modelReady:", modelReady);
  console.log(JSON.stringify(info));
  console.log("console issues:", logs.length ? logs : "none");
  await context.close();
  return { modelReady, info, logs };
}

const desktop = await run("desktop", { width: 1440, height: 900 }, { heroShot: true });
const mobile = await run("mobile", { width: 390, height: 844 }, { heroShot: true });
const reduced = await run("reduced-motion", { width: 1280, height: 800 }, { reducedMotion: "reduce" });
const aborted = await run("error-fallback", { width: 1280, height: 800 }, { abortModel: true });

await browser.close();
fs.writeFileSync(
  `${OUT}/summary.json`,
  JSON.stringify({ desktop, mobile, reduced, aborted }, null, 2),
);
