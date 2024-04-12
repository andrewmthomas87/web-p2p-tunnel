export function connectSignalingClient(roomID: string, serverURL: string, statusEl: HTMLElement) {
  const wsURL = new URL('ws', serverURL);
  wsURL.search = new URLSearchParams({ role: 'client', 'room-id': roomID }).toString();
  if (wsURL.protocol === 'https:') {
    wsURL.protocol = 'wss:';
  } else {
    wsURL.protocol = 'ws:';
  }

  const ws = new WebSocket(wsURL);

  ws.addEventListener('open', () => {
    statusEl.innerText = readyStateToString(ws.readyState);
  });
  ws.addEventListener('close', () => {
    statusEl.innerText = readyStateToString(ws.readyState);
  });
  ws.addEventListener('error', () => {
    alert('Signaling error');
  });

  return ws;
}

function readyStateToString(readyState: number): string {
  switch (readyState) {
    case WebSocket.CONNECTING:
      return 'connecting';
    case WebSocket.OPEN:
      return 'open';
    case WebSocket.CLOSING:
      return 'closing';
    case WebSocket.CLOSED:
      return 'closed';
    default:
      return 'unknown';
  }
}
