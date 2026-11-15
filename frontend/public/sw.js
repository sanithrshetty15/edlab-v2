const CACHE_NAME = "edwinslab-shell-v2";
const CACHE_PREFIX = "edwinslab-";
const APP_SHELL = "/index.html";
const BUILD_ASSETS = [];
const PRECACHE_URLS = [
  APP_SHELL,
  "/manifest.webmanifest",
  "/favicon.svg",
  "/icons/icon-192.png",
  "/icons/icon-512.png",
  "/icons/icon-maskable-512.png",
  "/icons/apple-touch-icon.png",
  ...BUILD_ASSETS,
];

const NEVER_CACHE_PREFIXES = ["/api/", "/downloads/"];
const APP_ROUTE_PATHS = ["/", "/login", "/forgot-password", "/app", "/explore"];

self.addEventListener("install", (event) => {
  event.waitUntil(
    (async () => {
      const cache = await caches.open(CACHE_NAME);
      await Promise.allSettled(
        PRECACHE_URLS.map((url) =>
          cache.add(new Request(url, { cache: "reload" })),
        ),
      );
    })(),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    (async () => {
      await self.clients.claim();
      const keys = await caches.keys();
      await Promise.all(
        keys
          .filter((key) => key.startsWith(CACHE_PREFIX) && key !== CACHE_NAME)
          .map((key) => caches.delete(key)),
      );
    })(),
  );
});

self.addEventListener("message", (event) => {
  if (event.data && event.data.type === "SKIP_WAITING") {
    self.skipWaiting();
  }
});

self.addEventListener("fetch", (event) => {
  const { request } = event;

  if (request.method !== "GET") return;

  let url;
  try {
    url = new URL(request.url);
  } catch {
    return;
  }

  if (url.origin !== self.location.origin) return;
  if (NEVER_CACHE_PREFIXES.some((prefix) => url.pathname.startsWith(prefix))) return;

  if (request.mode === "navigate") {
    event.respondWith(networkFirstNavigation(request));
    return;
  }

  if (isAppRoute(url.pathname)) return;

  if (url.pathname.startsWith("/assets/")) {
    event.respondWith(cacheFirstAsset(request));
    return;
  }

  if (request.destination === "document") return;

  event.respondWith(staleWhileRevalidate(request, event));
});

function isAppRoute(pathname) {
  return APP_ROUTE_PATHS.some(
    (route) => pathname === route || pathname.startsWith(`${route}/`),
  );
}

function isCacheableResponse(response) {
  return Boolean(response && response.ok && response.type === "basic");
}

function isHtmlResponse(response) {
  if (!isCacheableResponse(response)) return false;
  const contentType = response.headers.get("content-type") || "";
  return contentType.includes("text/html");
}

async function putInCache(cache, key, response) {
  try {
    await cache.put(key, response);
    return true;
  } catch {
    return false;
  }
}

async function networkFirstNavigation(request) {
  const cache = await caches.open(CACHE_NAME);

  try {
    const response = await fetch(request);

    if (isHtmlResponse(response)) {
      await putInCache(cache, new Request(APP_SHELL), response.clone());
    }

    return response;
  } catch {
    const cached = await cache.match(new Request(APP_SHELL));
    if (cached) return cached;

    return offlineDocumentResponse();
  }
}

async function cacheFirstAsset(request) {
  const cache = await caches.open(CACHE_NAME);
  const cached = await cache.match(request);
  if (cached) return cached;

  try {
    const response = await fetch(request);

    if (isCacheableResponse(response)) {
      await putInCache(cache, request, response.clone());
    }

    return response;
  } catch {
    return Response.error();
  }
}

async function staleWhileRevalidate(request, event) {
  const cache = await caches.open(CACHE_NAME);
  const cached = await cache.match(request);

  const revalidate = fetch(request)
    .then(async (response) => {
      if (isCacheableResponse(response)) {
        await putInCache(cache, request, response.clone());
      }
      return response;
    })
    .catch(() => null);

  if (cached) {
    event.waitUntil(revalidate);
    return cached;
  }

  const fresh = await revalidate;
  return fresh || Response.error();
}

function offlineDocumentResponse() {
  return new Response(
    "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>Edwins Lab — Offline</title><style>body{margin:0;min-height:100vh;display:grid;place-items:center;background:#1C1C1C;color:#fff;font-family:system-ui,sans-serif;text-align:center}main{padding:2rem;max-width:32rem}h1{font-size:1.5rem}p{line-height:1.7;color:#c7c7c7}</style></head><body><main><h1>Edwins Lab is offline</h1><p>Reconnect to the internet, then reload this page to continue.</p></main></body></html>",
    {
      status: 503,
      statusText: "Offline",
      headers: { "Content-Type": "text/html; charset=utf-8" },
    },
  );
}
