# Android releases

The Android app (`frontend/android`, application ID `com.edwinslab.app`) installs
updates from a JSON manifest published with the website:

```
https://edwinslab.vercel.app/downloads/edwinslab-apk.json
```

The app checks the manifest after the intro, compares `versionCode` with the
installed build, and only offers an update when the manifest is newer.

## Manifest

```json
{
  "name": "Edwins Lab",
  "packageId": "com.edwinslab.app",
  "versionName": "1.2",
  "versionCode": 3,
  "minSdk": 23,
  "apkUrl": "https://edwinslab.vercel.app/downloads/EdwinsLab-1.2.apk",
  "file": "/downloads/EdwinsLab-1.2.apk",
  "sizeBytes": 7490089,
  "sha256": "<64 hex characters>",
  "releasedAt": "2026-11-13",
  "releaseNotes": ["What changed in this build"],
  "mandatoryUpdate": false,
  "minimumSupportedVersion": 0
}
```

- `apkUrl` must be HTTPS. The app refuses any other scheme.
- `sha256` is required; the app discards anything that does not match.
- `mandatoryUpdate: true`, or an installed `versionCode` below
  `minimumSupportedVersion`, hides the “Later” button.
- `file` is kept for the website’s download card; it must resolve to the same APK.

## Release checklist

1. Increment `versionCode` (always) and set a matching `versionName` in
   `android/app/build.gradle`.
2. Export the existing signing credentials (never create a new keystore):
   `ANDROID_KEYSTORE_PATH`, `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS`,
   `ANDROID_KEY_PASSWORD`.
3. Run:

   ```bash
   cd frontend
   scripts/release-android.sh \
     --version-name 1.2 --version-code 3 \
     --notes "Login opens directly after the intro" \
     --notes "New launcher icon"
   ```

4. The script builds the web bundle, syncs Capacitor, builds and signs the
   release APK, verifies the signature, copies it to
   `public/downloads/EdwinsLab-<versionName>.apk`, and only then rewrites the
   manifest with the real SHA-256 and size.
5. Commit the APK, the manifest and the bumped `build.gradle`, then deploy the
   frontend so both files are live on Vercel.
6. Install the new build once by hand and confirm the next check reports no
   update (remote `versionCode` equals installed).

Do not publish the manifest before the APK is uploaded: the app verifies the
download checksum and would reject a missing or mismatched file.

## Signing rules

- Keep the same keystore for every release. Android will not install an APK
  signed with a different certificate.
- Never commit keystores, passwords or private keys. The keystore lives outside
  the repository and is provided through environment variables.
- Without signing credentials `assembleRelease` produces an unsigned APK; the
  release script refuses to publish it.
- Updates are never silent. The app downloads and verifies the APK, then opens
  Android’s package installer, where the user confirms installation.

## Failure handling

| Situation | App behaviour |
| --- | --- |
| No internet | Update check is skipped quietly; the login page still opens |
| Manifest unavailable / invalid | No dialog; the app continues normally |
| Download fails | Dialog stays open with a retry action |
| Checksum mismatch / corrupted APK | File is deleted and the user is asked to retry |
| Unknown-app installs not allowed | Android settings opens; user can retry after granting |
| User cancels installer | Nothing is installed; the dialog remains available |
