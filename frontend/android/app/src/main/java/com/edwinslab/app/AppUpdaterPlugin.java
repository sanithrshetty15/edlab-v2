package com.edwinslab.app;

import android.content.ActivityNotFoundException;
import android.content.Intent;
import android.content.pm.PackageInfo;
import android.content.pm.PackageManager;
import android.net.Uri;
import android.os.Build;
import android.provider.Settings;

import androidx.core.content.FileProvider;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import java.io.BufferedInputStream;
import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.security.MessageDigest;
import java.util.Locale;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicBoolean;

/**
 * Edwins Lab release updater.
 *
 * Downloads a release APK over HTTPS, verifies its SHA-256 checksum, checks that
 * the archive belongs to this application, then asks Android's package
 * installer to install it. Installation is never silent: the user always
 * confirms it in the system installer.
 */
@CapacitorPlugin(name = "AppUpdater")
public class AppUpdaterPlugin extends Plugin {

    private static final String APK_DIR = "apk";
    private static final int MAX_REDIRECTS = 5;
    private static final int CONNECT_TIMEOUT_MS = 15000;
    private static final int READ_TIMEOUT_MS = 45000;

    private final ExecutorService executor = Executors.newSingleThreadExecutor();
    private final AtomicBoolean downloading = new AtomicBoolean(false);

    @PluginMethod
    public void getAppInfo(PluginCall call) {
        try {
            PackageManager packageManager = getContext().getPackageManager();
            PackageInfo info = packageManager.getPackageInfo(getContext().getPackageName(), 0);

            JSObject result = new JSObject();
            result.put("packageId", getContext().getPackageName());
            result.put("versionName", info.versionName);
            result.put("versionCode", versionCode(info));
            call.resolve(result);
        } catch (PackageManager.NameNotFoundException error) {
            call.reject("The installed application could not be read.", "app_info_unavailable", error);
        }
    }

    @PluginMethod
    public void downloadApk(PluginCall call) {
        String url = call.getString("url");
        String expectedSha = call.getString("sha256");
        String requestedName = call.getString("fileName", "edwinslab-update.apk");

        if (url == null || !url.startsWith("https://")) {
            call.reject("Update downloads must use HTTPS.", "insecure_url");
            return;
        }

        String normalizedSha = expectedSha == null
            ? ""
            : expectedSha.replaceAll("[^A-Fa-f0-9]", "").toLowerCase(Locale.US);
        if (normalizedSha.length() != 64) {
            call.reject("A SHA-256 checksum is required to verify the update.", "invalid_checksum");
            return;
        }

        if (!downloading.compareAndSet(false, true)) {
            call.reject("An update download is already running.", "download_in_progress");
            return;
        }

        String safeName = requestedName.replaceAll("[^A-Za-z0-9._-]", "_");
        if (!safeName.toLowerCase(Locale.US).endsWith(".apk")) {
            safeName = safeName + ".apk";
        }
        final String fileName = safeName;
        final String sha256 = normalizedSha;

        executor.execute(() -> {
            File dir = new File(getContext().getCacheDir(), APK_DIR);
            File partial = new File(dir, fileName + ".part");
            File target = new File(dir, fileName);

            try {
                if (!dir.exists() && !dir.mkdirs()) {
                    throw new IOException("The update folder could not be created.");
                }

                downloadToFile(url, partial);

                String actualSha = sha256(partial);
                if (!actualSha.equals(sha256)) {
                    //noinspection ResultOfMethodCallIgnored
                    partial.delete();
                    emitProgress(0, 0, "failed");
                    call.reject(
                        "The downloaded file failed its SHA-256 integrity check.",
                        "checksum_mismatch"
                    );
                    return;
                }

                PackageManager packageManager = getContext().getPackageManager();
                PackageInfo archive = packageManager.getPackageArchiveInfo(partial.getPath(), 0);
                if (archive == null) {
                    //noinspection ResultOfMethodCallIgnored
                    partial.delete();
                    call.reject("The downloaded APK is corrupted.", "corrupted_apk");
                    return;
                }
                if (!getContext().getPackageName().equals(archive.packageName)) {
                    //noinspection ResultOfMethodCallIgnored
                    partial.delete();
                    call.reject(
                        "The downloaded APK belongs to a different application.",
                        "wrong_package"
                    );
                    return;
                }

                if (target.exists() && !target.delete()) {
                    throw new IOException("The previous update file could not be replaced.");
                }
                if (!partial.renameTo(target)) {
                    throw new IOException("The verified update file could not be saved.");
                }

                emitProgress(100, target.length(), "verified");

                JSObject result = new JSObject();
                result.put("filePath", target.getAbsolutePath());
                result.put("sizeBytes", target.length());
                result.put("sha256", actualSha);
                result.put("versionCode", versionCode(archive));
                result.put("versionName", archive.versionName);
                call.resolve(result);
            } catch (Exception error) {
                //noinspection ResultOfMethodCallIgnored
                partial.delete();
                emitProgress(0, 0, "failed");
                call.reject(
                    error.getMessage() == null ? "The update download failed." : error.getMessage(),
                    "download_failed",
                    error
                );
            } finally {
                downloading.set(false);
            }
        });
    }

