import { createBaseConfig } from '@browser-server/extension-core/vite.config.base'
import { defineConfig, mergeConfig } from 'vite'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const rootDir = dirname(fileURLToPath(import.meta.url))

export default defineConfig(({ mode }) => {
  if (mode === 'background') {
    return mergeConfig(createBaseConfig(rootDir), {
      build: {
        outDir: 'dist',
        emptyOutDir: false,
        rollupOptions: {
          input: resolve(rootDir, 'src/background.ts'),
          output: {
            format: 'iife',
            entryFileNames: 'background.js',
          },
        },
      },
    })
  }

  return mergeConfig(createBaseConfig(rootDir), {
    build: {
      outDir: 'dist',
      emptyOutDir: true,
      rollupOptions: {
        input: {
          popup: resolve(rootDir, 'popup.html'),
          options: resolve(rootDir, 'options.html'),
        },
      },
    },
  })
})
