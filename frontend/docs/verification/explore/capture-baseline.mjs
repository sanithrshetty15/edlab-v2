import { chromium } from "playwright";
import { mkdirSync } from "node:fs";

const base = process.env.BASE_URL || "http://localhost:5174";
const outDir = "docs/verification/explore/before";
mkdirSync(outDir, { recursive: true });

const sizes = [
  ["1440", 1440, 900],
  ["1024", 1024, 768],
  ["768", 768, 1024],
  ["390", 390, 844],
  ["320", 320, 568],
];

const browser = await chromium.launch({ channel: "chrome", headless: true });
const report = [];

for (const [label, width, height] of sizes) {
  const page = await browser.newPage({ viewport: { width, height } });
  const logs = [];
  page.on("console", (msg) => {
    if (["error", "warning"].includes(msg.type())) logs.push(`${msg.type()}: ${msg.text()}`);
  });
  page.on("pageerror", (err) => logs.push(`pageerror: ${err.message}`));

  await page.goto(`${base}/explore`, { waitUntil: "networkidle" });
  await page.evaluate(() => document.fonts?.ready);
  await page.waitForTimeout(1200);

  await page.screenshot({ path: `${outDir}/${label}-top.png` });

  const gallery = page.locator("#gallery");
  await gallery.scrollIntoViewIfNeeded();
  await page.waitForTimeout(700);
  await page.screenshot({ path: `${outDir}/${label}-gallery.png` });

  await page.locator(".explore-footer").scrollIntoViewIfNeeded();
  await page.waitForTimeout(1200);
  await page.screenshot({ path: `${outDir}/${label}-footer.png` });

  const metrics = await page.evaluate(() => {
    const firstCard = document.querySelector("[data-ring-card]");
    const cardRect = firstCard?.getBoundingClientRect();
    const staticCard = document.querySelector(".explore-static-card__media");
    const footer = document.querySelector(".explore-footer");
    return {
      scrollWidth: document.documentElement.scrollWidth,
      innerWidth: window.innerWidth,
      firstRingCard: cardRect ? { w: Math.round(cardRect.width), h: Math.round(cardRect.height) } : null,
      staticCard: staticCard
        ? { w: Math.round(staticCard.getBoundingClientRect().width), h: Math.round(staticCard.getBoundingClientRect().height) }
        : null,
      footerHeight: footer ? Math.round(footer.getBoundingClientRect().height) : null,
      cardCount: document.querySelectorAll("[data-ring-card]").length,
    };
  });

  report.push({ label, width, height, ...metrics, console: logs });
  await page.close();
}

await browser.close();
console.log(JSON.stringify(report, null, 2));
