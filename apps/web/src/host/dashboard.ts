import {
  DEFAULT_HOST_CONFIG,
  selectSources,
  type HostConfig,
  type RoomState,
  type StreamMetrics,
} from "@spidercam/shared";
import { SignalingClient } from "../webrtc/signaling.js";
import { PeerManager } from "../webrtc/peer.js";
import { collectStats, createAudioLevelMonitor } from "../webrtc/stats.js";
import { StreamMixer } from "./mixer.js";
import { BridgeClient } from "./bridge-client.js";

const app = document.getElementById("app")!;

let signaling: SignalingClient | null = null;
let peers: PeerManager | null = null;
let mixer: StreamMixer | null = null;
let bridge: BridgeClient | null = null;
let hostStream: MediaStream | null = null;
let hostAudioLevel = 0;
let stopHostMonitor: (() => void) | null = null;
let selectorInterval: ReturnType<typeof setInterval> | null = null;
let config: HostConfig = { ...DEFAULT_HOST_CONFIG };
let bridgeConnected = false;
let stopBridgeVideo: (() => void) | null = null;
let stopBridgeAudio: (() => void) | null = null;

const peerVideos = new Map<string, HTMLVideoElement>();
const peerAudioLevels = new Map<string, number>();

renderStart();

function renderStart(): void {
  app.innerHTML = `
    <div class="grid-participant">
      <div style="max-width:480px;width:100%">
        <div class="mono" style="font-size:18px;margin-bottom:4px">spidercam</div>
        <div class="muted mono" style="margin-bottom:24px">host / mixer</div>
        <div class="panel" style="padding:16px">
          <div class="field">
            <label>seats around table</label>
            <input id="seatCount" type="number" min="2" max="16" value="${config.seatCount}" />
          </div>
          <div class="field">
            <label>your seat</label>
            <input id="hostSeat" type="number" min="0" max="15" value="${config.hostSeat}" />
          </div>
          <button class="btn btn-primary" id="startBtn" style="width:100%">start host</button>
          <div id="error" class="error mono" style="margin-top:12px;font-size:11px"></div>
        </div>
      </div>
    </div>
  `;

  document.getElementById("startBtn")!.onclick = () => void startHost();
}

async function startHost(): Promise<void> {
  const errorEl = document.getElementById("error")!;
  config.seatCount = Number((document.getElementById("seatCount") as HTMLInputElement).value);
  config.hostSeat = Number((document.getElementById("hostSeat") as HTMLInputElement).value);

  try {
    hostStream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
    stopHostMonitor = createAudioLevelMonitor(hostStream, (l) => {
      hostAudioLevel = l;
    });

    signaling = new SignalingClient();
    await signaling.connect();

    signaling.send({
      type: "join",
      name: "host",
      seat: config.hostSeat,
      role: "host-mixer",
      hasVideo: true,
      hasAudio: true,
    });

    signaling.send({ type: "config", config });

    peers = new PeerManager(signaling, signaling.clientId!);
    peers.setHostMode(true);
    peers.setLocalStream(hostStream);

    mixer = new StreamMixer();
    const hostVideo = document.createElement("video");
    hostVideo.srcObject = hostStream;
    hostVideo.muted = true;
    void hostVideo.play();
    mixer.setHostVideo(hostVideo);
    mixer.connectHostAudio(hostStream);

    const output = mixer.start();

    bridge = new BridgeClient();
    bridgeConnected = await bridge.connect();
    if (bridgeConnected) {
      bridge.configure(1280, 720, 48000, 1);
      stopBridgeVideo = bridge.attachVideo(mixer.canvas);
      stopBridgeAudio = bridge.attachAudio(mixer.getAudioContext(), output.audioStream);
    }

    signaling.onMessage((msg) => {
      if (msg.type === "room-update") {
        syncPeers(msg.room);
        renderDashboard(msg.room);
      }
    });

    selectorInterval = setInterval(() => runSelector(), 200);

    renderDashboard(signaling.room!);
  } catch (err) {
    errorEl.textContent = err instanceof Error ? err.message : "failed to start";
  }
}

