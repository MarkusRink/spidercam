import { StateTimeline, VerticalVuMeter } from "@spidercam/ui-theme";
import { onCleanup, onMount } from "solid-js";
import { PreviewStream } from "../adapters/preview-stream.js";
import { useSessionStore } from "../stores/session-store.js";

export function OutputPreview() {
  const store = useSessionStore();
  let canvasRef: HTMLCanvasElement | undefined;
  let preview: PreviewStream | null = null;

  onMount(() => {
    if (!canvasRef) {
      return;
    }
    preview = new PreviewStream(canvasRef);
    preview.connect();
    onCleanup(() => preview?.disconnect());
  });

  const meters = () => store.meters();

  return (
    <div class="flex min-h-0 flex-col gap-2 rounded-(--radius-spider) border border-(--color-spider-border) bg-(--color-spider-surface) p-2">
      <div class="flex min-h-0 flex-1 gap-3">
        <canvas
          ref={(el) => {
            canvasRef = el;
          }}
          width={640}
          height={360}
          class="max-h-full w-auto shrink-0 rounded-(--radius-spider-sm) bg-black object-contain"
        />
        <div class="flex gap-3">
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
