import type {
  HostConfig,
  ParticipantInfo,
  RoomState,
  SelectionState,
  StreamMetrics,
} from "@spidercam/shared";
import { DEFAULT_HOST_CONFIG } from "@spidercam/shared";
import type WebSocket from "ws";

export interface ConnectedClient {
  id: string;
  ws: WebSocket;
  info: ParticipantInfo;
}

export class Room {
  private clients = new Map<string, ConnectedClient>();
  private metrics = new Map<string, StreamMetrics>();
  private selection: SelectionState | null = null;
  private config: HostConfig = { ...DEFAULT_HOST_CONFIG };

  getState(): RoomState {
    return {
      participants: [...this.clients.values()].map((c) => c.info),
      metrics: [...this.metrics.values()],
      selection: this.selection,
      seatCount: this.config.seatCount,
    };
  }

  getConfig(): HostConfig {
    return { ...this.config };
  }

  updateConfig(partial: Partial<HostConfig>): void {
    this.config = { ...this.config, ...partial };
  }

  addClient(id: string, ws: WebSocket, info: ParticipantInfo): void {
    this.clients.set(id, { id, ws, info });
    this.metrics.set(id, {
      participantId: id,
      seat: info.seat,
      audioLevel: 0,
      videoActive: info.hasVideo,
      audioActive: info.hasAudio,
      rttMs: null,
      packetLoss: null,
      jitterMs: null,
      bitrateKbps: null,
      framesPerSecond: null,
      lastUpdated: Date.now(),
    });
  }

  removeClient(id: string): void {
    this.clients.delete(id);
    this.metrics.delete(id);
  }

  getClient(id: string): ConnectedClient | undefined {
    return this.clients.get(id);
  }

  getAllClients(): ConnectedClient[] {
    return [...this.clients.values()];
  }

  getParticipantsExcept(id: string): ParticipantInfo[] {
    return this.getAllClients()
      .filter((c) => c.id !== id)
      .map((c) => c.info);
  }

  updateMetrics(id: string, partial: Partial<StreamMetrics>): void {
    const existing = this.metrics.get(id);
    if (!existing) return;
    this.metrics.set(id, {
      ...existing,
      ...partial,
      participantId: id,
      seat: existing.seat,
      lastUpdated: Date.now(),
    });
  }

  setSelection(selection: SelectionState): void {
    this.selection = selection;
  }

  getMetrics(): StreamMetrics[] {
    return [...this.metrics.values()];
  }
}
