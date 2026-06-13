import type {
  ParticipantMessage,
  ParticipantRoomView,
} from "@spidercam/protocol";
import type {
  ParticipantSignaling,
  SdpRelayMessage,
  WelcomePayload,
} from "./participant-signaling.js";

type WelcomeHandler = (clientId: string, view: ParticipantRoomView) => void;
type ViewHandler = (view: ParticipantRoomView) => void;
type CloseHandler = (wasClean: boolean) => void;

export class LiveParticipantSignaling implements ParticipantSignaling {
  private ws: WebSocket | null = null;
  private welcomeHandler: WelcomeHandler | null = null;
  private viewHandler: ViewHandler | null = null;
  private closeHandler: CloseHandler | null = null;

  connect(): Promise<WelcomePayload> {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return Promise.reject(new Error("already connected"));
    }

    return new Promise((resolve, reject) => {
      const protocol = location.protocol === "https:" ? "wss:" : "ws:";
      const ws = new WebSocket(`${protocol}//${location.host}/api/v1/ws`);
      let settled = false;

      ws.onopen = () => {
        this.ws = ws;
      };

      ws.onerror = () => {
        if (!settled) {
          settled = true;
          reject(new Error("WebSocket connection failed"));
        }
      };

      ws.onclose = (event) => {
        this.ws = null;
        if (!settled) {
          settled = true;
          reject(new Error("WebSocket closed before welcome"));
        }
        this.closeHandler?.(event.wasClean);
      };

      ws.onmessage = (event) => {
        let msg: ParticipantMessage;
        try {
          msg = JSON.parse(String(event.data)) as ParticipantMessage;
        } catch {
          return;
        }

        if (msg.type === "welcome") {
          this.welcomeHandler?.(msg.clientId, msg.view);
          if (!settled) {
            settled = true;
            resolve({ clientId: msg.clientId, view: msg.view });
          }
          return;
        }

        if (msg.type === "participant-view") {
          this.viewHandler?.(msg.view);
        }
      };
    });
  }

  disconnect(): void {
    if (!this.ws) {
      return;
    }
    const ws = this.ws;
    this.ws = null;
    ws.onclose = null;
    ws.close(1000, "client disconnect");
  }

  onWelcome(handler: WelcomeHandler): void {
    this.welcomeHandler = handler;
  }

  onView(handler: ViewHandler): void {
    this.viewHandler = handler;
  }

  onClose(handler: CloseHandler): void {
    this.closeHandler = handler;
  }

  sendJoin(
    name: string,
    hasVideo: boolean,
    hasAudio: boolean,
    clientId?: string,
  ): void {
    this.send({
      type: "join",
      name,
      hasVideo,
      hasAudio,
      ...(clientId ? { clientId } : {}),
    });
  }

  sendLeave(): void {
    this.send({ type: "leave" });
  }

  relaySDP(msg: SdpRelayMessage): void {
    this.send(msg);
  }

  private send(payload: unknown): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(payload));
    }
  }
}
