<template>
  <button
    type="button"
    class="group flex w-full items-center gap-1.5 rounded-md border px-2 py-1 text-left text-xs font-medium transition-all duration-150 active:scale-[0.98]"
    :class="[chipStyle, isDragging ? 'opacity-50 cursor-grabbing' : 'cursor-grab']"
    draggable="true"
    @click.stop="onClick"
    @dragstart.stop="onDragStart"
    @dragend.stop="onDragEnd"
  >
    <!-- Priority Dot / Status Indicator -->
    <span 
      class="h-1.5 w-1.5 shrink-0 rounded-full transition-transform group-hover:scale-125" 
      :class="priorityDot"
    />
    
    <!-- Title with native CSS truncation -->
    <span 
      class="truncate leading-none" 
      :class="{ 'line-through opacity-60': isCompleted }"
    >
      {{ todo.title }}
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Todo } from '../../types'
import { DRAG_MIME_TYPE } from '../../composables/useCalendarDragDrop'

const props = defineProps<{
  todo: Todo
}>()

const emit = defineEmits<{
  (e: 'click', todo: Todo): void
  (e: 'dragStart', todo: Todo): void
  (e: 'dragEnd', todo: Todo): void
}>()

const isCompleted = computed(() => props.todo.status === 'completed')
const isDragging = ref(false)

function onClick() {
  if (isDragging.value) return
  emit('click', props.todo)
}

function onDragStart(event: DragEvent) {
  isDragging.value = true
  event.dataTransfer?.setData(DRAG_MIME_TYPE, JSON.stringify({
    id: props.todo.id,
    startDate: props.todo.start_date,
  }))
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
  emit('dragStart', props.todo)
}

function onDragEnd() {
  isDragging.value = false
  emit('dragEnd', props.todo)
}

const priorityDot = computed(() => {
  if (isCompleted.value) return 'bg-slate-300 dark:bg-slate-600'
  
  const map: Record<string, string> = {
    low: 'bg-slate-400 dark:bg-slate-500',
    medium: 'bg-blue-500 dark:bg-blue-400',
    high: 'bg-amber-500 dark:bg-amber-400',
    urgent: 'bg-red-500 dark:bg-red-400',
  }
  return map[props.todo.priority] || 'bg-slate-400'
})

const chipStyle = computed(() => {
  if (isCompleted.value) {
    return 'border-transparent bg-slate-100/60 text-slate-400 dark:bg-slate-800/40 dark:text-slate-500'
  }

  // Soft themed backgrounds based on priority for better visual scanning
  const priorityStyles: Record<string, string> = {
    urgent: 
      'border-red-200/60 bg-red-50/80 text-red-900 hover:border-red-300 dark:border-red-900/30 dark:bg-red-950/30 dark:text-red-200 dark:hover:border-red-800/60',
    high: 
      'border-amber-200/60 bg-amber-50/80 text-amber-900 hover:border-amber-300 dark:border-amber-900/30 dark:bg-amber-950/30 dark:text-amber-200 dark:hover:border-amber-800/60',
    medium: 
      'border-blue-200/60 bg-blue-50/80 text-blue-900 hover:border-blue-300 dark:border-blue-900/30 dark:bg-blue-950/30 dark:text-blue-200 dark:hover:border-blue-800/60',
    low: 
      'border-slate-200/60 bg-slate-100/80 text-slate-700 hover:border-slate-300 dark:border-slate-700/60 dark:bg-slate-800/60 dark:text-slate-300 dark:hover:border-slate-600',
  }

  return priorityStyles[props.todo.priority] || priorityStyles.low
})
</script>