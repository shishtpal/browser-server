<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { ChevronDown, KeyRound, Save, Trash2 } from '@lucide/vue';
import { clearAdminToken, getAdminToken, setAdminToken } from '../../lib/auth';
import type { AdminAccess } from './composables/useProjectSettings';

const props = defineProps<{ access: AdminAccess }>();
const draft = ref('');
const saved = ref(false);
const hasSavedToken = ref(false);
const expanded = ref(true);

const stateLabel = computed(() => {
  switch (props.access) {
    case 'ready':
      return 'Connected';
    case 'checking':
      return 'Checking';
    case 'disabled':
      return 'Server disabled';
    case 'unauthorized':
      return 'Invalid token';
    default:
      return 'Token required';
  }
});

function refresh() {
  const token = getAdminToken() ?? '';
  draft.value = token;
  hasSavedToken.value = Boolean(token);
}

function save() {
  setAdminToken(draft.value);
  refresh();
  saved.value = true;
  if (hasSavedToken.value) expanded.value = false;
  window.setTimeout(() => (saved.value = false), 1500);
}

function clear() {
  clearAdminToken();
  refresh();
  expanded.value = true;
}

watch(
  () => props.access,
  (access) => {
    if (access === 'unauthorized' || access === 'missing-token') expanded.value = true;
  },
);

onMounted(() => {
  refresh();
  expanded.value =
    !hasSavedToken.value || props.access === 'unauthorized' || props.access === 'missing-token';
});
</script>

<template>
  <section
    class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-white/10 dark:bg-slate-900"
  >
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="flex items-start gap-3">
        <span
          class="grid h-9 w-9 shrink-0 place-items-center rounded-xl bg-violet-50 text-violet-600 dark:bg-violet-500/10 dark:text-violet-300"
        >
          <KeyRound class="h-4 w-4" :stroke-width="2.2" aria-hidden="true" />
        </span>
        <div>
          <h2 class="text-sm font-extrabold text-slate-900 dark:text-white">Administrator token</h2>
          <p class="mt-1 max-w-3xl text-xs leading-relaxed text-slate-500 dark:text-slate-400">
            {{
              !expanded && hasSavedToken
                ? 'An administrator token is saved. Expand this section to rotate or clear it.'
                : 'This separate credential can read and rewrite server configuration. Do not reuse the operator token stored by browser extensions.'
            }}
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <span
          class="rounded-full px-2.5 py-1 text-[10px] font-bold tracking-wide uppercase"
          :class="
            access === 'ready'
              ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
              : access === 'unauthorized'
                ? 'bg-rose-50 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
                : 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
          "
        >
          {{ stateLabel }}
        </span>
        <button
          type="button"
          class="grid h-8 w-8 place-items-center rounded-lg border border-slate-200 text-slate-500 transition hover:bg-slate-100 dark:border-white/10 dark:text-slate-300 dark:hover:bg-white/10"
          :title="
            expanded
              ? 'Collapse administrator token settings'
              : 'Expand administrator token settings'
          "
          :aria-label="
            expanded
              ? 'Collapse administrator token settings'
              : 'Expand administrator token settings'
          "
          :aria-expanded="expanded"
          aria-controls="administrator-token-fields"
          @click="expanded = !expanded"
        >
          <ChevronDown
            class="h-4 w-4 transition-transform duration-200"
            :class="{ 'rotate-180': expanded }"
            aria-hidden="true"
          />
        </button>
      </div>
    </div>

    <div
      v-if="expanded"
      id="administrator-token-fields"
      class="mt-4 grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto]"
    >
      <div>
        <label for="admin-token" class="mb-1 block text-[11px] font-bold text-slate-500">
          Admin API token
        </label>
        <input
          id="admin-token"
          v-model="draft"
          type="password"
          autocomplete="off"
          placeholder="Paste the token from server token admin-generate"
          class="w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2.5 font-mono text-xs text-slate-900 transition outline-none placeholder:font-sans placeholder:text-slate-400 focus:border-violet-400 focus:ring-4 focus:ring-violet-100 dark:border-white/10 dark:bg-slate-950 dark:text-white dark:focus:ring-violet-900/30"
          @keyup.enter="save"
        />
      </div>
      <div class="flex items-end gap-2">
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded-xl bg-violet-600 px-4 py-2.5 text-xs font-bold text-white transition hover:bg-violet-500"
          @click="save"
        >
          <Save class="h-3.5 w-3.5" aria-hidden="true" />
          {{ saved ? 'Saved' : 'Save token' }}
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded-xl border border-slate-200 px-3 py-2.5 text-xs font-bold text-slate-600 transition hover:bg-slate-100 dark:border-white/10 dark:text-slate-300 dark:hover:bg-white/10"
          @click="clear"
        >
          <Trash2 class="h-3.5 w-3.5" aria-hidden="true" />
          Clear
        </button>
      </div>
    </div>

    <p v-if="expanded" class="mt-3 text-[11px] text-slate-500 dark:text-slate-400">
      Generate with
      <code
        class="rounded bg-slate-100 px-1.5 py-0.5 text-violet-700 dark:bg-white/10 dark:text-violet-300"
        >server token admin-generate</code
      >, then restart the server so it loads <code>.bs-token-admin</code>.
    </p>
  </section>
</template>
