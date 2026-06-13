import { Show } from "solid-js";
import {
  anyProcessingOn,
  orderedStreamMetrics,
  useSessionStore,
} from "../stores/session-store.js";

function dspTone(pct: number): string {
  if (pct < 5) return "text-(--color-spider-accent)";
  if (pct <= 15) return "text-(--color-spider-warn)";
  return "text-(--color-spider-error)";
}

export function Header(props: { class?: string }) {
  const store = useSessionStore();

  const streamCount = () =>
    orderedStreamMetrics(store.state(), store.meters()).length;

  const copyUrl = async () => {
    const url = store.state()?.participantUrl;
    if (url) {
      await navigator.clipboard.writeText(url);
      return;
    }
    const unsub = store.signaling.onParticipantUrl((u) => {
      unsub();
      void navigator.clipboard.writeText(u);
    });
    store.signaling.copyParticipantUrl();
  };

  return (
    <header
      class={`flex items-center gap-4 px-2 font-mono text-xs tabular-nums ${props.class ?? ""}`}
    >
      <span class="text-(--color-spider-muted)">spidercam/host</span>
      <span class="flex items-center gap-1">
        <span
          class="inline-block h-2 w-2 rounded-full"
          classList={{
            "bg-(--color-spider-accent)": store.state()?.outputHealthy === true,
            "bg-(--color-spider-error)": store.state()?.outputHealthy === false,
            "bg-(--color-spider-muted)": store.state() == null,
          }}
        />
        <span class="text-(--color-spider-muted)">output</span>
      </span>
      <span>{streamCount()} streams</span>
      <span>
        {store.state()?.globalLatencyMs != null
          ? `${Math.round(store.state()!.globalLatencyMs!)}ms`
          : "—"}
      </span>
      <Show when={anyProcessingOn(store.state())}>
        <span class={dspTone(store.state()?.enhancementBudgetPct ?? 0)}>
          DSP {store.state()?.enhancementBudgetPct.toFixed(0)}%
        </span>
      </Show>
      <button
        type="button"
        class="ml-auto rounded-(--radius-spider-sm) border border-(--color-spider-border) px-2 py-0.5 text-(--color-spider-muted) hover:text-(--color-spider-text)"
        onClick={() => void copyUrl()}
      >
        copy URL
      </button>
    </header>
  );
}
