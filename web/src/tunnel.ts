import { connectSignalingClient } from './signalingClient';
import { setupSW } from './sw';
import { connectWebRTC } from './webrtc';

const MTU = 16 * 1024 - 1;
const TUNNEL_UNAVAILABLE_RESPONSE =
  'HTTP/1.1 503 Service Unavailable\r\n' +
  'Content-Type: text/html\r\n' +
  '\r\n' +
  '<h1>503: Service Unavailable</h1>' +
  '<p>The tunnel is disconnected. <a href="/tunnel">Tunnel page</a>.</p>';
const TUNNEL_ERROR_RESPONSE = 'HTTP/1.1 502 Bad Gateway\r\n\r\n';

const encoder = new TextEncoder();

const tunnelConnectFormEl = document.getElementById('tunnel-connect') as HTMLFormElement;
const swStatusEl = document.getElementById('sw-status')!;
const signalingStatusEl = document.getElementById('signaling-status')!;
const webRTCStatusEl = document.getElementById('webrtc-status')!;
const requestsEl = document.getElementById('requests')!;

let pc: RTCPeerConnection | null = null;

await setupSW(tunnel, swStatusEl, requestsEl);

async function tunnel(serialized: ArrayBuffer): Promise<ArrayBuffer> {
  return new Promise((resolve, reject) => {
    if (pc === null) {
      reject(encoder.encode(TUNNEL_UNAVAILABLE_RESPONSE).buffer);
      return;
    }

    const dc = pc.createDataChannel('http');
    dc.binaryType = 'arraybuffer';

    dc.addEventListener('open', () => {
      const arr = new Uint8Array(serialized);
      const count = Math.ceil(arr.length / MTU);
      for (let i = 0; i < count; i++) {
        const fragment = arr.subarray(i * MTU, Math.min((i + 1) * MTU, arr.length));
        dc.send(fragment);
      }

      dc.send(new ArrayBuffer(0));
    });

    dc.addEventListener('close', () => {
      reject(encoder.encode(TUNNEL_ERROR_RESPONSE).buffer);
    });

    const fragments: ArrayBuffer[] = [];

    const respond = () => {
      const byteLength = fragments.reduce((prev, curr) => prev + curr.byteLength, 0);
      const out = new ArrayBuffer(byteLength);
      const arr = new Uint8Array(out);

      let i = 0;
      for (const fragment of fragments) {
        arr.set(new Uint8Array(fragment), i);
        i += fragment.byteLength;
      }

      resolve(out);
    };

    dc.addEventListener('message', (ev) => {
      if (!(ev.data instanceof ArrayBuffer)) {
        reject(encoder.encode(TUNNEL_ERROR_RESPONSE).buffer);
        return;
      }

      if (ev.data.byteLength === 0) {
        respond();
        dc.close();
        return;
      }

      fragments.push(ev.data);
    });
  });
}

tunnelConnectFormEl.addEventListener('submit', (ev) => {
  ev.preventDefault();

  const data = new FormData(tunnelConnectFormEl);
  const roomID = data.get('room-id');
  if (!(typeof roomID === 'string' && roomID)) {
    alert('Invalid data');
    return;
  }

  const sc = connectSignalingClient(
    roomID,
    import.meta.env.PUBLIC_SIGNALING_SERVER_URL,
    signalingStatusEl,
  );

  sc.addEventListener('open', async () => {
    pc = await connectWebRTC(sc, webRTCStatusEl);
  });
});
