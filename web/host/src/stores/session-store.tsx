import {
  DefaultHostConfig,
  HostStreamID,
  PlaybackRefStreamID,
  type HostConfig,
  type MixerState,
  type RoomState,
  type StreamMetrics,
} from "@spidercam/protocol";
import {
  createContext,
  createSignal,
  onCleanup,
  onMount,
  useContext,
  type Accessor,
  type ParentProps,
} from "solid-js";
import { createStore } from "solid-js/store";
import type { HostSignaling } from "../adapters/host-signaling.js";
import { LiveHostSignaling } from "../adapters/live-host-signaling.js";

const TIMELINE_SECONDS = 45;
const HOST_STATE_HZ = 50;
const TIMELINE_MAX = TIMELINE_SECONDS * HOST_STATE_HZ;

export interface InterpolatedMeters {
  outLevelDbfs: number;
  outPeakDbfs: number;
  refRmsDbfs: number;
  refPeakDbfs: number;
  metrics: StreamMetrics[];
}

export interface SessionStore {
  state: Accessor<RoomState | null>;
  config: HostConfig;
  updateConfig: (partial: Partial<HostConfig>) => void;
  sendConfig: (partial: Partial<HostConfig>) => void;
  captureDevices: Accessor<import("@spidercam/protocol").CaptureDevices | null>;
  timeline: Accessor<MixerState[]>;
  meters: Accessor<InterpolatedMeters | null>;
  signaling: HostSignaling;
}

const SessionStoreContext = createContext<SessionStore>();

function lerp(a: number, b: number, t: number): number {
  return a + (b - a) * t;
}

function interpolateMetrics(
  from: StreamMetrics[],
  to: StreamMetrics[],
  t: number,
): StreamMetrics[] {
  const toById = new Map(to.map((m) => [m.participantId, m]));
  return from.map((prev) => {
    const next = toById.get(prev.participantId);
    if (!next) {
      return prev;
    }
    return {
      ...next,
      rmsDbfs: lerp(prev.rmsDbfs, next.rmsDbfs, t),
      peakDbfs: lerp(prev.peakDbfs, next.peakDbfs, t),
    };
  });
}

function snapshotMeters(state: RoomState): InterpolatedMeters {
  return {
    outLevelDbfs: state.outLevelDbfs,
    outPeakDbfs: state.outPeakDbfs,
    refRmsDbfs: state.reference.rmsDbfs,
    refPeakDbfs: state.reference.peakDbfs,
    metrics: state.metrics,
  };
}

export function SessionStoreProvider(props: ParentProps) {
  const signaling = new LiveHostSignaling();
  const [state, setState] = createSignal<RoomState | null>(null);
  const [captureDevices, setCaptureDevices] = createSignal<
    import("@spidercam/protocol").CaptureDevices | null
  >(null);
  const [timeline, setTimeline] = createSignal<MixerState[]>([]);
  const [meters, setMeters] = createSignal<InterpolatedMeters | null>(null);

  const [config, setConfigStore] = createStore<HostConfig>(
    structuredClone(DefaultHostConfig),
  );

  let prevSnapshot: InterpolatedMeters | null = null;
  let currSnapshot: InterpolatedMeters | null = null;
  let snapshotAt = 0;
  let rafId = 0;

  function tickInterpolation(): void {
    if (prevSnapshot && currSnapshot && snapshotAt > 0) {
      const elapsed = performance.now() - snapshotAt;
      const t = Math.min(elapsed / (1000 / HOST_STATE_HZ), 1);
      setMeters({
        outLevelDbfs: lerp(
          prevSnapshot.outLevelDbfs,
          currSnapshot.outLevelDbfs,
          t,
        ),
        outPeakDbfs: lerp(
          prevSnapshot.outPeakDbfs,
          currSnapshot.outPeakDbfs,
          t,
        ),
        refRmsDbfs: lerp(prevSnapshot.refRmsDbfs, currSnapshot.refRmsDbfs, t),
        refPeakDbfs: lerp(
          prevSnapshot.refPeakDbfs,
          currSnapshot.refPeakDbfs,
          t,
        ),
        metrics: interpolateMetrics(
          prevSnapshot.metrics,
          currSnapshot.metrics,
          t,
        ),
      });
    }
    rafId = requestAnimationFrame(tickInterpolation);
  }

  onMount(() => {
    rafId = requestAnimationFrame(tickInterpolation);

    const unsubState = signaling.onState((next) => {
      setState(next);
      const mixer = next.selection?.mixerState;
      if (mixer) {
        setTimeline((prev) => {
          const updated = [...prev, mixer];
          return updated.length > TIMELINE_MAX
            ? updated.slice(updated.length - TIMELINE_MAX)
            : updated;
        });
      }

      const snap = snapshotMeters(next);
      prevSnapshot = currSnapshot;
      currSnapshot = snap;
      snapshotAt = performance.now();
      if (!prevSnapshot) {
        setMeters(snap);
      }
    });

    const unsubDevices = signaling.onCaptureDevices(setCaptureDevices);

    void signaling.connect().then(() => {
      signaling.listCaptureDevices();
    });

    onCleanup(() => {
      cancelAnimationFrame(rafId);
      unsubState();
      unsubDevices();
      signaling.disconnect();
    });
  });

  const updateConfig = (partial: Partial<HostConfig>) => {
    setConfigStore(partial);
    if (partial.scoreWeights) {
      setConfigStore("scoreWeights", (w) => ({
        ...w,
        ...partial.scoreWeights,
      }));
    }
  };

  const sendConfig = (partial: Partial<HostConfig>) => {
    signaling.sendConfig(partial);
  };

  const store: SessionStore = {
    state,
    config,
    updateConfig,
    sendConfig,
    captureDevices,
    timeline,
    meters,
    signaling,
  };

  return (
    <SessionStoreContext.Provider value={store}>
      {props.children}
    </SessionStoreContext.Provider>
  );
}

export function useSessionStore(): SessionStore {
  const ctx = useContext(SessionStoreContext);
  if (!ctx) {
    throw new Error("useSessionStore requires SessionStoreProvider");
  }
  return ctx;
}

export function orderedStreamMetrics(
  state: RoomState | null | undefined,
  meters: InterpolatedMeters | null | undefined,
): StreamMetrics[] {
  if (!state) {
    return [];
  }
  const levelById = new Map(
    meters?.metrics.map((m) => [m.participantId, m]) ?? [],
  );
  const filtered = state.metrics.filter(
    (m) => m.role !== "reference" && m.participantId !== PlaybackRefStreamID,
  );
  const host = filtered.find((m) => m.participantId === HostStreamID);
  const participants = filtered
    .filter((m) => m.participantId !== HostStreamID)
    .sort((a, b) => {
      const aJoin =
        state.participants.find((p) => p.id === a.participantId)?.joinedAt ?? 0;
      const bJoin =
        state.participants.find((p) => p.id === b.participantId)?.joinedAt ?? 0;
      return aJoin - bJoin;
    });
  const ordered = host ? [host, ...participants] : participants;
  return ordered.map((m) => levelById.get(m.participantId) ?? m);
}

export function anyProcessingOn(state: RoomState | null | undefined): boolean {
  if (!state) {
    return false;
  }
  return state.metrics.some((m) => m.aecEnabled || m.denoiseEnabled);
}