    @PluginMethod
    public void installApk(PluginCall call) {
        String filePath = call.getString("filePath");
        if (filePath == null || filePath.isEmpty()) {
            call.reject("The downloaded update could not be found.", "file_missing");
            return;
        }

        File file = new File(filePath);
        try {
            File cacheRoot = new File(getContext().getCacheDir(), APK_DIR).getCanonicalFile();
            File candidate = file.getCanonicalFile();
            if (!candidate.getPath().startsWith(cacheRoot.getPath())
                || !candidate.isFile()
                || candidate.length() == 0) {
                call.reject("The downloaded update could not be found.", "file_missing");
                return;
            }
            file = candidate;
        } catch (IOException error) {
            call.reject("The downloaded update could not be read.", "file_missing", error);
            return;
        }

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            PackageManager packageManager = getContext().getPackageManager();
            if (!packageManager.canRequestPackageInstalls()) {
                openInstallPermissionSettings();
                call.reject(
                    "Allow Edwins Lab to install unknown apps, then try the update again.",
                    "install_permission_required"
                );
                return;
            }
        }

        try {
            Uri uri = FileProvider.getUriForFile(
                getContext(),
                getContext().getPackageName() + ".fileprovider",
                file
            );

            Intent intent = new Intent(Intent.ACTION_VIEW);
            intent.setDataAndType(uri, "application/vnd.android.package-archive");
            intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            getContext().startActivity(intent);

            JSObject result = new JSObject();
            result.put("openedInstaller", true);
            call.resolve(result);
        } catch (ActivityNotFoundException error) {
            call.reject("No Android package installer is available.", "installer_unavailable", error);
        } catch (Exception error) {
            call.reject("The Android package installer could not be opened.", "install_failed", error);
        }
    }

    private void openInstallPermissionSettings() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return;

        try {
            Intent intent = new Intent(
                Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,
                Uri.parse("package:" + getContext().getPackageName())
            );
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            getContext().startActivity(intent);
        } catch (ActivityNotFoundException ignored) {
            // The settings screen is optional; the caller still reports the error.
        }
    }

    private void downloadToFile(String url, File destination) throws Exception {
        HttpURLConnection connection = null;
        try {
            connection = openHttpsConnection(url);
            int status = connection.getResponseCode();
            if (status < 200 || status >= 300) {
                throw new IOException("The update server returned HTTP " + status + ".");
            }

            long total = connection.getContentLength();
            MessageDigest digest = MessageDigest.getInstance("SHA-256");

            try (
                InputStream input = new BufferedInputStream(connection.getInputStream());
                FileOutputStream output = new FileOutputStream(destination)
            ) {
                byte[] buffer = new byte[16384];
                long received = 0;
                long lastEmit = 0;
                int lastPercent = -1;
                int read;

                while ((read = input.read(buffer)) != -1) {
                    output.write(buffer, 0, read);
                    digest.update(buffer, 0, read);
                    received += read;

                    if (total > 0) {
                        int percent = (int) Math.min(99, (received * 100) / total);
                        long now = System.currentTimeMillis();
                        if (percent != lastPercent && (percent == 99 || now - lastEmit >= 200)) {
                            lastPercent = percent;
                            lastEmit = now;
                            emitProgress(percent, received, "downloading");
                        }
                    }
                }
                output.flush();
            }
        } finally {
            if (connection != null) connection.disconnect();
        }
    }

    private HttpURLConnection openHttpsConnection(String url) throws IOException {
        String current = url;

        for (int hop = 0; hop <= MAX_REDIRECTS; hop++) {
            if (!current.startsWith("https://")) {
                throw new IOException("Update downloads must use HTTPS.");
            }

            HttpURLConnection connection = (HttpURLConnection) new URL(current).openConnection();
            connection.setInstanceFollowRedirects(false);
            connection.setConnectTimeout(CONNECT_TIMEOUT_MS);
            connection.setReadTimeout(READ_TIMEOUT_MS);
            connection.setUseCaches(false);
            connection.setRequestProperty("Accept", "application/vnd.android.package-archive");

            int status = connection.getResponseCode();
            if (status == HttpURLConnection.HTTP_MOVED_PERM
                || status == HttpURLConnection.HTTP_MOVED_TEMP
                || status == HttpURLConnection.HTTP_SEE_OTHER
                || status == 307
                || status == 308) {
                String location = connection.getHeaderField("Location");
                connection.disconnect();
                if (location == null || location.isEmpty()) {
                    throw new IOException("The update server redirected the download incorrectly.");
                }
                current = new URL(new URL(current), location).toString();
                continue;
            }

            return connection;
        }

        throw new IOException("The update server redirected the download too many times.");
    }

    private void emitProgress(int percent, long receivedBytes, String phase) {
        JSObject payload = new JSObject();
        payload.put("progress", percent);
        payload.put("receivedBytes", receivedBytes);
        payload.put("phase", phase);
        notifyListeners("downloadProgress", payload);
    }

    private static long versionCode(PackageInfo info) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            return info.getLongVersionCode();
        }
        return info.versionCode;
    }

    private static String sha256(File file) throws Exception {
        MessageDigest digest = MessageDigest.getInstance("SHA-256");
        try (InputStream input = new BufferedInputStream(new java.io.FileInputStream(file))) {
            byte[] buffer = new byte[16384];
            int read;
            while ((read = input.read(buffer)) != -1) {
                digest.update(buffer, 0, read);
            }
        }

        StringBuilder hex = new StringBuilder(64);
        for (byte value : digest.digest()) {
            hex.append(String.format(Locale.US, "%02x", value));
        }
        return hex.toString();
    }
}
