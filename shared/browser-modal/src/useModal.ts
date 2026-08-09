import type { ModalApi, ModalOptions } from './types';
import { pushModal, pendingModalCount } from './store';

/**
 * Imperative modal API. Call from any setup scope or event handler:
 *
 * ```ts
 * const { confirm, confirmDelete } = useModal();
 * if (await confirmDelete('Delete this todo?')) { … }
 * ```
 *
 * Requires exactly one `<ModalHost />` mounted at the app root
 * (e.g. inside AppModalHost.vue in the layout).
 */
export function useModal(): ModalApi {
  return {
    confirm: (title: string, message?: string, options?: ModalOptions) =>
      pushModal('confirm', title, message, options),

    confirmDelete: (title: string, message?: string, options?: ModalOptions) =>
      pushModal('danger', title, message, options),

    alert: (title: string, message?: string, options?: ModalOptions) =>
      pushModal('info', title, message, options).then(() => undefined),

    pendingCount: pendingModalCount,
  };
}

export type { ModalApi, ModalKind, ModalOptions, ModalRequest } from './types';
