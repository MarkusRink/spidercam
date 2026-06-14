import type { HostConfig } from "@spidercam/protocol";
import { createMemo, onCleanup } from "solid-js";
import { useSessionStore } from "../stores/session-store.js";

const DEBOUNCE_MS = 150;

function panelShell(className?: string) {
  return `flex min-h-0 flex-col gap-3 overflow-y-auto rounded-(--radius-spider) border border-(--color-spider-border) bg-(--color-spider-surface) p-2 text-[11px] font-mono ${className ?? ""}`;
}

function SliderRow(props: {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  format?: (v: number) => string;
  onInput: (v: number) => void;
}) {
  const display = () => props.format?.(props.value) ?? String(props.value);
  return (
    <label class="flex flex-col gap-0.5 text-[11px]">
      <span class="flex justify-between text-(--color-spider-muted)">
        <span>{props.label}</span>
        <span class="tabular-nums">{display()}</span>
      </span>
      <input
        type="range"
        min={props.min}
        max={props.max}
        step={props.step ?? 1}
        value={props.value}
        onInput={(e) => props.onInput(Number(e.currentTarget.value))}
        class="w-full accent-(--color-spider-accent)"
      />
    </label>
  );
}

function useDebouncedConfig() {
  const store = useSessionStore();
  let timer: ReturnType<typeof setTimeout> | null = null;

  const push = (partial: Partial<HostConfig>) => {
    if (timer) {
      clearTimeout(timer);
    }
    timer = setTimeout(() => {
      store.sendConfig(partial);
      timer = null;
    }, DEBOUNCE_MS);
  };

  onCleanup(() => {
    if (timer) {
      clearTimeout(timer);
    }
  });

  return { store, push };
}

function DevicesSection() {
  const store = useSessionStore();
  const capture = () => store.state()?.capture;
  const devices = () => store.captureDevices();

  const setDevice = (field: "micId" | "cameraId" | "sinkId", id: string) => {
    const selection =
      field === "micId"
        ? { micId: id }
        : field === "cameraId"
          ? { cameraId: id }
          : { sinkId: id };
    store.signaling.setCaptureDevices(selection);
  };

  return (
    <section class="flex flex-col gap-2">
      <h2 class="text-(--color-spider-muted)">Devices</h2>
      <label class="flex flex-col gap-0.5">
        <span class="text-(--color-spider-muted)">Microphone</span>
        <select
          class="rounded-(--radius-spider-sm) border border-(--color-spider-border) bg-(--color-spider-bg) px-1 py-0.5"
          value={capture()?.micId ?? ""}
          onChange={(e) => setDevice("micId", e.currentTarget.value)}
        >
          {devices()?.mics.map((d) => (
            <option value={d.id}>{d.label}</option>
          ))}
        </select>
      </label>
      <label class="flex flex-col gap-0.5">
        <span class="text-(--color-spider-muted)">Webcam</span>
        <select
          class="rounded-(--radius-spider-sm) border border-(--color-spider-border) bg-(--color-spider-bg) px-1 py-0.5"
          value={capture()?.cameraId ?? ""}
          onChange={(e) => setDevice("cameraId", e.currentTarget.value)}
        >
          {devices()?.cameras.map((d) => (
            <option value={d.id}>{d.label}</option>
          ))}
        </select>
      </label>
      <label class="flex flex-col gap-0.5">
        <span class="text-(--color-spider-muted)">Playback output</span>
        <select
          class="rounded-(--radius-spider-sm) border border-(--color-spider-border) bg-(--color-spider-bg) px-1 py-0.5"
          value={capture()?.sinkId ?? ""}
          onChange={(e) => setDevice("sinkId", e.currentTarget.value)}
        >
          {devices()?.sinks.map((d) => (
            <option value={d.id}>{d.label}</option>
          ))}
        </select>
        <span class="text-[10px] text-(--color-spider-muted)">
          Teams meeting audio should play to this device.
        </span>
      </label>
    </section>
  );
}

