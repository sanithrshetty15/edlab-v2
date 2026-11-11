const CACHE_NAME = "edwinslab-shell-v1";
const APP_SHELL = "/index.html";
const PRECACHE_URLS = [
  APP_SHELL,
  "/manifest.webmanifest",
  "/favicon.svg",
  "/icons/icon-192.png",
  "/icons/icon-512.png",
  "/icons/icon-maskable-512.png",
  "/icons/apple-touch-icon.png",
];

const NEVER_CACHE_PREFIXES = ["/api/", "/downloads/"];

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
      const keys = await caches.keys();
      await Promise.all(
        keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key)),
      );
      await self.clients.claim();
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

  if (request.destination === "document") return;

  event.respondWith(staleWhileRevalidate(request));
});

async function networkFirstNavigation(request) {
  const cache = await caches.open(CACHE_NAME);

  try {
    const response = await fetch(request);
    if (response && response.ok) {
      await cache.put(new Request(APP_SHELL), response.clone());
    }
    return response;
  } catch {
    const cached = await cache.match(new Request(APP_SHELL));
    if (cached) return cached;

    return new Response(
      "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>Edwins Lab — Offline</title><style>body{margin:0;min-height:100vh;display:grid;place-items:center;background:#1C1C1C;color:#fff;font-family:system-ui,sans-serif;text-align:center}main{padding:2rem;max-width:32rem}h1{font-size:1.5rem}code{color:#C5FF41}</style></head><body><main><h1>Edwins Lab is offline</h1><p>This device has not cached the app shell yet. Reconnect and open the site once, then it will work offline.</p></main></body></html>",
      {
        status: 503,
        statusText: "Offline",
        headers: { "Content-Type": "text/html; charset=utf-8" },
      },
    );
  }
}

async function staleWhileRevalidate(request) {
  const cache = await caches.open(CACHE_NAME);
  const cached = await cache.match(request);

  const revalidate = fetch(request)
    .then((response) => {
      if (response && response.ok && response.type === "basic") {
        cache.put(request, response.clone());
      }
      return response;
    })
    .catch(() => null);

  if (cached) {
    revalidate.catch(() => {});
    return cached;
  }

  const fresh = await revalidate;
  if (fresh) return fresh;

  return new Response("", {
    status: 504,
    statusText: "Offline",
  });
}
