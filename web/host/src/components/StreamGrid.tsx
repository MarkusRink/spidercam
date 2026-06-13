import { HostStreamID } from "@spidercam/protocol";
import { StreamCard } from "@spidercam/ui-theme";
import { For } from "solid-js";
import {
  orderedStreamMetrics,
  useSessionStore,
} from "../stores/session-store.js";

export function StreamGrid(props: { class?: string }) {
  const store = useSessionStore();

  const streams = () => orderedStreamMetrics(store.state(), store.meters());

  const activeAudioId = () => store.state()?.selection?.activeAudioId ?? "";

  return (
    <div class={props.class}>
      <div class="grid grid-cols-5 gap-2 content-start font-mono">
        <For each={streams()}>
          {(metric) => (
            <StreamCard
              metric={metric}
              isOnAir={activeAudioId() === metric.participantId}
              isHost={metric.participantId === HostStreamID}
              onProcessingChange={(id, flags) =>
                store.signaling.setStreamProcessing(id, flags)
              }
            />
          )}
        </For>
      </div>
    </div>
  );
}
