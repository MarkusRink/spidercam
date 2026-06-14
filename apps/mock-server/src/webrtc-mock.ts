import type { WebSocket } from "ws";

type SessionDescription = {
  type: "offer" | "answer";
  sdp: string;
};

type IceCandidateInit = {
  candidate?: string;
  sdpMid?: string | null;
  sdpMLineIndex?: number | null;
  usernameFragment?: string | null;
};

type PeerConnection = {
  setRemoteDescription: (desc: SessionDescription) => Promise<void>;
  createAnswer: () => Promise<SessionDescription>;
  setLocalDescription: (desc: SessionDescription) => Promise<void>;
  addIceCandidate: (init: IceCandidateInit) => Promise<void>;
  close: () => void;
  onicecandidate:
    | ((event: {
        candidate: { toJSON: () => IceCandidateInit } | null;
      }) => void)
    | null;
};

type PeerConnectionCtor = new (config: {
  iceServers: { urls: string }[];
}) => PeerConnection;

let peerConnectionCtor: PeerConnectionCtor | null = null;

async function loadPeerConnectionCtor(): Promise<PeerConnectionCtor | null> {
  if (peerConnectionCtor) {
    return peerConnectionCtor;
  }
  try {
    const mod = await import("@roamhq/wrtc");
    peerConnectionCtor = mod.default.RTCPeerConnection as PeerConnectionCtor;
    return peerConnectionCtor;
  } catch {
    return null;
  }
}

export class MockWebRTCPeer {
  private pc: PeerConnection | null = null;

  constructor(
    private readonly clientId: string,
    private readonly send: (ws: WebSocket, message: unknown) => void,
    private readonly ws: WebSocket,
  ) {}

  async handleOffer(sdp: string): Promise<void> {
    const RTCPeerConnection = await loadPeerConnectionCtor();
    if (!RTCPeerConnection) {
      return;
    }

    this.close();
    const pc = new RTCPeerConnection({
      iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
    });
    this.pc = pc;

    pc.onicecandidate = (event) => {
      if (!event.candidate) {
        return;
      }
      this.send(this.ws, {
        type: "ice-candidate",
        from: "",
        to: this.clientId,
        candidate: JSON.stringify(event.candidate.toJSON()),
      });
    };

    await pc.setRemoteDescription({ type: "offer", sdp });
    const answer = await pc.createAnswer();
    await pc.setLocalDescription(answer);
    if (!answer.sdp) {
      return;
    }
    this.send(this.ws, {
      type: "answer",
      from: "",
      to: this.clientId,
      sdp: answer.sdp,
    });
  }

  async handleIce(candidate: string): Promise<void> {
    const pc = this.pc;
    if (!pc) {
      return;
    }
    let init: IceCandidateInit;
    try {
      init = JSON.parse(candidate) as IceCandidateInit;
    } catch {
      return;
    }
    try {
      await pc.addIceCandidate(init);
    } catch {
      /* ignore stale candidates */
    }
  }

  close(): void {
    if (this.pc) {
      this.pc.onicecandidate = null;
      this.pc.close();
      this.pc = null;
    }
  }
}

export class MockWebRTCHub {
  private peers = new Map<string, MockWebRTCPeer>();

  getOrCreate(
    clientId: string,
    send: (ws: WebSocket, message: unknown) => void,
    ws: WebSocket,
  ): MockWebRTCPeer {
    let peer = this.peers.get(clientId);
    if (!peer) {
      peer = new MockWebRTCPeer(clientId, send, ws);
      this.peers.set(clientId, peer);
    }
    return peer;
  }

  remove(clientId: string): void {
    const peer = this.peers.get(clientId);
    if (peer) {
      peer.close();
      this.peers.delete(clientId);
    }
  }
}
