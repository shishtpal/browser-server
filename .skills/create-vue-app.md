---
name: create-vue-app
label: Create Vue App
description: Scaffold a new Vue 3 app with Vite, TypeScript, Pinia, Vue Router (hash mode), and Tailwind CSS v4
category: Scaffolding
tags: [vite, vue, typescript, pinia, vue-router, tailwindcss, scaffold, composition-api, hash-routing]
tools:
  - execute_command
  - read_file
  - write_file
  - edit_file
  - directory_tree
---

You are an expert Vue 3 scaffolding assistant. Your job is to create a new Vue application using **Vite + TypeScript + Pinia + Vue Router (hash-based) + Tailwind CSS v4**, always written with the **Composition API** (`<script setup lang="ts">`).

## Stack Requirements

- **Build tool**: Vite (latest)
- **Language**: TypeScript (strict mode)
- **Framework**: Vue 3 with Composition API only (no Options API)
- **State**: Pinia (with setup-style stores)
- **Routing**: Vue Router 4 with `createWebHashHistory` (hash-based)
- **Styling**: Tailwind CSS v4 (using the new `@tailwindcss/vite` plugin, no `tailwind.config.js` unless needed)
- **Package manager**: Prefer `pnpm`, fall back to `npm` if unavailable

## Setup Steps

### 1. Scaffold the project

```bash
npm create vite@latest <app-name> -- --template vue-ts
cd <app-name>
```

### 2. Install dependencies

```bash
npm install pinia vue-router@4
npm install -D tailwindcss @tailwindcss/vite
npm install -D @lucide/vue
```

### 3. Configure Vite (`vite.config.ts`)

```ts
import path from 'node:path'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
})
```

### 4. Configure Tailwind v4 (`src/style.css`)

Tailwind v4 uses a single `@import` — no `@tailwind base/components/utilities` directives.

```css
@import "tailwindcss";
```

### 5. Create the router with hash history (`src/router/index.ts`)

```ts
import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'home', component: () => import('@/views/HomeView.vue') },
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
})
```

### 6. Create Pinia stores

Example setup-style store (`src/stores/counter.ts`):

```ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useCounterStore = defineStore('counter', () => {
  const count = ref(0)
  const double = computed(() => count.value * 2)
  const increment = () => { count.value++ }
  return { count, double, increment }
})
```

### 7. Wire everything in `src/main.ts`

```ts
import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router'
import { pinia } from './stores'
import './style.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.mount('#app')
```

### 8. Root component (`src/App.vue`)

```vue
<script setup lang="ts"></script>

<template>
  <div class="min-h-screen bg-gray-50 text-gray-900">
    <RouterView />
  </div>
</template>
```

### 9. Example view (`src/views/HomeView.vue`)

```vue
<script setup lang="ts">
import { useCounterStore } from '@/stores/counter'
const counter = useCounterStore()
</script>

<template>
  <main class="p-8">
    <h1 class="text-3xl font-bold">Hello Vue 3</h1>
    <button
      class="mt-4 rounded bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
      @click="counter.increment"
    >
      Count: {{ counter.count }} (x2 = {{ counter.double }})
    </button>
  </main>
</template>
```

### 10. Update `tsconfig.json` path alias

Add under `compilerOptions`:

```json
"baseUrl": ".",
"paths": { "@/*": ["src/*"] }
```

### 11. Clean up optional files
- Remove `src/assets/main.css` if it's replaced by `src/style.css` (and update references).
- Remove boilerplate demo code from Vite template that is no longer needed (e.g. `HelloWorld.vue`, old CSS) while keeping things clean.
- Ensure `src/components/` exists (can be empty initially).

### 12. Save Checkpoint using Git
- Stage and Commit using git tools with commit message: `Initial Commit`

### 13. Unload Skill
- Finally deactivate this skill using tool


## Recommended Project Structure

```
src/
├── apis/
├── assets/
├── components/       # Reusable UI components
├── composables/      # Reusable composition functions (useXxx)
├── router/
│   └── index.ts
├── stores/           # Pinia stores (setup style)
├── views/            # Route-level components
├── types/            # Shared TypeScript types
├── App.vue
├── main.ts
└── style.css
```

## Conventions to Enforce

- **Always** use `<script setup lang="ts">` — never Options API
- **Always** use setup-style Pinia stores (function form), not options-style
- **Always** use `createWebHashHistory()` — never `createWebHistory()`
- Prefer `ref`/`computed` over `reactive` for primitives and simple state
- Use `@/` alias for all `src/` imports
- Use Tailwind utility classes; avoid custom CSS unless necessary
- Lazy-load route components with dynamic `import()`
- Use `defineProps<T>()` and `defineEmits<T>()` with TypeScript generics — no runtime declarations

## Verification

After scaffolding, run:

```bash
npm run dev
```

Confirm:
1. Dev server starts without errors
2. Browser opens to `http://localhost:5173/#/` (note the `#` — hash routing)
3. Tailwind classes render correctly
4. Pinia counter increments on click
5. `npm run build` completes without TypeScript errors
```