import { StateTimeline, VerticalVuMeter } from "@spidercam/ui-theme";
import { onCleanup } from "solid-js";
import { PreviewStream } from "../adapters/preview-stream.js";
import { useSessionStore } from "../stores/session-store.js";

export function OutputPreview(props: { class?: string }) {
  const store = useSessionStore();
  let preview: PreviewStream | null = null;

  onCleanup(() => preview?.disconnect());

  const bindCanvas = (el: HTMLCanvasElement) => {
    if (!el) {
      return;
    }
    preview?.disconnect();
    preview = new PreviewStream(el);
    preview.connect();
  };

  const meters = () => store.meters();

  return (
    <div class={`flex min-h-0 flex-col gap-2 rounded-(--radius-spider) border border-(--color-spider-border) bg-(--color-spider-surface) p-2 ${props.class ?? ""}`}>
      <div class="flex min-h-0 min-w-0 flex-1 gap-3">
        <div class="flex min-h-0 min-w-0 flex-1 items-center justify-center">
          <canvas
            ref={bindCanvas}
            width={640}
            height={360}
            class="max-h-full max-w-full h-auto w-auto rounded-(--radius-spider-sm) bg-black object-contain"
          />
        </div>
        <div class="flex shrink-0 gap-3">
          <div class="flex flex-col items-center gap-1">
            <span class="text-[10px] text-(--color-spider-muted)">OUT</span>
            <VerticalVuMeter
              rmsDbfs={meters()?.outLevelDbfs ?? -60}
              peakDbfs={meters()?.outPeakDbfs ?? -60}
              compact={false}
            />
          </div>
          <div class="flex flex-col items-center gap-1">
            <span class="text-[10px] text-(--color-spider-muted)">REF</span>
            <VerticalVuMeter
              rmsDbfs={meters()?.refRmsDbfs ?? -60}
              peakDbfs={meters()?.refPeakDbfs ?? -60}
              compact={false}
            />
          </div>
        </div>
      </div>
      <StateTimeline samples={store.timeline()} />
    </div>
  );
}
