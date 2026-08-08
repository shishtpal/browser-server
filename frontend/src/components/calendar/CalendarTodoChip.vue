<template>
  <button
    type="button"
    class="group flex w-full items-center gap-1.5 rounded-md border px-2 py-1 text-left text-xs font-medium transition-all duration-150 active:scale-[0.98]"
    :class="[chipClass, isDragging ? 'cursor-grabbing opacity-50' : 'cursor-grab']"
    draggable="true"
    @click.stop="onClick"
    @dragstart.stop="onDragStart"
    @dragend.stop="onDragEnd"
  >
    <span
      class="h-1.5 w-1.5 shrink-0 rounded-full transition-transform group-hover:scale-125"
      :class="dotClass"
      aria-hidden="true"
    />
    <span class="truncate leading-none" :class="{ 'line-through opacity-60': isCompleted }">
      {{ todo.title }}
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { Todo } from '../../types';
import { DRAG_MIME_TYPE } from './composables/useCalendarDragDrop';
import { todoChipClass, todoDotClass } from '../todos/todoFormat';

const props = defineProps<{ todo: Todo }>();

const emit = defineEmits<{
  (e: 'click', todo: Todo): void;
  (e: 'dragStart', todo: Todo): void;
  (e: 'dragEnd', todo: Todo): void;
}>();

const isCompleted = computed(() => props.todo.status === 'completed');
const isDragging = ref(false);

// Priority dot + chip styling come from the shared todo metadata.
const dotClass = computed(() => todoDotClass(props.todo));
const chipClass = computed(() => todoChipClass(props.todo));

function onClick() {
  if (isDragging.value) return;
  emit('click', props.todo);
}

function onDragStart(event: DragEvent) {
  isDragging.value = true;
  event.dataTransfer?.setData(
    DRAG_MIME_TYPE,
    JSON.stringify({
      id: props.todo.id,
      startDate: props.todo.start_date,
    }),
  );
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move';
  }
  emit('dragStart', props.todo);
}

function onDragEnd() {
  isDragging.value = false;
  emit('dragEnd', props.todo);
}
</script>
