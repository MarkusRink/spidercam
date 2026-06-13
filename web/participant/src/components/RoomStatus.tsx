import {
  LoopDelayText,
  TransportBlock,
  deriveOnAirLabel,
  transportCells,
} from "@spidercam/ui-theme";
import { Show } from "solid-js";
import { selfTransportMetric } from "../lib/transport.js";
import { useStore } from "../stores/store-context.js";

export function RoomStatus() {
  const store = useStore();

  return (
    <Show
      when={store.state.phase === "connected" ? store.state.view : undefined}
    >
      {(view) => (
        <Show when={store.state.clientId}>
          {(clientId) => {
            const metric = () =>
              selfTransportMetric(
                view().selfMetric,
                clientId(),
                store.state.media.audioEnabled,
                store.state.media.videoEnabled,
              );

            return (
              <section class="space-y-2 border-t border-(--color-spider-border) pt-3 font-mono text-[11px]">
                <div class="flex gap-2">
                  <span class="text-(--color-spider-muted)">On air:</span>
                  <span class="text-(--color-spider-text)">
                    {deriveOnAirLabel(view(), clientId())}
                  </span>
                </div>
                <div class="flex items-center gap-2">
                  <span class="text-(--color-spider-muted)">Loop delay ·</span>
                  <LoopDelayText estimate={view().selfMetric.loopDelay} />
                </div>
                <p class="text-[9px] text-(--color-spider-muted)">
                  Updates when remote speaks in Teams
                </p>
                <div class="flex gap-2">
                  <span class="text-(--color-spider-muted)">SNR ·</span>
                  <span>{view().selfMetric.snrDb.toFixed(0)} dB</span>
                  <span class="text-(--color-spider-muted)">· room ·</span>
                  <span>{view().participants.length}</span>
                </div>
                <TransportBlock cells={transportCells(metric(), false)} />
              </section>
            );
          }}
        </Show>
      )}
    </Show>
  );
}
