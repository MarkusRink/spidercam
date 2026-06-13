import { Show } from "solid-js";
import { useStore } from "../stores/store-context.js";

export function LostHostBanner() {
  const store = useStore();

  const visible = () => store.state.phase === "reconnecting";

  const countdownSec = () => Math.ceil(store.state.reconnectCountdownMs / 1000);

  return (
    <Show when={visible()}>
      <section class="space-y-3 rounded-(--radius-spider) border border-(--color-spider-error)/40 bg-(--color-spider-surface) p-3">
        <p class="font-mono text-sm text-(--color-spider-error)">
          Lost host connection
        </p>
        <p class="font-mono text-[11px] text-(--color-spider-muted)">
          Retrying in {countdownSec()} s… (attempt{" "}
          {store.state.reconnectAttempt})
        </p>
        <div class="flex gap-2">
          <button
            type="button"
            class="rounded-(--radius-spider) border border-(--color-spider-accent) px-3 py-1.5 font-mono text-[11px] text-(--color-spider-accent)"
            onClick={() => store.retryNow()}
          >
            Retry now
          </button>
          <button
            type="button"
            class="rounded-(--radius-spider) border border-(--color-spider-border) px-3 py-1.5 font-mono text-[11px] text-(--color-spider-muted)"
            onClick={() => store.disconnect()}
          >
            Disconnect
          </button>
        </div>
      </section>
    </Show>
  );
}
