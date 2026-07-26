<template>
  <div class="group relative inline-flex items-center gap-1.5">
    <div class="relative">
      <span class="pointer-events-none absolute inset-y-0 left-2 flex items-center text-slate-400 transition group-focus-within:text-indigo-500 dark:text-slate-500 dark:group-focus-within:text-indigo-400">
        <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
          <rect x="3" y="4" width="18" height="18" rx="2" />
          <path d="M16 2v4M8 2v4M3 10h18" />
        </svg>
      </span>
      <input
        type="date"
        :value="modelValue"
        @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value || null)"
        class="h-7 w-[8.5rem] cursor-pointer rounded-lg border border-gray-200 bg-white pl-7 pr-2 text-[11px] font-black text-slate-700 shadow-sm transition placeholder:text-slate-400 hover:border-indigo-300 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:border-indigo-500/50 dark:focus:border-indigo-500 dark:focus:ring-indigo-900/30"
      />
    </div>
    <button
      v-if="modelValue"
      type="button"
      @click="$emit('update:modelValue', null)"
      class="grid h-6 w-6 place-items-center rounded-lg text-slate-400 transition hover:bg-red-50 hover:text-red-500 dark:text-slate-500 dark:hover:bg-red-900/20 dark:hover:text-red-400"
      title="Clear date"
    >
      <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2.5">
        <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
interface Props {
  modelValue: string | null
}

defineProps<Props>()
defineEmits<{ 'update:modelValue': [value: string | null] }>()
</script>
