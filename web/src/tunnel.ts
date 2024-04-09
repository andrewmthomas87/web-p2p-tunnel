console.log('[tunnel] Hello, world!');

if ('serviceWorker' in navigator) {
  try {
    await navigator.serviceWorker.register('/sw.js', { type: 'module' });
  } catch (error) {
    console.error('Failed to register service worker', error);
  }
} else {
  console.error('Service workers are not supported');
}

export {};
