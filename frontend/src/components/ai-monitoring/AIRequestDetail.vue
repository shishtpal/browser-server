<template>
  <Modal
    :open="Boolean(request)"
    title="AI request detail"
    :description="request?.id"
    fullscreen
    @close="$emit('close')"
  >
    <div v-if="request" class="h-full overflow-y-auto bg-white p-4 text-sm text-slate-700 sm:p-6 dark:bg-slate-950 dark:text-slate-300">
      <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <button
            type="button"
            class="inline-flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-bold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40 dark:border-white/10 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800"
            :disabled="!canGoPrev"
            @click="$emit('prev')"
          >
            ← Previous
          </button>
          <button
            type="button"
            class="inline-flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-bold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40 dark:border-white/10 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800"
            :disabled="!canGoNext"
            @click="$emit('next')"
          >
            Next →
          </button>
        </div>
        <span class="text-[10px] font-black uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">
          {{ request.source }} request
        </span>
      </div>
      <!-- Overview -->
      <section>
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <div>
            <p class="text-xs font-bold uppercase tracking-widest text-slate-500">Request overview</p>
            <h2 class="mt-1 text-lg font-black text-slate-900 dark:text-white">
              {{ request.provider }} / {{ request.model }}
            </h2>
          </div>

          <div class="flex flex-wrap gap-2">
            <span
              class="rounded-full border px-2.5 py-1 text-xs font-bold"
              :class="statusClass(request.status)"
            >
              {{ request.status }}
            </span>

            <span
              v-if="request.http_status"
              class="rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-bold text-slate-600 dark:border-white/10 dark:bg-white/5 dark:text-slate-300"
            >
              HTTP {{ request.http_status }}
            </span>

            <span
              class="rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-bold text-slate-600 dark:border-white/10 dark:bg-white/5 dark:text-slate-300"
            >
              {{ formatDuration(request.latency_ms) }}
            </span>
          </div>
        </div>

        <dl
          class="grid grid-cols-1 overflow-hidden rounded-2xl border border-slate-200 bg-slate-50/50 sm:grid-cols-2 lg:grid-cols-3 dark:border-white/10 dark:bg-white/[0.02]"
        >
          <div
            v-for="item in details"
            :key="item.label"
            class="min-w-0 border-b border-slate-200 p-4 last:border-b-0 sm:[&:nth-last-child(-n+2)]:border-b-0 lg:[&:nth-last-child(-n+3)]:border-b-0 dark:border-white/10"
          >
            <dt class="text-[10px] font-black uppercase tracking-wider text-slate-500">
              {{ item.label }}
            </dt>
            <dd class="mt-1 break-all font-medium text-slate-800 dark:text-slate-200">
              {{ item.value }}
            </dd>
          </div>
        </dl>
      </section>

      <!-- Error -->
      <section
        v-if="request.error_message"
        class="mt-5 rounded-2xl border border-rose-200 bg-rose-50 p-4 dark:border-rose-900/50 dark:bg-rose-950/30"
      >
        <div class="flex items-start gap-3">
          <div
            class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-rose-100 text-sm font-black text-rose-700 dark:bg-rose-900/40 dark:text-rose-300"
          >
            !
          </div>
          <div class="min-w-0">
            <h3 class="font-black text-rose-800 dark:text-rose-200">
              {{ request.error_code || 'Request error' }}
            </h3>
            <p class="mt-1 whitespace-pre-wrap break-words text-rose-700 dark:text-rose-300">
              {{ request.error_message }}
            </p>
          </div>
        </div>
      </section>

      <!-- Request / response -->
      <section class="mt-6 space-y-5">
        <PayloadBlock
          title="Request payload"
          :payload="request.request_payload"
          :truncated="request.payload_truncated"
        />
        <PayloadBlock
          title="Response payload"
          :payload="request.response_payload"
          :truncated="request.payload_truncated"
        />
      </section>

      <!-- Tools -->
      <section class="mt-8">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div>
            <p class="text-xs font-bold uppercase tracking-widest text-slate-500">Execution trace</p>
            <h3 class="mt-1 text-lg font-black text-slate-900 dark:text-white">
              Tool decisions
            </h3>
          </div>

          <span
            class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-bold text-slate-600 dark:bg-white/10 dark:text-slate-300"
          >
            {{ request.tool_calls?.length || 0 }}
          </span>
        </div>

        <div
          v-if="!request.tool_calls?.length"
          class="mt-3 rounded-2xl border border-dashed border-slate-300 p-6 text-center text-slate-500 dark:border-white/15 dark:text-slate-400"
        >
          No correlated tool activity was captured for this request.
        </div>

        <article
          v-for="tool in request.tool_calls"
          :key="tool.id"
          class="mt-4 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-white/10 dark:bg-white/[0.02]"
        >
          <header
            class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 bg-slate-50/70 p-4 dark:border-white/10 dark:bg-white/[0.03]"
          >
            <div class="min-w-0">
              <p class="text-[10px] font-black uppercase tracking-wider text-slate-500">Tool</p>
              <h4 class="mt-1 break-all font-black text-slate-900 dark:text-white">
                {{ tool.tool_name }}
              </h4>
            </div>

            <div class="flex flex-wrap items-center gap-2 text-xs font-bold">
              <span
                class="rounded-full border px-2 py-1"
                :class="statusClass(tool.status)"
              >
                {{ tool.status }}
              </span>
              <span
                class="rounded-full border border-slate-200 bg-white px-2 py-1 text-slate-600 dark:border-white/10 dark:bg-white/5 dark:text-slate-300"
              >
                {{ tool.decision }}
              </span>
              <span class="text-slate-500">
                {{ formatDuration(tool.duration_ms) }}
              </span>
            </div>
          </header>

          <div class="p-4">
            <div
              v-if="tool.error_message"
              class="mb-4 rounded-xl border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700 dark:border-rose-900/50 dark:bg-rose-950/30 dark:text-rose-300"
            >
              {{ tool.error_message }}
            </div>

            <div class="space-y-4">
              <PayloadBlock
                title="Arguments"
                :payload="tool.arguments"
                :truncated="tool.payload_truncated"
              />
              <PayloadBlock
                title="Result"
                :payload="tool.result"
                :truncated="tool.payload_truncated"
              />
            </div>
          </div>
        </article>
      </section>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import type { AIRequestLog } from '@browser-server/shared-types'
