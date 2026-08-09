<template>
  <Modal
    :open="!!bookmark"
    title="Edit bookmark"
    description="Update the saved link details."
    @close="$emit('close')"
  >
    <form v-if="bookmark" :key="bookmark.id" class="flex flex-col gap-4" @submit.prevent="onSave">
      <FormField label="Title" required>
        <div class="relative">
          <BookmarkIcon
            class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
            aria-hidden="true"
          />
          <input
            v-model="form.title"
            type="text"
            required
            placeholder="Title"
            class="w-full rounded-lg border border-gray-300 bg-gray-50 py-2 pr-3 pl-9 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30"
          />
        </div>
      </FormField>

      <FormField label="URL" required>
        <div class="relative">
          <Link2
            class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
            aria-hidden="true"
          />
          <input
            v-model="form.url"
            type="url"
            required
            placeholder="https://example.com"
            class="w-full rounded-lg border border-gray-300 bg-gray-50 py-2 pr-3 pl-9 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30"
          />
        </div>
      </FormField>

      <FormField label="Description">
        <input
          v-model="form.description"
          type="text"
          placeholder="Description"
          class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30"
        />
      </FormField>

      <FormField label="Tags" help-text="Comma-separated.">
        <input
          v-model="form.tagsStr"
          type="text"
          placeholder="Tags: comma, separated"
          class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30"
        />
      </FormField>

      <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <Button type="button" variant="secondary" size="sm" @click="$emit('close')">
          Cancel
        </Button>
        <Button
          type="submit"
          variant="gradient-cyan"
          size="sm"
          :loading="saving"
          loading-text="Saving…"
        >
          <span class="inline-flex items-center gap-1.5">
            <Check class="h-3.5 w-3.5" :stroke-width="3" aria-hidden="true" />
            Save changes
          </span>
        </Button>
      </div>
    </form>
  </Modal>
</template>

<script setup lang="ts">
import type { BookmarkResponse } from '../../types';
import { ref, watch } from 'vue';
import { Bookmark as BookmarkIcon, Check, Link2 } from '@lucide/vue';
import Button from '../ui/Button.vue';
import FormField from '../ui/FormField.vue';
import Modal from '../ui/Modal.vue';

const props = defineProps<{
  bookmark: BookmarkResponse | null;
}>();

const emit = defineEmits<{
  close: [];
  save: [data: { title: string; url: string; description: string; tagsStr: string }];
}>();

const form = ref({ title: '', url: '', description: '', tagsStr: '' });
const saving = ref(false);

watch(
  () => props.bookmark,
  (b) => {
    if (b) {
      form.value = {
        title: b.title,
        url: b.url,
        description: b.description,
        tagsStr: b.tags.join(', '),
      };
    }
  },
  { immediate: true },
);

const onSave = async () => {
  if (!form.value.title.trim() || !form.value.url.trim()) return;
  saving.value = true;
  try {
    emit('save', { ...form.value });
  } finally {
    saving.value = false;
  }
};
</script>
