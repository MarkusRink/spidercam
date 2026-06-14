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
    <div
      class={`flex min-h-0 flex-col gap-2 rounded-(--radius-spider) border border-(--color-spider-border) bg-(--color-spider-surface) p-2 lg:border-0 lg:bg-transparent lg:p-0 ${props.class ?? ""}`}
    >
      <h2 class="text-[11px] text-(--color-spider-muted) lg:hidden">
        Participants
      </h2>
      <div class="min-h-0 overflow-y-auto lg:overflow-visible">
        <div class="grid grid-cols-[repeat(auto-fill,minmax(var(--stream-card-w),1fr))] gap-2 content-start font-mono">
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
    </div>
  );
}
