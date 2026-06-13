import type { LoopDelayEstimate } from "@spidercam/protocol";
import { createMemo } from "solid-js";
import {
  formatLoopDelayText,
  loopDelayColor,
  loopDelayKnown,
} from "./derive.js";

const LOOP_DELAY_CSS: Record<ReturnType<typeof loopDelayColor>, string> = {
  muted: "var(--color-spider-muted)",
  accent: "var(--color-spider-accent)",
  warn: "var(--color-spider-warn)",
  error: "var(--color-spider-error)",
};

export function LoopDelayText(props: { estimate: LoopDelayEstimate }) {
  const label = createMemo(() => formatLoopDelayText(props.estimate));
  const color = createMemo(() => {
    const estimate = props.estimate;
    return LOOP_DELAY_CSS[
      loopDelayColor(estimate.ms, loopDelayKnown(estimate))
    ];
  });

  return (
    <span class="font-mono text-[10px] tabular-nums" style={{ color: color() }}>
      {label()}
    </span>
  );
}
