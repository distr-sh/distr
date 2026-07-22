// Self-destroying service worker.
// The site previously registered a Workbox service worker (astrojs-service-worker)
// at this path, which aggressively precached all pages and caused visitors to see
// stale content. This replacement unregisters itself and clears all caches for
// returning visitors. It can be removed once it has been deployed for a while.
self.addEventListener('install', () => {
  self.skipWaiting();
});

self.addEventListener('activate', event => {
  event.waitUntil(
    (async () => {
      const cacheKeys = await caches.keys();
      await Promise.all(cacheKeys.map(key => caches.delete(key)));
      await self.registration.unregister();
      const clients = await self.clients.matchAll({type: 'window'});
      clients.forEach(client => client.navigate(client.url));
    })(),
  );
});
