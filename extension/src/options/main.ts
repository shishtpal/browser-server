import { createApp } from 'vue'
import OptionsApp from './OptionsApp.vue'
import '../styles/tailwind.css'

const mountPoint = document.querySelector<HTMLDivElement>('#app')
if (!mountPoint) {
  throw new Error('Options root element not found')
}

createApp(OptionsApp).mount(mountPoint)
