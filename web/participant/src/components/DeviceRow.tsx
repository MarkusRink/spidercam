import { For } from "solid-js";
import { useStore } from "../stores/store-context.js";

export function DeviceRow(props: { label: string; kind: "audio" | "video" }) {
  const store = useStore();

  const devices = () =>
    props.kind === "audio"
      ? store.state.devices.mics
      : store.state.devices.cameras;

  const selectedId = () =>
    props.kind === "audio"
      ? store.state.media.micId
      : store.state.media.cameraId;

  const enabled = () =>
    props.kind === "audio"
      ? store.state.media.audioEnabled
      : store.state.media.videoEnabled;

  const onSelect = (deviceId: string) => {
    if (props.kind === "audio") {
      void store.selectMic(deviceId);
    } else {
      void store.selectCamera(deviceId);
    }
  };

  const onToggle = () => {
    if (props.kind === "audio") {
      void store.setAudioEnabled(!store.state.media.audioEnabled);
    } else {
      void store.setVideoEnabled(!store.state.media.videoEnabled);
    }
  };

  return (
    <div class="flex items-center gap-2">
      <label class="w-20 shrink-0 font-mono text-[10px] text-(--color-spider-muted)">
        {props.label}
      </label>
      <select
        class="min-w-0 flex-1 rounded-(--radius-spider) border border-(--color-spider-border) bg-(--color-spider-surface) px-2 py-1 font-mono text-[11px] outline-none focus:border-(--color-spider-accent)"
        value={selectedId()}
        onChange={(e) => onSelect(e.currentTarget.value)}
      >
        <For each={devices()}>
          {(device) => (
            <option value={device.deviceId}>
              {device.label || `${props.label} ${device.deviceId.slice(0, 8)}`}
            </option>
          )}
        </For>
      </select>
      <button
        type="button"
        class="shrink-0 rounded-(--radius-spider) border px-2 py-1 font-mono text-[10px] transition-colors"
        classList={{
          "border-(--color-spider-accent) text-(--color-spider-accent)":
            enabled(),
          "border-(--color-spider-border) text-(--color-spider-muted)":
            !enabled(),
        }}
        onClick={onToggle}
      >
        {enabled() ? "on" : "off"}
      </button>
    </div>
  );
}
