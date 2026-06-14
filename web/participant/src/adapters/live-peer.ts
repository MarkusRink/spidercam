import type { AnswerMsg, ICECandidateMsg } from "@spidercam/protocol";
import type { ParticipantSignaling } from "./participant-signaling.js";
import type {
  ParticipantPeer,
  PeerConnectionState,
} from "./participant-peer.js";

const ICE_SERVERS: RTCIceServer[] = [{ urls: "stun:stun.l.google.com:19302" }];

const ANSWER_TIMEOUT_MS = 10_000;

export class LiveParticipantPeer implements ParticipantPeer {
  private pc: RTCPeerConnection | null = null;
  private senders = new Map<"audio" | "video", RTCRtpSender>();
  private stateHandler: ((state: PeerConnectionState) => void) | null = null;
  private pendingCandidates: RTCIceCandidateInit[] = [];
  private remoteDescriptionSet = false;
  private answerResolver: ((sdp: string) => void) | null = null;
  private answerRejector: ((err: Error) => void) | null = null;

  constructor(private readonly signaling: ParticipantSignaling) {
    signaling.onInboundSdp((msg) => {
      void this.handleInboundSdp(msg);
    });
  }

  async start(stream: MediaStream): Promise<void> {
    this.closePc();
    this.remoteDescriptionSet = false;
    this.pendingCandidates = [];

    const pc = new RTCPeerConnection({ iceServers: ICE_SERVERS });
    this.pc = pc;

    pc.onconnectionstatechange = () => {
      this.mapConnectionState(pc.connectionState);
    };

    pc.onicecandidate = (event) => {
      if (!event.candidate) {
        return;
      }
      this.signaling.relaySDP({
        type: "ice-candidate",
        from: "",
        to: "",
        candidate: JSON.stringify(event.candidate.toJSON()),
      });
    };

    for (const track of stream.getTracks()) {
      const sender = pc.addTrack(track, stream);
      this.senders.set(track.kind as "audio" | "video", sender);
    }

    this.mapConnectionState("connecting");

    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    if (!offer.sdp) {
      throw new Error("offer missing sdp");
    }

    const answerWait = this.waitForAnswer();
    this.signaling.relaySDP({
      type: "offer",
      from: "",
      to: "",
      sdp: offer.sdp,
    });

    try {
      const answerSdp = await answerWait;
      await pc.setRemoteDescription({ type: "answer", sdp: answerSdp });
      this.remoteDescriptionSet = true;
      await this.flushPendingCandidates();
    } catch {
      /* mock hub or offline: continue without negotiated WebRTC */
    }
  }

  async replaceTrack(
    kind: "audio" | "video",
    track: MediaStreamTrack | null,
  ): Promise<void> {
    const pc = this.pc;
    if (!pc) {
      return;
    }

    const sender = this.senders.get(kind);
    if (sender) {
      await sender.replaceTrack(track);
      return;
    }

    if (!track) {
      return;
    }

    const stream = new MediaStream([track]);
    const next = pc.addTrack(track, stream);
    this.senders.set(kind, next);
  }

  close(): void {
    this.closePc();
    this.mapConnectionState("closed");
  }

  onConnectionStateChange(handler: (state: PeerConnectionState) => void): void {
    this.stateHandler = handler;
  }

  private waitForAnswer(): Promise<string> {
    return new Promise((resolve, reject) => {
      this.answerResolver = resolve;
      this.answerRejector = reject;
      setTimeout(() => {
        if (!this.answerRejector) {
          return;
        }
        this.answerRejector(new Error("answer timeout"));
        this.answerResolver = null;
        this.answerRejector = null;
      }, ANSWER_TIMEOUT_MS);
    });
  }

  private async handleInboundSdp(
    msg: AnswerMsg | ICECandidateMsg,
  ): Promise<void> {
    if (msg.type === "answer") {
      if (!msg.sdp?.trim()) {
        return;
      }
      if (this.answerResolver) {
        this.answerResolver(msg.sdp);
        this.answerResolver = null;
        this.answerRejector = null;
      }
      return;
    }

    if (!msg.candidate?.trim() || !this.pc) {
      return;
    }

    let init: RTCIceCandidateInit;
    try {
      init = JSON.parse(msg.candidate) as RTCIceCandidateInit;
    } catch {
      return;
    }

    if (!this.remoteDescriptionSet) {
      this.pendingCandidates.push(init);
      return;
    }

    try {
      await this.pc.addIceCandidate(init);
    } catch {
      /* ignore stale candidates */
    }
  }

  private async flushPendingCandidates(): Promise<void> {
    const pc = this.pc;
    if (!pc) {
      return;
    }
    const queued = this.pendingCandidates;
    this.pendingCandidates = [];
    for (const init of queued) {
      try {
        await pc.addIceCandidate(init);
      } catch {
        /* ignore stale candidates */
      }
    }
  }

  private mapConnectionState(
    state: RTCPeerConnectionState | "connecting",
  ): void {
    switch (state) {
      case "new":
        this.stateHandler?.("new");
        break;
      case "connecting":
        this.stateHandler?.("connecting");
        break;
      case "connected":
        this.stateHandler?.("connected");
        break;
      case "disconnected":
        this.stateHandler?.("disconnected");
        break;
      case "failed":
        this.stateHandler?.("failed");
        break;
      case "closed":
        this.stateHandler?.("closed");
        break;
    }
  }

  private closePc(): void {
    if (this.answerRejector) {
      this.answerRejector(new Error("peer closed"));
      this.answerResolver = null;
      this.answerRejector = null;
    }
    if (this.pc) {
      this.pc.onconnectionstatechange = null;
      this.pc.onicecandidate = null;
      this.pc.close();
      this.pc = null;
    }
    this.senders.clear();
    this.pendingCandidates = [];
    this.remoteDescriptionSet = false;
  }
}
