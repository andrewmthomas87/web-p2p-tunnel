export async function connectWebRTC(sc: WebSocket, statusEl: HTMLElement) {
  const pc = new RTCPeerConnection({
    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
  });

  statusEl.innerText = pc.connectionState;
  pc.addEventListener('connectionstatechange', () => {
    statusEl.innerText = pc.connectionState;
  });

  sc.addEventListener('message', (ev) => {
    const message = JSON.parse(ev.data);
    switch (message.type) {
      case 'answer':
        pc.setRemoteDescription(message.data);
        break;

      case 'icecandidate':
        pc.addIceCandidate(message.data);
        break;
    }
  });

  pc.addEventListener('icecandidate', (ev) => {
    if (ev.candidate === null) {
      return;
    }

    sc.send(
      JSON.stringify({
        type: 'icecandidate',
        data: ev.candidate,
      }),
    );
  });

  pc.createDataChannel('control');

  const offer = await pc.createOffer();
  sc.send(
    JSON.stringify({
      type: 'offer',
      data: offer,
    }),
  );

  await pc.setLocalDescription(offer);

  return pc;
}
