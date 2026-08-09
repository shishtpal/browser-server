/**
 * @browser-server/shared-modal — imperative modal service.
 *
 * Mount <ModalHost /> once at the app root (the frontend does this via
 * AppModalHost.vue inside the layout), then call `useModal()` anywhere:
 *
 * ```ts
 * const { confirm, confirmDelete, alert } = useModal();
 * if (await confirmDelete('Delete this bookmark?')) remove(id);
 * ```
 */
export { default as ModalHost } from './ModalHost.vue';
export { default as ConfirmDialog } from './ConfirmDialog.vue';
export { useModal } from './useModal';
export { activeModal, modalQueue, pendingModalCount, pushModal, settleModal } from './store';
export type { ModalApi, ModalKind, ModalOptions, ModalRequest } from './types';
