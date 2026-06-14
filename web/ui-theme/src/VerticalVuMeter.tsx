import { formatDbfs, levelPct } from "./derive.js";

export function VerticalVuMeter(props: {
  rmsDbfs: number;
  peakDbfs: number;
  label?: string;
  compact?: boolean;
}) {
  const clipped = props.peakDbfs >= -1;

  return (
    <div class="flex flex-col items-center gap-1">
      <div
        class="relative w-3 rounded-(--radius-spider-sm) bg-(--color-spider-meter-track) overflow-hidden"
        classList={{
          "h-14": props.compact !== false,
          "h-24": props.compact === false,
        }}
      >
        {clipped && (
          <div class="absolute top-0 inset-x-0 h-1 bg-(--color-spider-error)" />
        )}
        <div
          class="absolute bottom-0 inset-x-0 bg-(--color-spider-accent) transition-[height] duration-[16ms] linear"
          style={{ height: `${levelPct(props.rmsDbfs)}%` }}
        />
        <div
          class="absolute inset-x-0 h-px bg-(--color-spider-warn)"
          style={{ bottom: `${levelPct(props.peakDbfs)}%` }}
        />
      </div>
      <span class="inline-block min-w-[5ch] text-center font-mono text-[10px] tabular-nums text-(--color-spider-muted)">
        {props.label ?? formatDbfs(props.rmsDbfs)}
      </span>
    </div>
  );
}
