import { Show } from "solid-js";
import type { StreamMetrics, StreamProcessingFlags } from "@spidercam/protocol";

function formatMs(us: number): string {
  if (us >= 1000) return `${(us / 1000).toFixed(1)}ms`;
  return `${Math.round(us)}µs`;
}

export function StreamProcessingRow(props: {
  metric: StreamMetrics;
  onChange?: (flags: StreamProcessingFlags) => void;
}) {
  return (
    <div class="flex flex-col gap-0.5 text-[9px]">
      <div class="flex gap-2">
        <label class="flex items-center gap-1">
          <input
            type="checkbox"
            checked={props.metric.aecEnabled}
            onChange={(e) =>
              props.onChange?.({
                aecEnabled: e.currentTarget.checked,
                denoiseEnabled: props.metric.denoiseEnabled,
              })
            }
          />
          AEC
        </label>
        <label class="flex items-center gap-1">
          <input
            type="checkbox"
            checked={props.metric.denoiseEnabled}
            onChange={(e) =>
              props.onChange?.({
                aecEnabled: props.metric.aecEnabled,
                denoiseEnabled: e.currentTarget.checked,
              })
            }
          />
          NS
        </label>
      </div>
      <Show when={props.metric.aecEnabled}>
        <span class="text-(--color-spider-muted)">
          AEC · {formatMs(props.metric.aecUs)}
        </span>
      </Show>
      <Show when={props.metric.denoiseEnabled}>
        <span class="text-(--color-spider-muted)">
          NS · {formatMs(props.metric.denoiseUs)}
        </span>
      </Show>
    </div>
  );
}
