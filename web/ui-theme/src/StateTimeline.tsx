import { For } from "solid-js";
import type { MixerState } from "@spidercam/protocol";

const TIMELINE_COLORS: Record<MixerState, string> = {
  SILENCE: "var(--color-spider-meter-track)",
  LOCKED: "var(--color-spider-accent)",
  HOLD: "var(--color-spider-hold)",
  SWITCH: "var(--color-spider-warn)",
};

export function StateTimeline(props: { samples: MixerState[] }) {
  return (
    <div
      class="flex h-2 w-full gap-px rounded-(--radius-spider-sm) overflow-hidden"
      role="img"
    >
      <For each={props.samples}>
        {(s) => (
          <div
            class="flex-1 min-w-0"
            style={{ "background-color": TIMELINE_COLORS[s] }}
          />
        )}
      </For>
    </div>
  );
}
