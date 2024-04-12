export async function setupSW(statusEl: HTMLElement, requestsEl: HTMLElement) {
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
      const { id, method, url, headersList, body } = ev.data as {
        id: number;
        method: string;
        url: string;
        headersList: [string, string][];
        body: ReadableStream<Uint8Array> | null;
      };
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
  <td><pre>${body ? 'true' : 'false'}</pre></td>`;
      requestsEl.querySelector('tbody')?.appendChild(tr);

      await new Promise((resolve) => setTimeout(resolve, 2000));

      ev.source?.postMessage({
        type: 'response',
        id,
        n: Math.random(),
      });
    }
  });
}
