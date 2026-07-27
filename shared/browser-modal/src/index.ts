import type { App } from 'vue'
import ModalHost from './ModalHost.vue'
import { modal } from './service'

export { modal, useModal, fire, close, closeAll, isVisible } from './service'
export { default as ModalHost } from './ModalHost.vue'
export { default as ModalDialog } from './ModalDialog.vue'
export { default as ModalIcon } from './ModalIcon.vue'
export * from './types'

/** Vue plugin — registers ModalHost globally and adds $modal to all instances. */
export const ModalPlugin = {
  install(app: App) {
    app.component('ModalHost', ModalHost)
    app.config.globalProperties.$modal = modal
  },
}

export default ModalPlugin

declare module 'vue' {
  interface ComponentCustomProperties {
    $modal: typeof modal
  }
}
