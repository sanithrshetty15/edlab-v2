import { readdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const frontendDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const distDir = path.join(frontendDir, "dist");
const assetsDir = path.join(distDir, "assets");
const swPath = path.join(distDir, "sw.js");
const marker = "const BUILD_ASSETS = [];";

const entries = await readdir(assetsDir, { withFileTypes: true });
const assets = entries
  .filter((entry) => entry.isFile() && /\.(js|css)$/.test(entry.name))
  .map((entry) => `/assets/${entry.name}`)
  .sort();

const sw = await readFile(swPath, "utf8");
if (!sw.includes(marker)) {
  throw new Error(`Missing ${marker} marker in ${swPath}`);
}

const replacement = `const BUILD_ASSETS = ${JSON.stringify(assets)};`;
await writeFile(swPath, sw.replace(marker, replacement));
console.log(`Injected ${assets.length} versioned assets into dist/sw.js`);
