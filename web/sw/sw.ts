import { deserializeResponse, serializeRequest } from './http';

const sw = self as ServiceWorkerGlobalScope & typeof globalThis;

sw.addEventListener('install', () => {
  sw.skipWaiting();
});

sw.addEventListener('activate', (ev) => {
  ev.waitUntil(sw.clients.claim());
});

let id = 1;
const responseResolvers = new Map<number, (value: Response) => void>();

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
    const { id, serialized } = ev.data as {
      id: number;
      serialized: ArrayBuffer;
    };
    const res = deserializeResponse(serialized);

    const resolve = responseResolvers.get(id);
    if (!resolve) {
      console.warn(`Received response with unknown id ${id}`);
      return;
    }

    resolve(res);
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
    return new Response(null, { status: 503 });
  }

  const { method, url, headers } = ev.request;
  const headersList: [string, string][] = [];
  headers.forEach((value, key) => {
    headersList.push([key, value]);
  });
  const hasBody = ev.request.body !== null;
  const serialized = await serializeRequest(ev.request, {
    origin: sw.origin,
    userAgent: sw.navigator.userAgent,
  });

  const resPromise = new Promise<Response>((resolve) => {
    responseResolvers.set(id, resolve);
  });

  tc.postMessage(
    {
      type: 'request',
      id,
      method,
      url,
      headersList,
      hasBody,
      serialized,
    },
    [serialized],
  );
  id++;

  return await resPromise;
}
