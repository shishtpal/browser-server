import type { ModalKind, ModalOptions, ModalRequest } from './types';
import { computed, readonly, ref } from 'vue';

/**
 * The request queue. Module-level singleton so every `useModal()` caller and
 * the single <ModalHost /> share the same stack regardless of component tree.
 */

const queue = ref<ModalRequest[]>([]);
let nextId = 0;

/** The request currently on screen (first in the queue). */
export const activeModal = computed(() => queue.value[0] ?? null);

/** Read-only view of the queue for debugging. */
export const modalQueue = readonly(queue);

export const pendingModalCount = () => queue.value.length;

/**
 * Enqueue a request and get a promise that resolves when the user answers.
 * Alert dialogs resolve with `true` on dismiss (they only have one button).
 */
export function pushModal(
  kind: ModalKind,
  title: string,
  message?: string,
  options: ModalOptions = {},
): Promise<boolean> {
  return new Promise<boolean>((resolve) => {
    const request: ModalRequest = {
      id: ++nextId,
      kind,
      title,
      message,
      cancelText: options.cancelText ?? 'Cancel',
      confirmText:
        options.confirmText ?? (kind === 'danger' ? 'Delete' : kind === 'info' ? 'OK' : 'Confirm'),
      persistent: options.persistent ?? false,
      panelClass: options.panelClass,
      resolve,
    };
    queue.value = [...queue.value, request];

    // Auto-focus scroll target happens in the host.
  });
}

/** Settle and remove the active request (`true` = confirmed, `false` = dismissed). */
export function settleModal(id: number, value: boolean): void {
  const index = queue.value.findIndex((r) => r.id === id);
  if (index === -1) return;
  const [request] = queue.value.splice(index, 1);
  // Resolve on the next microtask so the exit transition can start first.
  request.resolve(kindResolvesTrueOnDismiss(request.kind) ? true : value);
}

function kindResolvesTrueOnDismiss(kind: ModalKind): boolean {
  // `info` (alert) dialogs have a single action — dismissing IS the answer.
  return kind === 'info';
}
