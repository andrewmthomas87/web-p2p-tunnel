const tunnelConnectFormEl = document.getElementById('tunnel-connect') as HTMLFormElement;
const swStatusEl = document.getElementById('sw-status')!;
const signalingStatusEl = document.getElementById('signaling-status')!;
const webRTCStatusEl = document.getElementById('webrtc-status')!;
const requestsEl = document.getElementById('requests')!;

import { connectSignalingClient } from './signalingClient';
import { setupSW } from './sw';

await setupSW(swStatusEl, requestsEl);

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
});
