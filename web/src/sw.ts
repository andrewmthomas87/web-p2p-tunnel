export type RequestData = {
  id: number;
  method: string;
  url: string;
  headersList: [string, any][];
  hasBody: boolean;
  serialized: ArrayBuffer;
};

export async function setupSW(
  tunnel: (serialized: ArrayBuffer) => Promise<ArrayBuffer>,
  statusEl: HTMLElement,
  requestsEl: HTMLElement,
) {
  if (!('serviceWorker' in navigator)) {
    statusEl.innerText = 'error: not supported';
    alert('Error: service workers are not supported');
    throw new Error('service workers are not supported');
  }

  let registration: ServiceWorkerRegistration;
  try {
    registration = await navigator.serviceWorker.register('/sw.js', { type: 'module' });
  } catch (error) {
    alert('Error: failed to register service worker');
    throw error;
  }

  const sw = registration.installing || registration.waiting || registration.active;
  if (sw) {
    statusEl.innerText = sw.state;

    sw.addEventListener('statechange', (ev) => {
      statusEl.innerText = (ev.target as ServiceWorker).state;
    });
  }

  navigator.serviceWorker.addEventListener('message', async (ev) => {
    if (ev.data.type === 'request') {
      const data = ev.data as RequestData;
      addToTable(data, requestsEl);

      let resp;
      try {
        resp = await tunnel(data.serialized);
      } catch (ex) {
        if (ex instanceof ArrayBuffer) {
          resp = ex;
        } else {
          throw ex;
        }
      }

      ev.source?.postMessage(
        {
          type: 'response',
          id: data.id,
          serialized: resp,
        },
        { transfer: [resp] },
      );
    }
  });
}

function addToTable(
  { id, method, url, headersList, hasBody }: RequestData,
  requestsEl: HTMLElement,
) {
  const headersStr = headersList.map(([key, value]) => `${key}: ${value}`).join('\n');

  const tr = document.createElement('tr');
  tr.innerHTML = `
  <td><pre>${id}</pre></td>
  <td><pre>${method}</pre></td>
  <td><pre>${url}</pre></td>
  <td>
    <details>
      <summary>${headersList.length} header${headersList.length !== 1 ? 's' : ''}</summary>
      <pre>${headersStr}</pre>
    </details>
  </td>
  <td><pre>${hasBody ? 'true' : 'false'}</pre></td>`;
  requestsEl.querySelector('tbody')?.appendChild(tr);
}
