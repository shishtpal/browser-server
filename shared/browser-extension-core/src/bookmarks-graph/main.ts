import { createApp } from 'vue'
import BookmarksGraphApp from './BookmarksGraphApp.vue'
import '../styles/tailwind.css'

const mountPoint = document.querySelector<HTMLDivElement>('#app')
if (!mountPoint) {
  throw new Error('Bookmarks graph root element not found')
}

createApp(BookmarksGraphApp).mount(mountPoint)
