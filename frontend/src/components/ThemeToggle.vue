<template>
  <div class="relative" ref="dropdownRef">
    <button
      type="button"
      @click="toggleOpen"
      class="inline-flex h-10 w-10 items-center justify-center rounded-2xl border border-gray-200 bg-white/70 text-slate-600 shadow-sm transition hover:bg-white hover:text-slate-900 dark:border-white/10 dark:bg-white/5 dark:text-slate-300 dark:hover:bg-white/10 dark:hover:text-white"
      :aria-label="`Theme: ${currentLabel}`"
      aria-haspopup="true"
      :aria-expanded="open"
    >
      <svg v-if="resolvedTheme === 'light'" class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
      </svg>
      <svg v-else class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
      </svg>
    </button>

    <transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0 scale-95 -translate-y-1"
      enter-to-class="opacity-100 scale-100 translate-y-0"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="opacity-100 scale-100 translate-y-0"
      leave-to-class="opacity-0 scale-95 -translate-y-1"
    >
      <div
        v-if="open"
        class="absolute right-0 top-full z-50 mt-2 w-44 overflow-hidden rounded-2xl border border-gray-200 bg-white p-1.5 shadow-xl dark:border-slate-700 dark:bg-slate-900"
        role="menu"
      >
        <button
          v-for="option in options"
          :key="option.value"
          type="button"
          @click="setTheme(option.value)"
          :class="[
            'flex w-full items-center gap-2.5 rounded-xl px-3 py-2.5 text-sm font-bold transition',
            theme === option.value
              ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300'
              : 'text-slate-700 hover:bg-gray-50 dark:text-slate-300 dark:hover:bg-slate-800'
          ]"
          role="menuitem"
        >
          <component :is="option.icon" class="h-4 w-4 shrink-0" />
          {{ option.label }}
          <svg v-if="theme === option.value" class="ml-auto h-4 w-4 text-indigo-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
          </svg>
        </button>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, h } from 'vue'

const SunIcon = () => h('svg', { class: 'h-4 w-4', fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
  h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z' })
])
const MoonIcon = () => h('svg', { class: 'h-4 w-4', fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
  h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z' })
])
const SystemIcon = () => h('svg', { class: 'h-4 w-4', fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' }, [
  h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z' })
])

const options = [
  { label: 'Light', value: 'light', icon: SunIcon },
  { label: 'Dark', value: 'dark', icon: MoonIcon },
  { label: 'System', value: 'system', icon: SystemIcon },
]

const theme = ref<'light' | 'dark' | 'system'>('system')
const open = ref(false)
const dropdownRef = ref<HTMLElement>()

const resolvedTheme = computed(() => {
  if (typeof window === 'undefined') return 'light'
  if (theme.value === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return theme.value
})

const currentLabel = computed(() => options.find(o => o.value === theme.value)?.label)

const applyTheme = () => {
  if (typeof document === 'undefined') return
  const isDark = resolvedTheme.value === 'dark'
  document.documentElement.classList.toggle('dark', isDark)
}

const setTheme = (value: 'light' | 'dark' | 'system') => {
  theme.value = value
  localStorage.setItem('theme', value)
  applyTheme()
  open.value = false
}

const toggleOpen = () => {
  open.value = !open.value
}

const handleClickOutside = (e: MouseEvent) => {
  if (dropdownRef.value && !dropdownRef.value.contains(e.target as Node)) {
    open.value = false
  }
}

const handleSystemChange = (e: MediaQueryListEvent) => {
  if (theme.value === 'system') {
    document.documentElement.classList.toggle('dark', e.matches)
  }
}

onMounted(() => {
  const saved = localStorage.getItem('theme') as 'light' | 'dark' | 'system' | null
  if (saved && ['light', 'dark', 'system'].includes(saved)) {
    theme.value = saved
  }
  applyTheme()
  document.addEventListener('click', handleClickOutside)
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', handleSystemChange)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', handleSystemChange)
})
</script>
