import { connectSignalingClient } from './signalingClient';
import { setupSW } from './sw';
import { connectWebRTC } from './webrtc';

const tunnelConnectFormEl = document.getElementById('tunnel-connect') as HTMLFormElement;
const swStatusEl = document.getElementById('sw-status')!;
const signalingStatusEl = document.getElementById('signaling-status')!;
const webRTCStatusEl = document.getElementById('webrtc-status')!;
const requestsEl = document.getElementById('requests')!;

let pc: RTCPeerConnection | null = null;

await setupSW(tunnel, swStatusEl, requestsEl);

async function tunnel(serialized: ArrayBuffer): Promise<Response> {
  if (pc === null) {
    return Response.error();
  }

  const dc = pc.createDataChannel('http');
  dc.binaryType = 'arraybuffer';

  dc.addEventListener('open', () => {
    dc.send(serialized);
  });

  dc.addEventListener('message', () => {
    dc.close();
  });

  return Response.error();
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
