<template>
  <section class="rounded-2xl border border-slate-200 dark:border-white/10">
    <header
      class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 bg-slate-50/70 px-4 py-3 dark:border-white/10 dark:bg-white/[0.03]"
    >
      <div>
        <h4 class="font-black text-slate-900 dark:text-white">{{ title }}</h4>
        <p
          v-if="formattedPayload"
          :class="[
            'mt-0.5 text-xs',
            formattedPayload.isJson
              ? 'text-emerald-600 dark:text-emerald-400'
              : 'text-slate-500 dark:text-slate-400',
          ]"
        >
          {{ formattedPayload.isJson ? 'Formatted JSON' : 'Raw text' }}
        </p>
      </div>

      <button
        v-if="formattedPayload"
        type="button"
        class="inline-flex items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-2.5 py-1.5 text-xs font-bold text-slate-700 transition hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:cursor-not-allowed dark:border-white/15 dark:bg-white/5 dark:text-slate-200 dark:hover:bg-white/10"
        @click="copy"
      >
        {{ copied ? 'Copied!' : copyFailed ? 'Copy failed' : 'Copy JSON' }}
      </button>
    </header>

    <pre
      v-if="formattedPayload"
      class="max-h-[28rem] overflow-auto bg-slate-950 p-4 font-mono text-xs leading-5 text-slate-100 scrollbar-thin scrollbar-track-slate-900 scrollbar-thumb-slate-700"
    >
      <code class="whitespace-pre">{{ formattedPayload.value }}</code>
    </pre>
    <p
      v-else
      class="m-4 rounded-xl border border-dashed border-slate-300 bg-slate-50 p-4 text-xs text-slate-500 dark:border-white/10 dark:bg-white/[0.03] dark:text-slate-400"
    >
      Payload omitted or not captured.
    </p>

    <p
      v-if="truncated"
      class="border-t border-amber-200 bg-amber-50 px-4 py-2 text-xs font-semibold text-amber-700 dark:border-amber-900/40 dark:bg-amber-950/20 dark:text-amber-300"
    >
      This payload was truncated by the server.
    </p>
  </section>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { copyToClipboard } from '../../utils/copyToClipboard'

interface FormattedPayload {
  value: string
  isJson: boolean
}

const props = withDefaults(
  defineProps<{
    title: string
    payload?: string | null
    truncated?: boolean
  }>(),
  {
    payload: null,
    truncated: false,
  },
)

const copied = ref(false)
const copyFailed = ref(false)

const formatPayload = (payload: unknown): FormattedPayload | null => {
  if (payload === undefined || payload === null || payload === '') {
    return null
  }

  if (typeof payload === 'string') {
    try {
      return {
        value: "\n" + JSON.stringify(JSON.parse(payload), null, 2),
        isJson: true,
      }
    } catch {
      return {
        value: payload,
        isJson: false,
      }
    }
  }

  try {
    return {
      value: JSON.stringify(payload, null, 2),
      isJson: true,
    }
  } catch {
    return {
      value: String(payload),
      isJson: false,
    }
  }
}

const formattedPayload = computed(() => formatPayload(props.payload))

const copy = async () => {
  if (!formattedPayload.value) return

  copied.value = false
  copyFailed.value = false

  try {
    await copyToClipboard(formattedPayload.value.value)
    copied.value = true
    window.setTimeout(() => {
      copied.value = false
    }, 1800)
  } catch {
    copyFailed.value = true
  }
}
</script>
