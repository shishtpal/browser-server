import { createApp } from 'vue'
import PopupApp from './PopupApp.vue'
import '../styles/tailwind.css'

const mountPoint = document.querySelector<HTMLDivElement>('#app')
if (!mountPoint) {
  throw new Error('Popup root element not found')
}

createApp(PopupApp, { initialPanel: 'history' }).mount(mountPoint)
