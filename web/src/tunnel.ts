let temp = document.createElement('template');
temp.innerHTML = `
<table>
  <thead>
    <tr>
      <th>ID</th>
      <th>Method</th>
      <th>URL</th>
      <th>Headers</th>
      <th>Body?</th>
    </tr>
  </thead>
  <tbody>
  </tbody>
</table>`;
const messagesTable = temp.content.children[0];
const messagesTbody = messagesTable.querySelector('tbody');
document.body.appendChild(temp.content);

if (!('serviceWorker' in navigator)) {
  throw new Error('service workers are not supported');
}

try {
  await navigator.serviceWorker.register('/sw.js', { type: 'module' });
} catch (error) {
  console.error('Failed to register service worker', error);
  throw error;
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

    temp.innerHTML = `
<tr>
  <td><pre>${id}</pre></td>
  <td><pre>${method}</pre></td>
  <td><pre>${url}</pre></td>
  <td>
    <details>
      <summary>${headersList.length} header${headersList.length !== 1 ? 's' : ''}</summary>
      <pre>${headersStr}</pre>
    </details>
  </td>
  <td><pre>${body ? 'true' : 'false'}</pre></td>
</tr>`;
    messagesTbody?.appendChild(temp.content);

    await new Promise((resolve) => setTimeout(resolve, 2000));

    ev.source?.postMessage({
      type: 'response',
      id,
      n: Math.random(),
    });
  }
});

temp.innerHTML = `
<form>
  <label>
    Room ID:
    <input type="text" name="room-id" />
  </label>
  <button type="submit">Connect</button>
</form>`;
const connectForm = temp.content.children[0] as HTMLFormElement;
document.body.appendChild(temp.content);

connectForm.addEventListener('submit', (ev) => {
  ev.preventDefault();

  const data = new FormData(connectForm);
  const roomID = data.get('room-id');
  if (!(typeof roomID === 'string' && roomID)) {
    throw new Error('connect form: invalid data');
  }

  const wsURL = new URL('ws', import.meta.env.PUBLIC_SIGNALING_SERVER_URL);
  wsURL.search = new URLSearchParams({ role: 'client', 'room-id': roomID }).toString();
  if (wsURL.protocol === 'https:') {
    wsURL.protocol = 'wss:';
  } else {
    wsURL.protocol = 'ws:';
  }

  const ws = new WebSocket(wsURL);
});

export {};
