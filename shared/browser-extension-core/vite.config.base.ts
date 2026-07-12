import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

/**
 * Creates a shared Vite base configuration for extension wrapper packages.
 * Each wrapper (Chrome, Firefox) imports this and merges with its own
 * build.outDir, entry points, and any browser-specific overrides.
 *
 * @param rootDir - The wrapper package's root directory (pass `dirname(fileURLToPath(import.meta.url))`)
 */
export function createBaseConfig(_rootDir: string) {
  return defineConfig({
    base: './',
    plugins: [vue(), tailwindcss()],
    build: {
      rollupOptions: {
        output: {
          // background entry → background.js at root; all others → assets/
          entryFileNames: (chunkInfo) =>
            chunkInfo.name === 'background' ? 'background.js' : 'assets/[name]-[hash].js',
          chunkFileNames: 'assets/[name]-[hash].js',
          assetFileNames: 'assets/[name]-[hash][extname]',
        },
      },
    },
  })
}
