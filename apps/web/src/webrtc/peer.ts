import type { SignalingMessage } from "@spidercam/shared";
import type { SignalingClient } from "./signaling.js";

const ICE_SERVERS: RTCIceServer[] = [{ urls: "stun:stun.l.google.com:19302" }];

export interface PeerConnection {
  id: string;
  pc: RTCPeerConnection;
  stream: MediaStream | null;
}

export class PeerManager {
  private peers = new Map<string, PeerConnection>();
  private localStream: MediaStream | null = null;
  private isHost = false;

  constructor(
    private signaling: SignalingClient,
    private selfId: string,
  ) {
    this.signaling.onMessage((msg) => this.handleSignaling(msg));
  }

  setLocalStream(stream: MediaStream | null): void {
    this.localStream = stream;
    for (const peer of this.peers.values()) {
      this.syncTracks(peer);
    }
  }

  setHostMode(host: boolean): void {
    this.isHost = host;
  }

  async connectTo(peerId: string): Promise<PeerConnection> {
    if (this.peers.has(peerId)) return this.peers.get(peerId)!;

    const pc = new RTCPeerConnection({ iceServers: ICE_SERVERS });
    const entry: PeerConnection = { id: peerId, pc, stream: null };
    this.peers.set(peerId, entry);

    pc.ontrack = (ev) => {
      entry.stream = ev.streams[0] ?? new MediaStream([ev.track]);
    };

    pc.onicecandidate = (ev) => {
      this.signaling.send({
        type: "ice-candidate",
        from: this.selfId,
        to: peerId,
        candidate: ev.candidate?.toJSON() ?? null,
      });
    };

    this.syncTracks(entry);

    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    this.signaling.send({
      type: "offer",
      from: this.selfId,
      to: peerId,
      sdp: offer,
    });

    return entry;
  }

  private syncTracks(peer: PeerConnection): void {
    if (!this.localStream) return;
    const senders = peer.pc.getSenders();
    for (const track of this.localStream.getTracks()) {
      const existing = senders.find((s) => s.track?.kind === track.kind);
      if (existing) {
        void existing.replaceTrack(track);
      } else {
        peer.pc.addTrack(track, this.localStream);
      }
    }
  }

  private async handleSignaling(msg: SignalingMessage): Promise<void> {
    if (msg.type === "offer") {
      if (msg.to !== this.selfId) return;

      let entry = this.peers.get(msg.from);
      if (!entry) {
        const pc = new RTCPeerConnection({ iceServers: ICE_SERVERS });
        entry = { id: msg.from, pc, stream: null };
        this.peers.set(msg.from, entry);

        pc.ontrack = (ev) => {
          entry!.stream = ev.streams[0] ?? new MediaStream([ev.track]);
        };

        pc.onicecandidate = (ev) => {
          this.signaling.send({
            type: "ice-candidate",
            from: this.selfId,
            to: msg.from,
            candidate: ev.candidate?.toJSON() ?? null,
          });
        };

        this.syncTracks(entry);
      }

      await entry.pc.setRemoteDescription(msg.sdp);
      const answer = await entry.pc.createAnswer();
      await entry.pc.setLocalDescription(answer);
      this.signaling.send({
        type: "answer",
        from: this.selfId,
        to: msg.from,
        sdp: answer,
      });
    }

    if (msg.type === "answer" && msg.to === this.selfId) {
      const entry = this.peers.get(msg.from);
      if (entry) {
        await entry.pc.setRemoteDescription(msg.sdp);
      }
    }

    if (msg.type === "ice-candidate" && msg.to === this.selfId) {
      const entry = this.peers.get(msg.from);
      if (entry && msg.candidate) {
        await entry.pc.addIceCandidate(msg.candidate);
      }
    }

    if (msg.type === "room-update" && this.isHost) {
      const participants = msg.room.participants.filter(
        (p) => p.id !== this.selfId && p.role === "participant",
      );
      for (const p of participants) {
        if (!this.peers.has(p.id)) {
          void this.connectTo(p.id);
        }
      }
      const activeIds = new Set(participants.map((p) => p.id));
      for (const [id, peer] of this.peers) {
        if (!activeIds.has(id)) {
          peer.pc.close();
          this.peers.delete(id);
        }
      }
    }
  }

  getPeer(id: string): PeerConnection | undefined {
    return this.peers.get(id);
  }

  getAllPeers(): PeerConnection[] {
    return [...this.peers.values()];
  }

  close(): void {
    for (const peer of this.peers.values()) {
      peer.pc.close();
    }
    this.peers.clear();
  }
}
