import { OnAirDot, isRouted } from "@spidercam/ui-theme";
import { Show } from "solid-js";
import { useStore } from "../stores/store-context.js";

export function ParticipantHeader() {
  const store = useStore();

  const statusLine = () => {
    switch (store.state.phase) {
      case "connected":
        return "connected · WebRTC active";
      case "connecting":
        return "connecting…";
      case "reconnecting":
        return "reconnecting…";
      default:
        return "disconnected · local preview only";
    }
  };

  const routed = () => {
    const view = store.state.view;
    const id = store.state.clientId;
    if (!view || !id || store.state.phase !== "connected") {
      return false;
    }
    return isRouted(view, id);
  };

  return (
    <header class="space-y-1 border-b border-(--color-spider-border) pb-3">
      <div class="flex items-center gap-2">
        <input
          type="text"
          class="min-w-0 flex-1 rounded-(--radius-spider) border border-(--color-spider-border) bg-(--color-spider-surface) px-2 py-1.5 text-sm outline-none focus:border-(--color-spider-accent)"
          value={store.state.displayName}
          onInput={(e) => store.setDisplayName(e.currentTarget.value)}
          aria-label="Display name"
        />
        <OnAirDot onAir={routed()} title="Routed to Teams output" />
      </div>
      <Show when={store.state.clientId}>
        {(id) => (
          <p class="truncate font-mono text-[10px] text-(--color-spider-muted)">
            {id()}
          </p>
        )}
      </Show>
      <p class="font-mono text-[10px] text-(--color-spider-muted)">
        {statusLine()}
      </p>
    </header>
  );
}
