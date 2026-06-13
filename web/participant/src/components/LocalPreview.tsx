import { createEffect, onCleanup } from "solid-js";
import { useStore } from "../stores/store-context.js";

function rmsToPct(rms: number): number {
  const db = rms <= 0 ? -60 : Math.max(-60, 20 * Math.log10(rms));
  return Math.min(100, Math.max(0, ((db + 60) / 60) * 100));
}

export function LocalPreview() {
  const store = useStore();
  let videoRef: HTMLVideoElement | undefined;
  let meterRef: HTMLDivElement | undefined;

  createEffect(() => {
    const stream = store.state.localStream;
    if (videoRef && stream) {
      videoRef.srcObject = stream;
    }
  });

  createEffect(() => {
    const stream = store.state.localStream;
    if (!stream || !meterRef) {
      return;
    }

    let raf = 0;
    let ctx: AudioContext | null = null;
    let analyser: AnalyserNode | null = null;
    const data = new Uint8Array(256);

    try {
      ctx = new AudioContext();
      const source = ctx.createMediaStreamSource(stream);
      analyser = ctx.createAnalyser();
      analyser.fftSize = 256;
      source.connect(analyser);
    } catch {
      return;
    }

    const sample = () => {
      if (!analyser || !meterRef) {
        return;
      }
      if (!store.state.media.audioEnabled) {
        meterRef.style.height = "0%";
        raf = requestAnimationFrame(sample);
        return;
      }
      analyser.getByteTimeDomainData(data);
      let sum = 0;
      for (let i = 0; i < data.length; i++) {
        const v = (data[i]! - 128) / 128;
        sum += v * v;
      }
      const rms = Math.sqrt(sum / data.length);
      meterRef.style.height = `${rmsToPct(rms)}%`;
      raf = requestAnimationFrame(sample);
    };

    raf = requestAnimationFrame(sample);

    onCleanup(() => {
      cancelAnimationFrame(raf);
      void ctx?.close();
    });
  });

  const dimmed = () => !store.state.media.videoEnabled;

  return (
    <section class="space-y-2">
      <div
        class="relative aspect-video overflow-hidden rounded-(--radius-spider) border border-(--color-spider-border) bg-(--color-spider-surface)"
        classList={{ "opacity-40": dimmed() }}
      >
        <video
          ref={(el) => {
            videoRef = el;
          }}
          class="h-full w-full object-cover"
          autoplay
          playsinline
          muted
        />
        <div class="absolute bottom-2 right-2 flex h-14 w-3 flex-col justify-end overflow-hidden rounded-(--radius-spider-sm) bg-(--color-spider-meter-track)">
          <div
            ref={(el) => {
              meterRef = el;
            }}
            class="w-full bg-(--color-spider-accent) transition-[height] duration-[16ms] linear"
            style={{ height: "0%" }}
          />
        </div>
      </div>
    </section>
  );
}