function syncPeers(room: RoomState): void {
  if (!peers || !mixer) return;

  for (const p of room.participants) {
    if (p.role !== "participant") continue;

    const peer = peers.getPeer(p.id);
    if (peer?.stream) {
      if (!peerVideos.has(p.id)) {
        const video = document.createElement("video");
        video.srcObject = peer.stream;
        video.muted = true;
        void video.play();
        peerVideos.set(p.id, video);
        mixer.setPeerVideo(p.id, video);

        if (peer.stream.getAudioTracks().length > 0) {
          mixer.connectAudio(p.id, peer.stream);
          createAudioLevelMonitor(peer.stream, (l) => peerAudioLevels.set(p.id, l));
        }
      }
    }
  }

  for (const id of peerVideos.keys()) {
    if (!room.participants.find((p) => p.id === id)) {
      peerVideos.delete(id);
      peerAudioLevels.delete(id);
      mixer.removePeer(id);
    }
  }
}

function runSelector(): void {
  if (!signaling?.clientId || !mixer) return;

  const room = signaling.room;
  if (!room) return;

  const metrics: StreamMetrics[] = room.metrics.map((m) => ({
    ...m,
    audioLevel: peerAudioLevels.get(m.participantId) ?? m.audioLevel,
  }));

  const selection = selectSources({
    config,
    metrics,
    hostAudioLevel,
    hostVideoActive: true,
  });

  mixer.setSelection(selection.activeVideoId, selection.activeAudioId);
  signaling.send({ type: "selection", selection });
}

