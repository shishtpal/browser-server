import { computed, ref } from "vue";

const MIN_ZOOM = 1;
const MAX_ZOOM = 8;

/** Wheel/button zoom with pointer panning for a single displayed image. */
export function useImageZoom() {
  const zoom = ref(MIN_ZOOM);
  const offsetX = ref(0);
  const offsetY = ref(0);
  const panning = ref(false);
  let start = { x: 0, y: 0, ox: 0, oy: 0 };

  const canPan = computed(() => zoom.value > MIN_ZOOM);

  const transform = computed(() => ({
    transform: `translate(${offsetX.value}px, ${offsetY.value}px) scale(${zoom.value})`,
    transition: panning.value ? "none" : "transform 120ms ease-out",
  }));

  function reset() {
    zoom.value = MIN_ZOOM;
    offsetX.value = 0;
    offsetY.value = 0;
  }

  function setZoom(value: number) {
    zoom.value = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, Number(value.toFixed(2))));
    if (zoom.value === MIN_ZOOM) {
      offsetX.value = 0;
      offsetY.value = 0;
    }
  }

  function onWheel(e: WheelEvent) {
    setZoom(zoom.value * (e.deltaY < 0 ? 1.15 : 1 / 1.15));
  }

  function startPan(e: PointerEvent) {
    if (!canPan.value) return;
    panning.value = true;
    start = { x: e.clientX, y: e.clientY, ox: offsetX.value, oy: offsetY.value };
    (e.currentTarget as Element).setPointerCapture(e.pointerId);
  }

  function onPan(e: PointerEvent) {
    if (!panning.value) return;
    offsetX.value = start.ox + (e.clientX - start.x);
    offsetY.value = start.oy + (e.clientY - start.y);
  }

  function endPan() {
    panning.value = false;
  }

  return { zoom, panning, canPan, transform, reset, setZoom, onWheel, startPan, onPan, endPan };
}
