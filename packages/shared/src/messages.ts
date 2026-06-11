import type { HostConfig, RoomState, SelectionState, StreamMetrics } from "./types.js";

export interface SessionDescription {
  type: "offer" | "answer" | "pranswer" | "rollback";
  sdp?: string;
}

export interface IceCandidate {
  candidate?: string;
  sdpMid?: string | null;
  sdpMLineIndex?: number | null;
  usernameFragment?: string | null;
}

export type SignalingMessage =
  | { type: "welcome"; clientId: string; room: RoomState }
  | { type: "join"; name: string; seat: number; role: "participant" | "host-mixer"; hasVideo: boolean; hasAudio: boolean }
  | { type: "leave" }
  | { type: "room-update"; room: RoomState }
  | { type: "offer"; from: string; to: string; sdp: SessionDescription }
  | { type: "answer"; from: string; to: string; sdp: SessionDescription }
  | { type: "ice-candidate"; from: string; to: string; candidate: IceCandidate | null }
  | { type: "metrics"; from: string; metrics: Partial<StreamMetrics> }
  | { type: "selection"; selection: SelectionState }
  | { type: "config"; config: Partial<HostConfig> }
  | { type: "error"; message: string }
  | { type: "ping"; ts: number }
  | { type: "pong"; ts: number; serverTs: number };

export function isSignalingMessage(data: unknown): data is SignalingMessage {
  return typeof data === "object" && data !== null && "type" in data;
}
