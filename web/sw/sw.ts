const sw = self as ServiceWorkerGlobalScope & typeof globalThis;

sw.addEventListener('install', () => {
  sw.skipWaiting();
});

sw.addEventListener('activate', (ev) => {
  ev.waitUntil(sw.clients.claim());
});

let id = 1;
const responseResolvers = new Map<number, (value: number) => void>();

const tunnelPattern = /^\/tunnel/;

sw.addEventListener('fetch', (ev) => {
  const url = new URL(ev.request.url);
  if (tunnelPattern.test(url.pathname)) {
    return;
  }

  ev.respondWith(tunnelRequest(ev));
});

sw.addEventListener('message', (ev) => {
  if (ev.data.type === 'response') {
    const { id, n } = ev.data as {
      id: number;
      n: number;
    };

    const resolve = responseResolvers.get(id);
    if (!resolve) {
      console.warn(`Received response with unknown id ${id}`);
      return;
    }

    resolve(n);
    responseResolvers.delete(id);
  }
});

async function getTunnelClient() {
  const clients = await sw.clients.matchAll();
  return clients.find((client) => {
    const url = new URL(client.url);
    return url.pathname === '/tunnel';
  });
}

async function tunnelRequest(ev: FetchEvent): Promise<Response> {
  const tc = await getTunnelClient();
  if (!tc) {
    return Response.error();
  }

  const { method, url, headers, body } = ev.request;
  const headersList: [string, string][] = [];
  headers.forEach((value, key) => {
    headersList.push([key, value]);
  });

  const response = new Promise<number>((resolve) => {
    responseResolvers.set(id, resolve);
  });

  tc.postMessage(
    {
      type: 'request',
      id,
      method,
      url,
      headersList,
      body,
    },
    body ? [body] : [],
  );
  id++;

  const n = await response;

  return Response.json({ n });
}
