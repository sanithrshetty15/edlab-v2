package com.edwinslab.app;

import android.app.DownloadManager;
import android.content.Context;
import android.content.Intent;
import android.net.Uri;
import android.os.Environment;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

@CapacitorPlugin(name = "FileHandler")
public class FileHandlerPlugin extends Plugin {
    @PluginMethod
    public void openUrl(PluginCall call) {
        String url = call.getString("url");
        if (url == null || url.isEmpty()) {
            call.reject("A URL is required");
            return;
        }

        try {
            Intent intent = new Intent(Intent.ACTION_VIEW, Uri.parse(url));
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            getContext().startActivity(intent);
            call.resolve();
        } catch (Exception error) {
            call.reject("No app can open this link", error);
        }
    }

    @PluginMethod
    public void download(PluginCall call) {
        String url = call.getString("url");
        String filename = call.getString("filename", "edwinslab-download");
        if (url == null || url.isEmpty()) {
            call.reject("A download URL is required");
            return;
        }

        try {
            String safeName = filename.replaceAll("[^A-Za-z0-9._-]", "_");
            DownloadManager manager =
                (DownloadManager) getContext().getSystemService(Context.DOWNLOAD_SERVICE);

            DownloadManager.Request request = new DownloadManager.Request(Uri.parse(url));
            request.setTitle(safeName);
            request.setDescription("Edwins Lab download");
            request.setNotificationVisibility(
                DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED
            );
            request.setDestinationInExternalFilesDir(
                getContext(),
                Environment.DIRECTORY_DOWNLOADS,
                safeName
            );
            request.setAllowedOverMetered(true);
            request.setAllowedOverRoaming(true);

            long id = manager.enqueue(request);

            JSObject result = new JSObject();
            result.put("downloadId", id);
            result.put("filename", safeName);
            call.resolve(result);
        } catch (Exception error) {
            call.reject("Download could not be started", error);
        }
    }
}
