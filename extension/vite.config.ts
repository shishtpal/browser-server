import { createBaseConfig } from '@browser-server/extension-core/vite.config.base'
import { mergeConfig } from 'vite'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const rootDir = dirname(fileURLToPath(import.meta.url))

export default mergeConfig(createBaseConfig(rootDir), {
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        popup: resolve(rootDir, 'popup.html'),
        options: resolve(rootDir, 'options.html'),
        background: resolve(rootDir, 'src/background.ts'),
      },
    },
  },
})
