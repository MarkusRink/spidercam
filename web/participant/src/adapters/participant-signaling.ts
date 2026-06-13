import type {
  AnswerMsg,
  ICECandidateMsg,
  OfferMsg,
  ParticipantRoomView,
} from "@spidercam/protocol";

export type SdpRelayMessage = OfferMsg | AnswerMsg | ICECandidateMsg;

export interface WelcomePayload {
  clientId: string;
  view: ParticipantRoomView;
}

export interface ParticipantSignaling {
  connect(): Promise<WelcomePayload>;
  disconnect(): void;
  onWelcome(
    handler: (clientId: string, view: ParticipantRoomView) => void,
  ): void;
  onView(handler: (view: ParticipantRoomView) => void): void;
  onClose(handler: (wasClean: boolean) => void): void;
  sendJoin(
    name: string,
    hasVideo: boolean,
    hasAudio: boolean,
    clientId?: string,
  ): void;
  sendLeave(): void;
  relaySDP(msg: SdpRelayMessage): void;
}
