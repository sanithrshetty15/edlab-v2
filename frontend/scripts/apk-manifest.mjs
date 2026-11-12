#!/usr/bin/env node
/**
 * Writes public/downloads/edwinslab-apk.json for a built APK.
 * Used by scripts/release-android.sh after the APK exists on disk.
 *
 * Run only after the APK has been built and copied into public/downloads/.
 */
import { createHash } from "node:crypto";
import { readFile, stat, writeFile } from "node:fs/promises";
import { basename, resolve } from "node:path";

const PUBLIC_ORIGIN = "https://edwinslab.vercel.app";
const PACKAGE_ID = "com.edwinslab.app";

function parseArgs(argv) {
  const args = { notes: [], mandatory: false, minimum: 0, minSdk: 23 };

  for (let index = 0; index < argv.length; index += 1) {
    const flag = argv[index];
    const value = argv[index + 1];

    switch (flag) {
      case "--apk":
        args.apk = value;
        index += 1;
        break;
      case "--version-name":
        args.versionName = value;
        index += 1;
        break;
      case "--version-code":
        args.versionCode = Number(value);
        index += 1;
        break;
      case "--notes":
        args.notes.push(value);
        index += 1;
        break;
      case "--min-sdk":
        args.minSdk = Number(value);
        index += 1;
        break;
      case "--minimum-supported-version":
        args.minimum = Number(value);
        index += 1;
        break;
      case "--mandatory":
        args.mandatory = true;
        break;
      case "--out":
        args.out = value;
        index += 1;
        break;
      default:
        throw new Error(`Unknown argument: ${flag}`);
    }
  }

  return args;
}

function requireValue(value, message) {
  if (value === undefined || value === null || value === "" || Number.isNaN(value)) {
    throw new Error(message);
  }
}

async function main() {
  const args = parseArgs(process.argv.slice(2));

  requireValue(args.apk, "Pass --apk <path to built APK>");
  requireValue(args.versionName, "Pass --version-name <versionName>");
  requireValue(args.versionCode, "Pass --version-code <versionCode>");

  const apkPath = resolve(args.apk);
  const apkName = basename(apkPath);
  if (!apkName.toLowerCase().endsWith(".apk")) {
    throw new Error("The release artifact must be an .apk file.");
  }

  const [buffer, info] = await Promise.all([readFile(apkPath), stat(apkPath)]);
  const sha256 = createHash("sha256").update(buffer).digest("hex");

  const manifest = {
    name: "Edwins Lab",
    packageId: PACKAGE_ID,
    versionName: String(args.versionName),
    versionCode: Number(args.versionCode),
    minSdk: Number(args.minSdk) || 23,
    apkUrl: `${PUBLIC_ORIGIN}/downloads/${apkName}`,
    file: `/downloads/${apkName}`,
    sizeBytes: info.size,
    sha256,
    releasedAt: new Date().toISOString().slice(0, 10),
    releaseNotes: args.notes.length
      ? args.notes
      : ["Maintenance and reliability improvements."],
    mandatoryUpdate: Boolean(args.mandatory),
    minimumSupportedVersion: Number(args.minimum) || 0,
  };

  const out = resolve(args.out || "public/downloads/edwinslab-apk.json");
  await writeFile(out, `${JSON.stringify(manifest, null, 2)}\n`, "utf8");

  console.log(`Manifest written to ${out}`);
  console.log(`  version ${manifest.versionName} (${manifest.versionCode})`);
  console.log(`  size    ${manifest.sizeBytes} bytes`);
  console.log(`  sha256  ${manifest.sha256}`);
}

main().catch((error) => {
  console.error(`apk-manifest: ${error.message}`);
  process.exit(1);
});
