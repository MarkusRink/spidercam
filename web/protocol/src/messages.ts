import type { HostConfig } from "./config.js";
import type {
  CaptureDevices,
  CaptureSelection,
  CaptureState,
  ParticipantRoomView,
  RoomState,
  StreamProcessingFlags,
} from "./types.js";

export interface WelcomeMsg {
  type: "welcome";
  clientId: string;
  view: ParticipantRoomView;
}

export interface ParticipantViewMsg {
  type: "participant-view";
  view: ParticipantRoomView;
}

export interface JoinMsg {
  type: "join";
  name: string;
  hasVideo: boolean;
  hasAudio: boolean;
  clientId?: string;
}

export interface LeaveMsg {
  type: "leave";
}

export interface OfferMsg {
  type: "offer";
  from: string;
  to: string;
  sdp: string;
}

export interface AnswerMsg {
  type: "answer";
  from: string;
  to: string;
  sdp: string;
}

export interface ICECandidateMsg {
  type: "ice-candidate";
  from: string;
  to: string;
  candidate: string;
}

export interface ErrorMsg {
  type: "error";
  message: string;
}

export type ParticipantInboundMessage =
  | JoinMsg
  | LeaveMsg
  | OfferMsg
  | AnswerMsg
  | ICECandidateMsg;

export type ParticipantOutboundMessage =
  | WelcomeMsg
  | ParticipantViewMsg
  | ErrorMsg;

export type ParticipantMessage =
  | ParticipantInboundMessage
  | ParticipantOutboundMessage;

export interface HostStateMsg {
  type: "host-state";
  state: RoomState;
}

export interface ConfigMsg {
  type: "config";
  config: Partial<HostConfig>;
}

export interface ListCaptureDevicesMsg {
  type: "list-capture-devices";
}

export interface CaptureDevicesMsg {
  type: "capture-devices";
  devices: CaptureDevices;
}

export interface SetCaptureDevicesMsg {
  type: "set-capture-devices";
  selection: CaptureSelection;
}

export interface CaptureDevicesUpdatedMsg {
  type: "capture-devices-updated";
  capture: CaptureState;
  error?: string;
}

export interface ParticipantURLMsg {
  type: "participant-url";
  url: string;
}

export interface CopyParticipantURLMsg {
  type: "copy-participant-url";
}

export interface SetStreamProcessingMsg {
  type: "set-stream-processing";
  participantId: string;
  flags: StreamProcessingFlags;
}

export interface PreviewStreamInitMsg {
  type: "preview-stream-init";
  codec: string;
  width: number;
  height: number;
  fps: number;
}

export interface PreviewCutMsg {
  type: "preview-cut";
  activeVideoId: string;
  seq: number;
}

export type HostInboundMessage =
  | ConfigMsg
  | ListCaptureDevicesMsg
  | SetCaptureDevicesMsg
  | CopyParticipantURLMsg
  | SetStreamProcessingMsg;

export type HostOutboundMessage =
  | HostStateMsg
  | CaptureDevicesMsg
  | CaptureDevicesUpdatedMsg
  | ParticipantURLMsg
  | PreviewStreamInitMsg
  | PreviewCutMsg;

export type HostControlMessage = HostInboundMessage | HostOutboundMessage;

export type PreviewMessage = PreviewStreamInitMsg | PreviewCutMsg;
