import { createStore } from "solid-js/store";
import type { ParticipantRoomView } from "@spidercam/protocol";
import type {
  ParticipantPeer,
  PeerConnectionState,
} from "../adapters/participant-peer.js";
import type { ParticipantSignaling } from "../adapters/participant-signaling.js";

const DISPLAY_NAME_KEY = "spidercam.displayName";
const CLIENT_ID_KEY = "spidercam.clientId";

export type ConnectionPhase =
  | "idle"
  | "connecting"
  | "connected"
  | "reconnecting"
  | "lost";

export interface MediaDeviceState {
  micId: string;
  cameraId: string;
  audioEnabled: boolean;
  videoEnabled: boolean;
}

export interface ParticipantStoreState {
  phase: ConnectionPhase;
  clientId: string | null;
  displayName: string;
  localStream: MediaStream | null;
  view: ParticipantRoomView | null;
  reconnectAttempt: number;
  reconnectDelayMs: number;
  reconnectCountdownMs: number;
  media: MediaDeviceState;
  devices: {
    mics: MediaDeviceInfo[];
    cameras: MediaDeviceInfo[];
  };
  peerConnected: boolean;
}

function defaultDisplayName(): string {
  const digits = String(Math.floor(1000 + Math.random() * 900000));
  return `client-${digits}`;
}

function loadDisplayName(): string {
  const stored = sessionStorage.getItem(DISPLAY_NAME_KEY);
  if (stored?.trim()) {
    return stored.trim();
  }
  const name = defaultDisplayName();
  sessionStorage.setItem(DISPLAY_NAME_KEY, name);
  return name;
}

function loadClientId(): string | null {
  return sessionStorage.getItem(CLIENT_ID_KEY);
}

function persistClientId(clientId: string | null): void {
  if (clientId) {
    sessionStorage.setItem(CLIENT_ID_KEY, clientId);
  } else {
    sessionStorage.removeItem(CLIENT_ID_KEY);
  }
}

function reconnectBackoffMs(attempt: number): number {
  const base = 1000 * 2 ** attempt;
  return Math.min(base, 30000);
}

