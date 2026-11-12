#!/usr/bin/env bash
#
# Builds a signed Android release APK, verifies it, copies it into the public
# downloads folder and rewrites the release manifest.
#
# Usage:
#   ANDROID_KEYSTORE_PATH=... ANDROID_KEYSTORE_PASSWORD=... \
#   ANDROID_KEY_ALIAS=... ANDROID_KEY_PASSWORD=... \
#   scripts/release-android.sh \
#     --version-name 1.2 --version-code 3 \
#     --notes "Login now opens after the intro" \
#     --notes "New launcher icon" \
#     [--mandatory] [--minimum-supported-version 2]
#
# The manifest is written only after the APK is built and copied successfully.
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION_NAME=""
VERSION_CODE=""
NOTES=()
MANDATORY=0
MINIMUM=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version-name) VERSION_NAME="$2"; shift 2 ;;
    --version-code) VERSION_CODE="$2"; shift 2 ;;
    --notes) NOTES+=(--notes "$2"); shift 2 ;;
    --minimum-supported-version) MINIMUM="$2"; shift 2 ;;
    --mandatory) MANDATORY=1; shift ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

if [[ -z "$VERSION_NAME" || -z "$VERSION_CODE" ]]; then
  echo "Both --version-name and --version-code are required." >&2
  exit 1
fi

if [[ ! "$VERSION_CODE" =~ ^[0-9]+$ ]]; then
  echo "--version-code must be a positive integer." >&2
  exit 1
fi

if [[ -z "${ANDROID_KEYSTORE_PATH:-}" || ! -f "${ANDROID_KEYSTORE_PATH}" ]]; then
  echo "ANDROID_KEYSTORE_PATH must point to the existing release keystore." >&2
  echo "Never generate a new keystore for an update: Android rejects APKs signed with a different certificate." >&2
  exit 1
fi

for variable in ANDROID_KEYSTORE_PASSWORD ANDROID_KEY_ALIAS ANDROID_KEY_PASSWORD; do
  if [[ -z "${!variable:-}" ]]; then
    echo "$variable is not set." >&2
    exit 1
  fi
done

APK_NAME="EdwinsLab-${VERSION_NAME}.apk"
APK_SOURCE="android/app/build/outputs/apk/release/app-release.apk"
APK_TARGET="public/downloads/${APK_NAME}"

echo "==> Setting versionName ${VERSION_NAME} / versionCode ${VERSION_CODE}"
sed -i -E "s/versionCode [0-9]+/versionCode ${VERSION_CODE}/" android/app/build.gradle
sed -i -E "s/versionName \"[^\"]*\"/versionName \"${VERSION_NAME}\"/" android/app/build.gradle

echo "==> Building web bundle"
npm run build

echo "==> Syncing Capacitor Android project"
npx cap sync android

echo "==> Building signed release APK"
(cd android && ./gradlew :app:assembleRelease)

if [[ ! -f "$APK_SOURCE" ]]; then
  echo "Release APK was not produced at $APK_SOURCE" >&2
  exit 1
fi

echo "==> Verifying APK signature"
APKSIGNER="$(ls -d "${ANDROID_HOME:-/opt/android-sdk}"/build-tools/*/apksigner 2>/dev/null | sort -V | tail -1 || true)"
if [[ -n "$APKSIGNER" ]]; then
  "$APKSIGNER" verify --print-certs "$APK_SOURCE"
else
  echo "apksigner not found; skipping signature verification." >&2
fi

echo "==> Publishing APK to $APK_TARGET"
cp "$APK_SOURCE" "$APK_TARGET"

echo "==> Updating release manifest"
MANIFEST_ARGS=(
  --apk "$APK_TARGET"
  --version-name "$VERSION_NAME"
  --version-code "$VERSION_CODE"
  --minimum-supported-version "$MINIMUM"
)
if [[ "$MANDATORY" == "1" ]]; then
  MANIFEST_ARGS+=(--mandatory)
fi
node scripts/apk-manifest.mjs "${MANIFEST_ARGS[@]}" "${NOTES[@]}"

cat <<SUMMARY

Release built and staged:
  APK      $APK_TARGET
  Manifest public/downloads/edwinslab-apk.json

Next steps:
  1. Commit the APK, manifest and the bumped android/app/build.gradle.
  2. Deploy the frontend so /downloads/${APK_NAME} and the manifest are live on Vercel.
  3. Verify the manifest is reachable over HTTPS before announcing the release.
SUMMARY
