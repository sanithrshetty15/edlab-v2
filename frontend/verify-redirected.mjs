import { chromium } from "playwright";
import { mkdir } from "node:fs/promises";

const OUT = "verification2";
await mkdir(OUT, { recursive: true });

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

const consoleErrors = [];
page.on("console", (msg) => {
  if (msg.type() === "error") consoleErrors.push(msg.text());
});
page.on("pageerror", (err) => consoleErrors.push(`pageerror: ${err}`));

await page.goto("http://localhost:5179/", { waitUntil: "networkidle" });
await page.waitForTimeout(4000);

await page.locator("#linux").scrollIntoViewIfNeeded();
await page.waitForTimeout(2000);

// === CHECK 1: Testimonials has NO particle animation ===
const testimonialsHasCanvas = await page.locator("#testimonials canvas").count();
const testimonialsShape = await page.locator(".particle-stage__canvas").getAttribute("data-shape");
console.log("CHECK1 - Testimonials canvas count:", testimonialsHasCanvas);
console.log("CHECK1 - Testimonials data-shape:", testimonialsShape);

// === CHECK 2: Linux/About has particle animation ===
const linuxHasCanvas = await page.locator("#linux canvas").count();
const linuxShape = await page.locator(".particle-stage__canvas").getAttribute("data-shape");
console.log("CHECK2 - Linux canvas count:", linuxHasCanvas);
console.log("CHECK2 - Linux data-shape:", linuxShape);

// === CHECK 3: Verify shape cycles through globe → tux → text ===
await page.waitForFunction(
  () => document.querySelector(".particle-stage__canvas")?.dataset.shape === "globe",
  "globe",
  { timeout: 30000, polling: "raf" },
);
await page.waitForTimeout(800);
await page.locator(".particle-stage").screenshot({ path: `${OUT}/linux-globe.png` });
console.log("SCREENSHOT", `${OUT}/linux-globe.png`);

await page.waitForFunction(
  () => document.querySelector(".particle-stage__canvas")?.dataset.shape === "tux",
  "tux",
  { timeout: 30000, polling: "raf" },
);
await page.waitForTimeout(800);
await page.locator(".particle-stage").screenshot({ path: `${OUT}/linux-tux.png` });
console.log("SCREENSHOT", `${OUT}/linux-tux.png`);

await page.waitForFunction(
  () => document.querySelector(".particle-stage__canvas")?.dataset.shape === "text",
  "text",
  { timeout: 30000, polling: "raf" },
);
await page.waitForTimeout(800);
await page.locator(".particle-stage").screenshot({ path: `${OUT}/linux-text.png` });
console.log("SCREENSHOT", `${OUT}/linux-text.png`);

// === CHECK 4: Phone mockup has no Tux ===
const downloadHasTux = await page.locator("#download .tux-artwork, #download [class*='tux' i]").count();
const phoneScreenText = await page.locator("#download .phone-screen").textContent();
console.log("CHECK4 - Download Tux nodes:", downloadHasTux);
console.log("CHECK4 - Phone screen text includes Welcome back:", phoneScreenText.includes("Welcome back"));

// === CHECK 5: Single Download CTA ===
const downloadCTAs = await page.locator("#download a[href], #download button").count();
const downloadLabels = await page.locator("#download a[href], #download button").allTextContents();
console.log("CHECK5 - Download CTA count:", downloadCTAs);
console.log("CHECK5 - Download labels:", downloadLabels);

// === CHECK 6: Linux section heading preserved ===
const linuxHeading = await page.locator("#linux-title").textContent();
console.log("CHECK6 - Linux title:", linuxHeading);

// === CHECK 7: Mobile layout ===
await page.setViewportSize({ width: 390, height: 844 });
await page.waitForTimeout(1000);
await page.locator("#linux").scrollIntoViewIfNeeded();
await page.waitForTimeout(1200);
const mobileShape = await page.locator(".particle-stage__canvas").getAttribute("data-shape");
const mobileCanvas = await page.locator("#linux canvas").count();
console.log("CHECK7 - Mobile shape:", mobileShape);
console.log("CHECK7 - Mobile canvas count:", mobileCanvas);

console.log("CONSOLE_ERRORS", JSON.stringify(consoleErrors));

await browser.close();