<template>
  <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
    <TodoGridCard
      v-for="todo in todos"
      :key="todo.id"
      :todo="todo"
      @toggle="$emit('toggle', $event)"
      @toggle-pin="$emit('toggle-pin', $event)"
      @archive="$emit('archive', $event)"
      @restore="$emit('restore', $event)"
      @view-screenshot="$emit('view-screenshot', $event)"
      @start-edit="$emit('start-edit', $event)"
      @delete="$emit('delete', $event)"
      @toggle-subtask="$emit('toggle-subtask', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import type { Todo } from '../../types';
import TodoGridCard from './TodoGridCard.vue';

interface Props {
  todos: Todo[];
}

defineProps<Props>();

defineEmits<{
  toggle: [todo: Todo];
  'toggle-pin': [todo: Todo];
  archive: [todo: Todo];
  restore: [todo: Todo];
  'view-screenshot': [todo: Todo];
  'start-edit': [todo: Todo];
  delete: [id: number];
  'toggle-subtask': [todo: Todo];
}>();
</script>