export function createParticipantStore(deps: {
  signaling: ParticipantSignaling;
  peer: ParticipantPeer;
}) {
  const { signaling, peer } = deps;

  let userDisconnect = false;
  let wasConnectedThisSession = false;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let countdownTimer: ReturnType<typeof setInterval> | null = null;
  let lastJoinHint: string | undefined;

  const [state, setState] = createStore<ParticipantStoreState>({
    phase: "idle",
    clientId: loadClientId(),
    displayName: loadDisplayName(),
    localStream: null,
    view: null,
    reconnectAttempt: 0,
    reconnectDelayMs: 0,
    reconnectCountdownMs: 0,
    media: {
      micId: "",
      cameraId: "",
      audioEnabled: true,
      videoEnabled: true,
    },
    devices: { mics: [], cameras: [] },
    peerConnected: false,
  });

  function clearReconnectTimers(): void {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    if (countdownTimer) {
      clearInterval(countdownTimer);
      countdownTimer = null;
    }
  }

  async function refreshDevices(): Promise<void> {
    const devices = await navigator.mediaDevices.enumerateDevices();
    setState("devices", {
      mics: devices.filter((d) => d.kind === "audioinput"),
      cameras: devices.filter((d) => d.kind === "videoinput"),
    });
  }

  async function ensureLocalStream(): Promise<MediaStream> {
    if (state.localStream) {
      return state.localStream;
    }

    const constraints: MediaStreamConstraints = {
      audio: state.media.audioEnabled
        ? state.media.micId
          ? { deviceId: { exact: state.media.micId } }
          : true
        : false,
      video: state.media.videoEnabled
        ? state.media.cameraId
          ? { deviceId: { exact: state.media.cameraId } }
          : true
        : false,
    };

    const stream = await navigator.mediaDevices.getUserMedia(constraints);
    const audioTrack = stream.getAudioTracks()[0];
    const videoTrack = stream.getVideoTracks()[0];
    setState({
      localStream: stream,
      media: {
        ...state.media,
        micId: audioTrack?.getSettings().deviceId ?? state.media.micId,
        cameraId: videoTrack?.getSettings().deviceId ?? state.media.cameraId,
      },
    });
    await refreshDevices();
    return stream;
  }

  async function applyTrackToggles(): Promise<void> {
    const stream = state.localStream;
    if (!stream) {
      return;
    }

    for (const track of stream.getAudioTracks()) {
      track.enabled = state.media.audioEnabled;
    }
    for (const track of stream.getVideoTracks()) {
      track.enabled = state.media.videoEnabled;
    }

    if (state.phase === "connected") {
      await peer.replaceTrack(
        "audio",
        state.media.audioEnabled ? (stream.getAudioTracks()[0] ?? null) : null,
      );
      await peer.replaceTrack(
        "video",
        state.media.videoEnabled ? (stream.getVideoTracks()[0] ?? null) : null,
      );
    }
  }

  async function swapDevice(
    kind: "audio" | "video",
    deviceId: string,
  ): Promise<void> {
    setState("media", kind === "audio" ? "micId" : "cameraId", deviceId);

    if (!state.localStream) {
      await ensureLocalStream();
      return;
    }

    const constraints: MediaStreamConstraints =
      kind === "audio"
        ? { audio: { deviceId: { exact: deviceId } }, video: false }
        : { audio: false, video: { deviceId: { exact: deviceId } } };

    const fresh = await navigator.mediaDevices.getUserMedia(constraints);
    const newTrack =
      kind === "audio" ? fresh.getAudioTracks()[0] : fresh.getVideoTracks()[0];
    if (!newTrack) {
      fresh.getTracks().forEach((t) => t.stop());
      return;
    }

    const oldTracks =
      kind === "audio"
        ? state.localStream.getAudioTracks()
        : state.localStream.getVideoTracks();
    for (const track of oldTracks) {
      state.localStream.removeTrack(track);
      track.stop();
    }
    state.localStream.addTrack(newTrack);
    newTrack.enabled =
      kind === "audio" ? state.media.audioEnabled : state.media.videoEnabled;

    if (state.phase === "connected") {
      await peer.replaceTrack(kind, newTrack);
    }
    setState("localStream", state.localStream);
  }

  async function completeJoin(stream: MediaStream): Promise<void> {
    signaling.sendJoin(
      state.displayName,
      state.media.videoEnabled && stream.getVideoTracks().length > 0,
      state.media.audioEnabled && stream.getAudioTracks().length > 0,
      lastJoinHint,
    );
    await peer.start(stream);
    wasConnectedThisSession = true;
    setState({
      phase: "connected",
      reconnectAttempt: 0,
      reconnectDelayMs: 0,
      reconnectCountdownMs: 0,
      peerConnected: true,
    });
  }

  async function connectSession(auto = false): Promise<void> {
    if (
      state.phase === "connecting" ||
      state.phase === "connected" ||
      (state.phase === "reconnecting" && !auto)
    ) {
      return;
    }

    userDisconnect = false;
    clearReconnectTimers();
    setState("phase", "connecting");

    try {
      const welcome = await signaling.connect();
      persistClientId(welcome.clientId);
      lastJoinHint = welcome.clientId;
      setState({ clientId: welcome.clientId, view: welcome.view });
      const stream = await ensureLocalStream();
      await completeJoin(stream);
    } catch {
      if (!userDisconnect && wasConnectedThisSession) {
        beginReconnect();
      } else {
        setState("phase", "idle");
      }
    }
  }

  function beginReconnect(): void {
    if (userDisconnect || !wasConnectedThisSession) {
      setState("phase", "idle");
      return;
    }

    clearReconnectTimers();
    peer.close();
    signaling.disconnect();

    const attempt = state.reconnectAttempt;
    const delayMs = reconnectBackoffMs(attempt);
    setState({
      phase: "reconnecting",
      reconnectAttempt: attempt + 1,
      reconnectDelayMs: delayMs,
      reconnectCountdownMs: delayMs,
      view: null,
      peerConnected: false,
    });

    countdownTimer = setInterval(() => {
      setState(
        "reconnectCountdownMs",
        Math.max(0, state.reconnectCountdownMs - 100),
      );
    }, 100);

    reconnectTimer = setTimeout(() => {
      clearReconnectTimers();
      void connectSession(true);
    }, delayMs);
  }

  function handleSignalingClose(wasClean: boolean): void {
    setState("peerConnected", false);
    if (userDisconnect) {
      setState("phase", "idle");
      return;
    }
    if (wasConnectedThisSession && !wasClean) {
      beginReconnect();
      return;
    }
    if (state.phase === "connected" || state.phase === "connecting") {
      beginReconnect();
    }
  }

  function handlePeerState(peerState: PeerConnectionState): void {
    if (peerState === "connected") {
      setState("peerConnected", true);
      return;
    }
    if (
      (peerState === "failed" || peerState === "disconnected") &&
      !userDisconnect &&
      wasConnectedThisSession
    ) {
      beginReconnect();
    }
  }

  signaling.onWelcome((clientId, view) => {
    persistClientId(clientId);
    lastJoinHint = clientId;
    setState({ clientId, view });
  });

  signaling.onView((view) => {
    setState("view", view);
  });

  signaling.onClose(handleSignalingClose);
  peer.onConnectionStateChange(handlePeerState);

  return {
    state,
    async init(): Promise<void> {
      try {
        await refreshDevices();
        await ensureLocalStream();
      } catch {
        await refreshDevices();
      }
    },
    setDisplayName(name: string): void {
      const trimmed = name.trim();
      if (!trimmed) {
        return;
      }
      sessionStorage.setItem(DISPLAY_NAME_KEY, trimmed);
      setState("displayName", trimmed);
    },
    async connect(): Promise<void> {
      await connectSession(false);
    },
    disconnect(): void {
      userDisconnect = true;
      clearReconnectTimers();
      signaling.sendLeave();
      peer.close();
      signaling.disconnect();
      setState({
        phase: "idle",
        view: null,
        peerConnected: false,
        reconnectAttempt: 0,
        reconnectDelayMs: 0,
        reconnectCountdownMs: 0,
      });
    },
    retryNow(): void {
      if (state.phase !== "reconnecting") {
        return;
      }
      userDisconnect = false;
      clearReconnectTimers();
      void connectSession(true);
    },
    async setAudioEnabled(enabled: boolean): Promise<void> {
      setState("media", "audioEnabled", enabled);
      if (!enabled && !state.media.videoEnabled) {
        return;
      }
      if (!state.localStream && enabled) {
        await ensureLocalStream();
      }
      await applyTrackToggles();
    },
    async setVideoEnabled(enabled: boolean): Promise<void> {
      setState("media", "videoEnabled", enabled);
      if (!enabled && !state.media.audioEnabled) {
        return;
      }
      if (!state.localStream && enabled) {
        await ensureLocalStream();
      }
      await applyTrackToggles();
    },
    async selectMic(deviceId: string): Promise<void> {
      await swapDevice("audio", deviceId);
    },
    async selectCamera(deviceId: string): Promise<void> {
      await swapDevice("video", deviceId);
    },
  };
}

export type ParticipantStore = ReturnType<typeof createParticipantStore>;
