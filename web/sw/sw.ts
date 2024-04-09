const sw = self as ServiceWorkerGlobalScope & typeof globalThis;

sw.addEventListener('activate', () => {
  console.log('[sw] Hello, world!');
});