import { computed, onBeforeUnmount, onMounted } from 'vue'
import PayloadBlock from './PayloadBlock.vue'
import Modal from '../ui/Modal.vue'

const props = defineProps<{ request: AIRequestLog | null; canGoPrev?: boolean; canGoNext?: boolean }>()

const emit = defineEmits<{ close: []; prev: []; next: [] }>()

const handleKeydown = (event: KeyboardEvent) => {
  if (!props.request) return
  if (event.key === 'ArrowLeft' && props.canGoPrev) {
    event.preventDefault()
    emit('prev')
  }
  if (event.key === 'ArrowRight' && props.canGoNext) {
    event.preventDefault()
    emit('next')
  }
}

onMounted(() => window.addEventListener('keydown', handleKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', handleKeydown))

const text = (value: unknown) => {
  return value === undefined || value === null || value === '' ? '—' : String(value)
}

const formatDuration = (ms?: number) => {
  if (ms === undefined || ms === null) return '—'
  return ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms}ms`
}

const statusClass = (status?: string) => {
  const value = status?.toLowerCase() ?? ''

  if (['success', 'completed', 'complete', 'ok'].includes(value)) {
    return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-950/30 dark:text-emerald-300'
  }

  if (['failed', 'error', 'cancelled'].includes(value)) {
    return 'border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-900/50 dark:bg-rose-950/30 dark:text-rose-300'
  }

  if (['pending', 'queued', 'running', 'processing'].includes(value)) {
    return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-300'
  }

  return 'border-slate-200 bg-slate-50 text-slate-600 dark:border-white/10 dark:bg-white/5 dark:text-slate-300'
}

const details = computed(() => {
  if (!props.request) return []

  return [
    { label: 'Created', value: new Date(props.request.created_at).toLocaleString() },
    { label: 'Source', value: text(props.request.source) },
    { label: 'Endpoint', value: text(props.request.endpoint) },
    { label: 'Conversation ID', value: text(props.request.conversation_id) },
    { label: 'Message ID', value: text(props.request.message_id) },
    { label: 'Task ID', value: text(props.request.task_id) },
    { label: 'Iteration', value: text(props.request.iteration) },
    {
      label: 'Tokens (prompt / completion / total)',
      value: `${text(props.request.prompt_tokens)} / ${text(props.request.completion_tokens)} / ${text(props.request.total_tokens)}`,
    },
  ]
})

</script>
