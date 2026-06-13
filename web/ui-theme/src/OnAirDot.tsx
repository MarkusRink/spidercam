export function OnAirDot(props: { onAir: boolean; title?: string }) {
  return (
    <span
      class="h-2 w-2 shrink-0 rounded-full bg-(--color-spider-error)"
      classList={{ invisible: !props.onAir }}
      title={props.title ?? "On air"}
    />
  );
}
