import { For } from "solid-js";
import type { TransportCell } from "./derive.js";

function toneClass(tone?: string) {
  if (tone === "error") return "text-(--color-spider-error)";
  if (tone === "warn") return "text-(--color-spider-warn)";
  return "text-(--color-spider-muted)";
}

export function TransportBlock(props: { cells: TransportCell[] }) {
  return (
    <div class="grid grid-cols-3 gap-x-1 gap-y-0.5 font-mono text-[9px] tabular-nums">
      <For each={props.cells}>
        {(c) => <span class={toneClass(c.tone)}>{c.text}</span>}
      </For>
    </div>
  );
}
