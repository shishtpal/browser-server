<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { createApiClient, useExtensionSettings, useTodosView, useUserId } from '../composables/composables'

const emit = defineEmits<{ (event: 'stats', label: string): void }>()

const title = ref('')
const { settings } = useExtensionSettings()
const userId = useUserId(computed(() => settings.value))
const client = computed(() => (settings.value ? createApiClient(settings.value) : null))

const {
  currentDomain,
  domainDisplay,
  screenshotPreview,
  items,
  stats,
  errorMessage,
  init,
  add,
  toggle,
  remove,
  clearAll,
} = useTodosView(client, userId)

defineExpose({ refresh: () => void init(Boolean(settings.value?.autoCapture)), clearAll })

async function initTodos() {
  await init(Boolean(settings.value?.autoCapture))
}

async function submit() {
  if (!title.value.trim()) {
    return
  }
  await add(title.value)
  title.value = ''
}

onMounted(() => {
  void initTodos()
})

watch(stats, (label) => emit('stats', label))
watch(errorMessage, () => emit('stats', stats.value))
watch(currentDomain, () => {
  if (errorMessage.value) {
    emit('stats', '0 todos · 0 done')
  }
})
</script>

<template>
  <section class="max-h-[392px] overflow-y-auto">
    <p class="border-b border-slate-800 px-4 py-2 text-center text-xs text-slate-400">
      {{ domainDisplay }}
    </p>

    <div
      v-if="screenshotPreview"
      class="flex items-center gap-3 border-b border-slate-800 px-4 py-3"
    >
      <img
        :src="screenshotPreview"
        alt="Screenshot preview"
        class="h-14 w-20 rounded border border-slate-700 object-cover"
      />
      <span class="text-xs text-slate-400">Screenshot captured for the next todo</span>
    </div>

    <form class="flex gap-2 border-b border-slate-800 px-4 py-3" @submit.prevent="submit">
      <input
        v-model="title"
        type="text"
        placeholder="Add todo for this page..."
        class="flex-1 rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 outline-none ring-0 placeholder:text-slate-500 focus:border-rose-400"
      />
      <button
        type="submit"
        class="rounded-md bg-rose-500 px-3 py-2 text-sm font-medium text-white hover:bg-rose-400"
      >
        Add
      </button>
    </form>

    <p v-if="errorMessage" class="px-4 py-6 text-center text-xs text-rose-300">
      {{ errorMessage }}
    </p>
    <p
      v-else-if="!currentDomain"
      class="px-4 py-10 text-center text-sm text-slate-500"
    >
      No active domain detected.
    </p>
    <p
      v-else-if="items.length === 0"
      class="px-4 py-10 text-center text-sm text-slate-500"
    >
      No todos for this site.
    </p>
    <ul v-else class="divide-y divide-slate-800">
      <li
        v-for="todo in items"
        :key="todo.id"
        class="flex items-center gap-3 px-4 py-3 hover:bg-slate-800/40"
      >
        <input
          type="checkbox"
          class="h-4 w-4 shrink-0 accent-rose-500"
          :checked="todo.completed"
          @change="toggle(todo.id, ($event.target as HTMLInputElement).checked)"
        />
        <img
          v-if="todo.screenshotUrl"
          :src="todo.screenshotUrl"
          alt="Screenshot"
          class="h-8 w-12 shrink-0 rounded border border-slate-700 object-cover"
        />
        <div class="min-w-0 flex-1">
          <p
            class="truncate text-sm font-medium"
            :class="todo.completed ? 'text-slate-500 line-through' : 'text-slate-100'"
            :title="todo.title"
          >
            {{ todo.title }}
          </p>
          <p class="mt-1 text-[11px] text-slate-500">{{ todo.updatedAtLabel }}</p>
        </div>
        <button
          type="button"
          class="rounded px-2 py-1 text-xs text-slate-400 hover:bg-rose-500 hover:text-white"
          @click="remove(todo.id)"
        >
          Delete
        </button>
      </li>
    </ul>
  </section>
</template>
