import { Show } from "solid-js";
import { DeviceRow } from "./DeviceRow.js";
import { LocalPreview } from "./LocalPreview.js";
import { LostHostBanner } from "./LostHostBanner.js";
import { ParticipantHeader } from "./ParticipantHeader.js";
import { RoomStatus } from "./RoomStatus.js";
import { useStore } from "../stores/store-context.js";

export function ParticipantShell() {
  const store = useStore();

  const showConnect = () => store.state.phase === "idle";

  const showDisconnect = () =>
    store.state.phase === "connected" || store.state.phase === "connecting";

  const showRoom = () => store.state.phase === "connected";

  const showPlaceholder = () =>
    store.state.phase === "idle" ||
    store.state.phase === "connecting" ||
    store.state.phase === "reconnecting";

  return (
    <div class="mx-auto flex min-h-dvh max-w-[420px] flex-col gap-4 p-4">
      <ParticipantHeader />
      <LocalPreview />
      <section class="space-y-2">
        <DeviceRow label="Microphone" kind="audio" />
        <DeviceRow label="Camera" kind="video" />
        <div class="pt-1">
          <Show
            when={showConnect()}
            fallback={
              <Show when={showDisconnect()}>
                <button
                  type="button"
                  class="w-full rounded-(--radius-spider) border border-(--color-spider-border) py-2 font-mono text-[11px] text-(--color-spider-muted) hover:border-(--color-spider-error) hover:text-(--color-spider-error)"
                  onClick={() => store.disconnect()}
                  disabled={store.state.phase === "connecting"}
                >
                  Disconnect
                </button>
              </Show>
            }
          >
            <button
              type="button"
              class="w-full rounded-(--radius-spider) border border-(--color-spider-accent) py-2 font-mono text-[11px] text-(--color-spider-accent) hover:bg-(--color-spider-accent)/10"
              onClick={() => void store.connect()}
              disabled={store.state.phase === "reconnecting"}
            >
              Connect
            </button>
          </Show>
        </div>
      </section>

      <Show when={store.state.phase === "reconnecting"}>
        <LostHostBanner />
      </Show>

      <Show when={showRoom()}>
        <RoomStatus />
      </Show>

      <Show when={showPlaceholder() && store.state.phase !== "reconnecting"}>
        <section class="space-y-2 border-t border-(--color-spider-border) pt-3 font-mono text-[11px] text-(--color-spider-muted)">
          <div class="flex gap-2">
            <span>On air:</span>
            <span>—</span>
          </div>
          <div class="flex gap-2">
            <span>Loop delay ·</span>
            <span>—</span>
          </div>
          <div class="flex gap-2">
            <span>SNR ·</span>
            <span>—</span>
          </div>
        </section>
      </Show>
    </div>
  );
}
