<template>
  <form
    class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm transition-colors dark:border-white/10 dark:bg-slate-800/90"
    aria-label="Add a bookmark"
    @submit.prevent="onSubmit"
  >
    <div class="grid gap-2 sm:grid-cols-2 lg:flex lg:items-center lg:gap-2">
      <div class="relative">
        <BookmarkIcon
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="title"
          type="text"
          placeholder="Title"
          required
          class="w-full rounded-lg border border-gray-300 bg-gray-50 py-2 pr-3 pl-9 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30"
        />
      </div>

      <div class="relative">
        <Link2
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="url"
          type="url"
          placeholder="https://example.com"
          required
          class="w-full rounded-lg border border-gray-300 bg-gray-50 py-2 pr-3 pl-9 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30"
        />
      </div>

      <div class="relative lg:flex-1">
        <AlignLeft
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="description"
          type="text"
          placeholder="Description (optional)"
          class="w-full rounded-lg border border-gray-300 bg-gray-50 py-2 pr-3 pl-9 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30"
        />
      </div>

      <div class="relative lg:w-40">
        <Hash
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="tagsStr"
          type="text"
          placeholder="Tags: comma, separated"
          class="w-full rounded-lg border border-gray-300 bg-gray-50 py-2 pr-3 pl-9 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30"
        />
      </div>

      <Button type="submit" variant="gradient-cyan" size="sm" class="sm:col-span-2 lg:col-span-1">
        <span class="inline-flex items-center gap-1.5">
          <Plus class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          Add
        </span>
      </Button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { AlignLeft, Bookmark as BookmarkIcon, Hash, Link2, Plus } from '@lucide/vue';
import Button from '../ui/Button.vue';
import { parseTags } from './bookmarkFormat';
import type { BookmarkCreateInput } from './composables/useBookmarks';

const emit = defineEmits<{ submit: [input: BookmarkCreateInput] }>();

const title = ref('');
const url = ref('');
const description = ref('');
const tagsStr = ref('');

const onSubmit = () => {
  emit('submit', {
    title: title.value,
    url: url.value,
    description: description.value,
    tags: parseTags(tagsStr.value),
  });
  title.value = '';
  url.value = '';
  description.value = '';
  tagsStr.value = '';
};
</script>
