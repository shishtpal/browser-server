<template>
  <div class="flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-slate-700 dark:bg-slate-800/80">
    <button
      type="button"
      @click="$emit('update:view', 'list')"
      :class="['flex items-center gap-1.5 rounded-md px-3 py-1.5 text-[11px] font-black transition', view === 'list' ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-700 dark:text-white' : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200']"
    >
      <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M4 6h16M4 12h16M4 18h16" />
      </svg>
      List
    </button>
    <button
      type="button"
      @click="$emit('update:view', 'kanban')"
      :class="['flex items-center gap-1.5 rounded-md px-3 py-1.5 text-[11px] font-black transition', view === 'kanban' ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-700 dark:text-white' : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200']"
    >
      <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M9 17V7m0 10v-10m4 10V7m0 10v-10" />
      </svg>
      Kanban
    </button>
  </div>
</template>

<script setup lang="ts">
interface Props {
  view: 'list' | 'kanban'
}

defineProps<Props>()
defineEmits<{ 'update:view': [value: 'list' | 'kanban'] }>()
</script>
