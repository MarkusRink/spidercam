import type { RoomState } from "@spidercam/shared";
import { SignalingClient } from "./webrtc/signaling.js";
import { PeerManager } from "./webrtc/peer.js";
import { collectStats, createAudioLevelMonitor } from "./webrtc/stats.js";

const app = document.getElementById("app")!;

let signaling: SignalingClient | null = null;
let peers: PeerManager | null = null;
let localStream: MediaStream | null = null;
let audioLevel = 0;
let stopMonitor: (() => void) | null = null;
let statsInterval: ReturnType<typeof setInterval> | null = null;
let latencyMs = -1;

renderConnect();

function renderConnect(): void {
  app.className = "grid-participant";
  app.innerHTML = `
    <div style="max-width:400px;width:100%">
      <div class="mono" style="font-size:18px;margin-bottom:4px">spidercam</div>
      <div class="muted mono" style="margin-bottom:24px">participant / connect</div>

      <div class="panel" style="padding:16px">
        <div class="field">
          <label>display name</label>
          <input id="name" type="text" placeholder="your name" value="" />
        </div>
        <div class="field">
          <label>seat</label>
          <select id="seat">
            ${Array.from({ length: 8 }, (_, i) => `<option value="${i}">seat ${i}</option>`).join("")}
          </select>
        </div>
        <div class="field">
          <label>server</label>
          <input id="server" type="text" value="${location.host}" />
        </div>
        <div class="field" style="display:flex;gap:16px">
          <label style="display:flex;align-items:center;gap:6px;text-transform:none">
            <input id="useVideo" type="checkbox" checked /> webcam
          </label>
          <label style="display:flex;align-items:center;gap:6px;text-transform:none">
            <input id="useAudio" type="checkbox" checked /> microphone
          </label>
        </div>
        <button class="btn btn-primary" id="connectBtn" style="width:100%">connect</button>
        <div id="error" class="error mono" style="margin-top:12px;font-size:11px"></div>
      </div>
    </div>
  `;

  document.getElementById("connectBtn")!.onclick = () => void doConnect();
}

async function doConnect(): Promise<void> {
  const name = (document.getElementById("name") as HTMLInputElement).value.trim() || "anonymous";
  const seat = Number((document.getElementById("seat") as HTMLSelectElement).value);
  const server = (document.getElementById("server") as HTMLInputElement).value.trim();
  const useVideo = (document.getElementById("useVideo") as HTMLInputElement).checked;
  const useAudio = (document.getElementById("useAudio") as HTMLInputElement).checked;
  const errorEl = document.getElementById("error")!;

  try {
    localStream = await navigator.mediaDevices.getUserMedia({
      video: useVideo,
      audio: useAudio,
    });

    if (useAudio) {
      stopMonitor = createAudioLevelMonitor(localStream, (l) => {
        audioLevel = l;
      });
    }

    const proto = location.protocol === "https:" ? "wss" : "ws";
    signaling = new SignalingClient();
    await signaling.connect(`${proto}://${server}/ws`);

    signaling.send({
      type: "join",
      name,
      seat,
      role: "participant",
      hasVideo: useVideo,
      hasAudio: useAudio,
    });

    peers = new PeerManager(signaling, signaling.clientId!);
    peers.setLocalStream(localStream);

    signaling.onMessage((msg) => {
      if (msg.type === "room-update") renderSession(msg.room);
    });

    latencyMs = await signaling.measureLatency();
    renderSession(signaling.room!);

    statsInterval = setInterval(() => void reportStats(seat), 1000);
  } catch (err) {
    errorEl.textContent = err instanceof Error ? err.message : "connection failed";
    cleanup();
  }
}

async function reportStats(seat: number): Promise<void> {
  if (!signaling?.clientId || !peers) return;
  const peer = peers.getAllPeers()[0];
  const metrics = peer
    ? await collectStats(peer.pc, signaling.clientId, seat, audioLevel)
    : { participantId: signaling.clientId, seat, audioLevel, audioActive: audioLevel > 0.02 };

  signaling.send({
    type: "metrics",
    from: signaling.clientId,
    metrics,
  });
}

function renderSession(room: RoomState): void {
  app.className = "grid-participant";
  const me = room.participants.find((p) => p.id === signaling?.clientId);

  app.innerHTML = `
    <div style="max-width:520px;width:100%">
      <div class="mono" style="font-size:18px;margin-bottom:4px">spidercam</div>
      <div class="muted mono" style="margin-bottom:16px">connected · seat ${me?.seat ?? "?"} · ${me?.name ?? ""}</div>

      <div class="preview-box">
        <video id="preview" autoplay muted playsinline></video>
      </div>

      <div class="panel" style="padding:12px;margin-top:12px">
        <table>
          <tr><td class="muted">signal latency</td><td class="mono">${latencyMs >= 0 ? `${latencyMs} ms` : "—"}</td></tr>
          <tr><td class="muted">audio level</td><td>
            <div class="meter"><div class="meter-fill" id="levelBar" style="width:${Math.round(audioLevel * 100)}%"></div></div>
          </td></tr>
          <tr><td class="muted">participants</td><td class="mono">${room.participants.length}</td></tr>
          <tr><td class="muted">active video</td><td class="mono accent">${room.selection?.activeVideoId ?? "—"}</td></tr>
          <tr><td class="muted">active audio</td><td class="mono accent">${room.selection?.activeAudioId ?? "—"}</td></tr>
        </table>
      </div>

      <button class="btn btn-danger" id="disconnectBtn" style="margin-top:12px;width:100%">disconnect</button>
    </div>
  `;

  const preview = document.getElementById("preview") as HTMLVideoElement;
  if (localStream) preview.srcObject = localStream;

  const levelBar = document.getElementById("levelBar");
  const levelLoop = () => {
    if (levelBar) levelBar.style.width = `${Math.round(audioLevel * 100)}%`;
    requestAnimationFrame(levelLoop);
  };
  levelLoop();

  document.getElementById("disconnectBtn")!.onclick = () => {
    signaling?.send({ type: "leave" });
    cleanup();
    renderConnect();
  };
}

function cleanup(): void {
  if (statsInterval) clearInterval(statsInterval);
  stopMonitor?.();
  peers?.close();
  signaling?.close();
  localStream?.getTracks().forEach((t) => t.stop());
  localStream = null;
  signaling = null;
  peers = null;
}
