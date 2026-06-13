import type { StreamMetrics, StreamProcessingFlags } from "@spidercam/protocol";
import { scoreBorderOpacity, transportCells } from "./derive.js";
import { LoopDelayText } from "./LoopDelayText.js";
import { OnAirDot } from "./OnAirDot.js";
import { StreamProcessingRow } from "./StreamProcessingRow.js";
import { TransportBlock } from "./TransportBlock.js";
import { VerticalVuMeter } from "./VerticalVuMeter.js";

export function StreamCard(props: {
  metric: StreamMetrics;
  isOnAir: boolean;
  isHost: boolean;
  onProcessingChange?: (id: string, flags: StreamProcessingFlags) => void;
}) {
  const opacity = scoreBorderOpacity(props.metric.scoreSmooth);

  return (
    <div
      class="flex flex-col gap-1 p-2 font-mono shrink-0 grow-0 rounded-(--radius-spider) border bg-(--color-spider-surface)"
      style={{
        width: "var(--stream-card-w)",
        height: "var(--stream-card-h)",
        "border-color": `color-mix(in srgb, var(--color-spider-accent) ${opacity * 100}%, var(--color-spider-border))`,
      }}
    >
      <div class="flex items-center gap-1 min-w-0">
        <OnAirDot onAir={props.isOnAir} />
        <span class="truncate text-[10px]">{props.metric.name}</span>
      </div>
      <VerticalVuMeter
        rmsDbfs={props.metric.rmsDbfs}
        peakDbfs={props.metric.peakDbfs}
        compact
      />
      <LoopDelayText estimate={props.metric.loopDelay} />
      <StreamProcessingRow
        metric={props.metric}
        onChange={(flags) =>
          props.onProcessingChange?.(props.metric.participantId, flags)
        }
      />
      <TransportBlock cells={transportCells(props.metric, props.isHost)} />
    </div>
  );
}
