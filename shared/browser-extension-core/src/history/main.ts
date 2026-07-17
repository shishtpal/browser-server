import { createApp } from 'vue'
import HistoryBrowserApp from './HistoryBrowserApp.vue'
import '../styles/tailwind.css'

const mountPoint = document.querySelector<HTMLDivElement>('#app')
if (!mountPoint) {
  throw new Error('History browser root element not found')
}

createApp(HistoryBrowserApp).mount(mountPoint)
