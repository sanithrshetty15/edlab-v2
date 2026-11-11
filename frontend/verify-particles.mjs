import { chromium } from "playwright";
import { mkdir } from "node:fs/promises";

const OUT = "verification";
await mkdir(OUT, { recursive: true });

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

const consoleErrors = [];
page.on("console", (msg) => {
  if (msg.type() === "error") consoleErrors.push(msg.text());
});
page.on("pageerror", (err) => consoleErrors.push(`pageerror: ${err}`));

await page.goto("http://localhost:5178/", { waitUntil: "networkidle" });
await page.waitForTimeout(6000); // let the intro overlay finish

await page.locator("#testimonials").scrollIntoViewIfNeeded();
await page.locator(".particle-stage").scrollIntoViewIfNeeded();
await page.waitForFunction(
  () => document.querySelector(".particle-stage__canvas")?.dataset.ready === "true",
  null,
  { timeout: 20000 },
);
// Let the intro finish and the intersection observer mark the stage visible so
// the real animation loop is running before shapes are sampled.
await page.waitForTimeout(2500);

const canvasInfo = await page.evaluate(() => {
  const mount = document.querySelector(".particle-stage__canvas");
  const canvas = mount.querySelector("canvas");
  const rect = canvas.getBoundingClientRect();
  return {
    width: Math.round(rect.width),
    height: Math.round(rect.height),
    backingWidth: canvas.width,
    backingHeight: canvas.height,
    shape: mount.dataset.shape,
    label: document.querySelector(".particle-stage__shape")?.textContent,
  };
});
console.log("CANVAS", JSON.stringify(canvasInfo));

const captured = [];
for (const key of ["globe", "tux", "text"]) {
  await page.waitForFunction(
    (k) => {
      const mount = document.querySelector(".particle-stage__canvas");
      return mount?.dataset.shape === k && mount?.dataset.phase === "hold";
    },
    key,
    { timeout: 40000, polling: "raf" },
  );
  await page.waitForTimeout(400);
  const label = await page
    .locator(".particle-stage__shape")
    .textContent();
  const shape = await page.evaluate(
    () => document.querySelector(".particle-stage__canvas").dataset.shape,
  );
  await page
    .locator(".particle-stage")
    .screenshot({ path: `${OUT}/particle-${key}.png` });
  captured.push({ key, shape, label });
  console.log("CAPTURED", key, "data-shape=", shape, "hud=", label);
}

await page.locator("#testimonials").screenshot({
  path: `${OUT}/testimonials-section.png`,
});

// Confirm motion: particle-stage screenshot must change between two moments.
const frameA = await page.locator(".particle-stage").screenshot();
await page.waitForTimeout(700);
const frameB = await page.locator(".particle-stage").screenshot();
console.log(
  "FRAME_BYTES",
  frameA.length,
  frameB.length,
  "identical=",
  frameA.equals(frameB),
);

const download = await page.evaluate(() => {
  const hero = document.querySelector("#download .phone-hero");
  const screen = document.querySelector("#download .phone-screen");
  const tuxNodes = document.querySelectorAll(
    "#download .tux-artwork, #download .phone-tux, #download [class*='tux' i]",
  );
  const cta = document.querySelector("#download .download-panel a[href]");
  const visibleDownloadApp = [...document.querySelectorAll("a, button")]
    .filter((el) => el.offsetParent !== null)
    .filter((el) => /download app/i.test(el.textContent || ""))
    .map((el) => (el.textContent || "").replace(/\s+/g, " ").trim());
  return {
    heroChildren: hero ? hero.children.length : -1,
    heroMedia: hero ? hero.querySelectorAll("img, svg, canvas").length : -1,
    phoneScreenText: screen
      ? screen.textContent.replace(/\s+/g, " ").trim()
      : null,
    heroText: hero ? hero.textContent.replace(/\s+/g, " ").trim() : null,
    tuxNodeCount: tuxNodes.length,
    ctaLabel: cta ? cta.textContent.replace(/\s+/g, " ").trim() : null,
    ctaHref: cta ? cta.getAttribute("href") : null,
    visibleDownloadApp,
  };
});
console.log("DOWNLOAD", JSON.stringify(download, null, 2));

const testimonials = await page.evaluate(() => ({
  tuxNodes: document.querySelectorAll(
    "#testimonials .tux-artwork, #testimonials [class*='tux' i]",
  ).length,
  images: document.querySelectorAll("#testimonials img").length,
  canvases: document.querySelectorAll("#testimonials canvas").length,
  heading: document.querySelector("#testimonials-title")?.textContent,
}));
console.log("TESTIMONIALS", JSON.stringify(testimonials));

await page.locator("#download").scrollIntoViewIfNeeded();
await page.waitForTimeout(1800); // let the scroll reveal finish
await page.locator("#download").screenshot({ path: `${OUT}/download-section.png` });

// The single Download app CTA must navigate to the download section.
await page.evaluate(() => window.scrollTo(0, 0));
await page.waitForTimeout(600);
await page.getByRole("link", { name: /download app/i }).first().click();
await page.waitForTimeout(1200);
const navResult = await page.evaluate(() => ({
  hash: window.location.hash,
  downloadTop: Math.round(
    document.querySelector("#download").getBoundingClientRect().top,
  ),
}));
console.log("DOWNLOAD_CTA_NAV", JSON.stringify(navResult));

// Mobile pass
await page.setViewportSize({ width: 390, height: 844 });
await page.waitForTimeout(1500);
await page.locator("#testimonials").scrollIntoViewIfNeeded();
await page.locator(".particle-stage").scrollIntoViewIfNeeded();
await page.waitForTimeout(2000);
await page.waitForFunction(
  () => {
    const mount = document.querySelector(".particle-stage__canvas");
    return mount?.dataset.shape === "tux" && mount?.dataset.phase === "hold";
  },
  null,
  { timeout: 30000, polling: "raf" },
);
await page.waitForTimeout(300);
await page.locator(".particle-stage").screenshot({
  path: `${OUT}/particle-tux-mobile.png`,
});
await page.locator("#testimonials").screenshot({
  path: `${OUT}/testimonials-mobile.png`,
});
const mobile = await page.evaluate(() => {
  const canvas = document.querySelector(".particle-stage canvas");
  const rect = canvas.getBoundingClientRect();
  return {
    width: Math.round(rect.width),
    height: Math.round(rect.height),
    overflow: document.documentElement.scrollWidth - window.innerWidth,
    shape: document.querySelector(".particle-stage__canvas").dataset.shape,
  };
});
console.log("MOBILE", JSON.stringify(mobile));

console.log("CONSOLE_ERRORS", JSON.stringify(consoleErrors));
console.log("CAPTURED_ORDER", JSON.stringify(captured));

await browser.close();
