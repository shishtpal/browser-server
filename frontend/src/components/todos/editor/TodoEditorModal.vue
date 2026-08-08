<template>
  <Modal
    :open="open"
    :title="editingTodo ? 'Edit Todo' : 'New Todo'"
    :description="
      editingTodo
        ? 'Update the title, dates, priority, and tags.'
        : 'Create a new todo; it appears in the Todos workspace and on the calendar.'
    "
    @close="$emit('close')"
  >
    <div class="max-h-[70vh] overflow-y-auto overscroll-contain pr-1">
      <TodoEditorForm
        :key="`${editingTodo?.id ?? 'new'}-${open}`"
        :editing-todo="editingTodo"
        :initial-due-date="initialDueDate"
        :user-id="userId"
        @submit="onSubmit"
        @update="onUpdate"
      />
    </div>

    <!-- Footer actions -->
    <div
      class="mt-4 flex items-center justify-between gap-2 border-t border-gray-200 pt-4 dark:border-slate-700"
    >
      <Button v-if="editingTodo" type="button" variant="danger" size="sm" @click="$emit('delete')">
        <span class="inline-flex items-center gap-1.5">
          <Trash2 class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          Delete
        </span>
      </Button>
      <div v-else></div>

      <div class="flex items-center gap-2">
        <Button type="button" variant="secondary" size="sm" @click="$emit('close')">Cancel</Button>
        <Button
          type="submit"
          form="todo-editor"
          variant="gradient-violet"
          size="sm"
          :loading="saving"
        >
          <span class="inline-flex items-center gap-1.5">
            <Check class="h-3.5 w-3.5" :stroke-width="3" aria-hidden="true" />
            {{ editingTodo ? 'Save' : 'Create' }}
          </span>
        </Button>
      </div>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import type { CreateTodoInput, Todo } from '../../../types';
import { ref } from 'vue';
import { Check, Trash2 } from '@lucide/vue';
import Modal from '../../ui/Modal.vue';
import Button from '../../ui/Button.vue';
import TodoEditorForm from './TodoEditorForm.vue';

withDefaults(
  defineProps<{
    open: boolean;
    editingTodo?: Todo | null;
    initialDueDate?: string;
    userId: number;
  }>(),
  { editingTodo: null, initialDueDate: '' },
);

const emit = defineEmits<{
  close: [];
  submit: [data: CreateTodoInput];
  update: [id: number, data: Partial<Todo>];
  delete: [];
}>();

const saving = ref(false);

async function onSubmit(data: CreateTodoInput) {
  saving.value = true;
  try {
    emit('submit', data);
    emit('close');
  } finally {
    saving.value = false;
  }
}

async function onUpdate(id: number, data: Partial<Todo>) {
  saving.value = true;
  try {
    emit('update', id, data);
    emit('close');
  } finally {
    saving.value = false;
  }
}
</script>