function renderDashboard(room: RoomState): void {
  const selection = room.selection;

  app.className = "";
  app.innerHTML = `
    <div class="grid-host">
      <header style="grid-column:1/-1;display:flex;justify-content:space-between;align-items:center;padding:4px 8px">
        <div>
          <span class="mono" style="font-size:16px">spidercam</span>
          <span class="muted mono"> / host</span>
        </div>
        <div class="mono" style="font-size:11px">
          <span class="status-dot ${bridgeConnected ? "ok" : "err"}"></span>
          bridge ${bridgeConnected ? "connected" : "offline"}
          · ${room.participants.filter((p) => p.role === "participant").length} participants
        </div>
      </header>

      <main class="panel" style="padding:8px;display:flex;flex-direction:column;gap:8px;overflow:hidden">
        <div style="flex:1;display:flex;align-items:center;justify-content:center;background:#000;border-radius:3px;min-height:0">
          <canvas id="outputCanvas" width="1280" height="720" style="max-width:100%;max-height:100%;object-fit:contain"></canvas>
        </div>
        <div class="mono" style="font-size:11px;padding:4px 8px;border-top:1px solid var(--border)">
          <span class="accent">OUT</span>
          video → <span class="accent">${selection?.activeVideoId ?? "—"}</span>
          · audio → <span class="accent">${selection?.activeAudioId ?? "—"}</span>
          · <span class="muted">${selection?.reason ?? ""}</span>
        </div>
      </main>

      <aside style="display:flex;flex-direction:column;gap:8px;overflow-y:auto">
        <div class="panel" style="padding:8px">
          <div class="mono muted" style="font-size:10px;margin-bottom:8px">HOST INPUT</div>
          <video id="hostPreview" style="width:100%;aspect-ratio:16/9" autoplay muted playsinline></video>
          <div class="meter" style="margin-top:6px">
            <div class="meter-fill" id="hostLevel" style="width:0%"></div>
          </div>
        </div>

        <div class="panel" style="padding:8px;flex:1;overflow-y:auto">
          <div class="mono muted" style="font-size:10px;margin-bottom:8px">STREAMS</div>
          <div id="streamList"></div>
        </div>
      </aside>

      <footer style="grid-column:1/-1" class="panel" style="padding:0">
        <table id="metricsTable">
          <thead>
            <tr>
              <th>id</th>
              <th>seat</th>
              <th>level</th>
              <th>rtt</th>
              <th>loss</th>
              <th>jitter</th>
              <th>fps</th>
              <th>bitrate</th>
              <th>role</th>
            </tr>
          </thead>
          <tbody id="metricsBody"></tbody>
        </table>
      </footer>
    </div>
  `;

  const hostPreview = document.getElementById("hostPreview") as HTMLVideoElement;
  if (hostStream) hostPreview.srcObject = hostStream;

  const outputCanvas = document.getElementById("outputCanvas") as HTMLCanvasElement;
  if (mixer) {
    const drawOutput = () => {
      const ctx = outputCanvas.getContext("2d");
      if (ctx) ctx.drawImage(mixer!.canvas, 0, 0, outputCanvas.width, outputCanvas.height);
      requestAnimationFrame(drawOutput);
    };
    drawOutput();
  }

  const hostLevel = document.getElementById("hostLevel");
  const levelLoop = () => {
    if (hostLevel) hostLevel.style.width = `${Math.round(hostAudioLevel * 100)}%`;
    requestAnimationFrame(levelLoop);
  };
  levelLoop();

  const streamList = document.getElementById("streamList")!;
  streamList.innerHTML = room.participants
    .filter((p) => p.role === "participant")
    .map((p) => {
      const m = room.metrics.find((x) => x.participantId === p.id);
      const isVideo = selection?.activeVideoId === p.id;
      const isAudio = selection?.activeAudioId === p.id;
      return `
        <div style="margin-bottom:8px;padding:6px;border:1px solid var(--border);border-radius:3px;${isVideo || isAudio ? "border-color:var(--accent)" : ""}">
          <div style="display:flex;justify-content:space-between;margin-bottom:4px">
            <span class="mono">${p.name}</span>
            <span>
              ${isVideo ? '<span class="badge badge-active">V</span>' : ""}
              ${isAudio ? '<span class="badge badge-active">A</span>' : ""}
            </span>
          </div>
          <video id="vid-${p.id}" style="width:100%;aspect-ratio:16/9" autoplay muted playsinline></video>
          <div class="meter" style="margin-top:4px">
            <div class="meter-fill" style="width:${Math.round((m?.audioLevel ?? 0) * 100)}%"></div>
          </div>
          <div class="mono muted" style="font-size:10px;margin-top:4px">seat ${p.seat} · ${p.hasVideo ? "cam" : "—"} · ${p.hasAudio ? "mic" : "—"}</div>
        </div>
      `;
    })
    .join("") || '<div class="muted mono">no participants</div>';

  for (const p of room.participants) {
    if (p.role !== "participant") continue;
    const el = document.getElementById(`vid-${p.id}`) as HTMLVideoElement | null;
    const video = peerVideos.get(p.id);
    if (el && video) el.srcObject = video.srcObject;
  }

  const metricsBody = document.getElementById("metricsBody")!;
  const hostMetrics: StreamMetrics = {
    participantId: "host",
    seat: config.hostSeat,
    audioLevel: hostAudioLevel,
    videoActive: true,
    audioActive: hostAudioLevel > config.audioThreshold,
    rttMs: 0,
    packetLoss: 0,
    jitterMs: 0,
    bitrateKbps: null,
    framesPerSecond: null,
    lastUpdated: Date.now(),
  };

  const allMetrics = [hostMetrics, ...room.metrics];
  metricsBody.innerHTML = allMetrics
    .map((m) => {
      const p = room.participants.find((x) => x.id === m.participantId);
      const isActive = m.participantId === selection?.activeVideoId || m.participantId === selection?.activeAudioId;
      return `
        <tr style="${isActive ? "color:var(--accent)" : ""}">
          <td>${m.participantId === "host" ? "host" : m.participantId.slice(0, 8)}</td>
          <td>${m.seat}</td>
          <td>${(m.audioLevel * 100).toFixed(1)}%</td>
          <td>${m.rttMs != null ? `${m.rttMs}ms` : "—"}</td>
          <td>${m.packetLoss != null ? `${m.packetLoss}%` : "—"}</td>
          <td>${m.jitterMs != null ? `${m.jitterMs}ms` : "—"}</td>
          <td>${m.framesPerSecond ?? "—"}</td>
          <td>${m.bitrateKbps != null ? `${m.bitrateKbps}` : "—"}</td>
          <td>${m.participantId === "host" ? "host" : p?.role ?? "—"}</td>
        </tr>
      `;
    })
    .join("");
}