function MixerSection() {
  const { store, push } = useDebouncedConfig();

  return (
    <section class="flex flex-col gap-2">
      <h2 class="text-(--color-spider-muted)">Mixer</h2>
      <SliderRow
        label="Hold time"
        value={store.config.audioHoldMs}
        min={200}
        max={800}
        step={10}
        format={(v) => `${v} ms`}
        onInput={(v) => {
          store.updateConfig({ audioHoldMs: v });
          push({ audioHoldMs: v });
        }}
      />
      <SliderRow
        label="Crossfade"
        value={store.config.crossfadeMs}
        min={50}
        max={200}
        step={5}
        format={(v) => `${v} ms`}
        onInput={(v) => {
          store.updateConfig({ crossfadeMs: v });
          push({ crossfadeMs: v });
        }}
      />
      <SliderRow
        label="Ducking"
        value={store.config.referenceDuckDb}
        min={-12}
        max={0}
        step={1}
        format={(v) => `${v} dB`}
        onInput={(v) => {
          store.updateConfig({ referenceDuckDb: v });
          push({ referenceDuckDb: v });
        }}
      />
      <p class="text-[10px] text-(--color-spider-muted)">
        Attenuates room mics while remote Teams speech is active. Set to 0 dB to
        disable.
      </p>
      <SliderRow
        label="Switch margin"
        value={store.config.switchMargin}
        min={0.5}
        max={2}
        step={0.1}
        format={(v) => v.toFixed(1)}
        onInput={(v) => {
          store.updateConfig({ switchMargin: v });
          push({ switchMargin: v });
        }}
      />
    </section>
  );
}

function ScoreWeightsSection() {
  const { store, push } = useDebouncedConfig();
  const weights = createMemo(() => store.config.scoreWeights);

  return (
    <section class="flex flex-col gap-2">
      <h2 class="text-(--color-spider-muted)">Score weights</h2>
      <SliderRow
        label="Level"
        value={weights().level}
        min={0}
        max={1}
        step={0.05}
        format={(v) => v.toFixed(2)}
        onInput={(v) => {
          store.updateConfig({ scoreWeights: { ...weights(), level: v } });
          push({ scoreWeights: { level: v } } as Partial<HostConfig>);
        }}
      />
      <SliderRow
        label="SNR"
        value={weights().snr}
        min={0}
        max={1}
        step={0.05}
        format={(v) => v.toFixed(2)}
        onInput={(v) => {
          store.updateConfig({ scoreWeights: { ...weights(), snr: v } });
          push({ scoreWeights: { snr: v } } as Partial<HostConfig>);
        }}
      />
      <SliderRow
        label="VAD"
        value={weights().vad}
        min={0}
        max={1}
        step={0.05}
        format={(v) => v.toFixed(2)}
        onInput={(v) => {
          store.updateConfig({ scoreWeights: { ...weights(), vad: v } });
          push({ scoreWeights: { vad: v } } as Partial<HostConfig>);
        }}
      />
      <SliderRow
        label="Priority"
        value={weights().priority}
        min={0}
        max={1}
        step={0.05}
        format={(v) => v.toFixed(2)}
        onInput={(v) => {
          store.updateConfig({ scoreWeights: { ...weights(), priority: v } });
          push({ scoreWeights: { priority: v } } as Partial<HostConfig>);
        }}
      />
      <SliderRow
        label="Echo penalty"
        value={weights().echoPenalty}
        min={0}
        max={1}
        step={0.05}
        format={(v) => v.toFixed(2)}
        onInput={(v) => {
          store.updateConfig({
            scoreWeights: { ...weights(), echoPenalty: v },
          });
          push({ scoreWeights: { echoPenalty: v } } as Partial<HostConfig>);
        }}
      />
    </section>
  );
}

export function DevicesPanel(props: { class?: string }) {
  return (
    <aside class={panelShell(props.class)}>
      <DevicesSection />
    </aside>
  );
}

export function MixerPanel(props: { class?: string }) {
  return (
    <aside class={panelShell(props.class)}>
      <MixerSection />
      <ScoreWeightsSection />
    </aside>
  );
}

export function SettingsPanel(props: { class?: string }) {
  return (
    <aside class={panelShell(props.class)}>
      <DevicesSection />
      <MixerSection />
      <ScoreWeightsSection />
    </aside>
  );
}
