<script setup lang="ts">
import { AlertCircle, ServerCog, ShieldCheck } from '@lucide/vue';
import PageHeader from './ui/PageHeader.vue';
import AdminTokenCard from './project-settings/AdminTokenCard.vue';
import ConfigEditor from './project-settings/ConfigEditor.vue';
import ConfigFileList from './project-settings/ConfigFileList.vue';
import { useProjectSettings } from './project-settings/composables/useProjectSettings';

const {
  access,
  status,
  files,
  selected,
  draft,
  schema,
  dirty,
  needsRestart,
  canRestart,
  loadingFiles,
  loadingContent,
  saving,
  reloading,
  restarting,
  error,
  toast,
  loadFiles,
  selectFile,
  save,
  reloadSelected,
  restart,
} = useProjectSettings();

function refreshFiles() {
  void loadFiles(selected.value?.name);
}
</script>

<template>
  <div class="mx-auto max-w-full px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Administration" title="Project Settings" color="violet">
      <template #stats>
        <span
          v-if="status"
          class="inline-flex items-center gap-1.5 rounded-full bg-slate-100 px-2.5 py-1 text-[10px] font-bold text-slate-600 dark:bg-white/10 dark:text-slate-300"
        >
          <ServerCog class="h-3 w-3" aria-hidden="true" />
          {{ status.managed ? 'Managed service' : 'Manual process' }}
        </span>
        <span
          v-if="access === 'ready'"
          class="inline-flex items-center gap-1.5 rounded-full bg-emerald-50 px-2.5 py-1 text-[10px] font-bold text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300"
        >
          <ShieldCheck class="h-3 w-3" aria-hidden="true" />
          Admin authenticated
        </span>
      </template>
    </PageHeader>

    <AdminTokenCard :access="access" />

    <div
      v-if="access !== 'ready'"
      class="mt-4 rounded-2xl border p-5"
      :class="
        access === 'disabled'
          ? 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-100'
          : 'border-slate-200 bg-white text-slate-700 dark:border-white/10 dark:bg-slate-900 dark:text-slate-200'
      "
    >
      <div class="flex items-start gap-3">
        <AlertCircle class="mt-0.5 h-5 w-5 shrink-0" aria-hidden="true" />
        <div>
          <h2 class="text-sm font-extrabold">
            {{
              access === 'checking'
                ? 'Checking administrator access…'
                : access === 'disabled'
                  ? 'Administrator API is disabled'
                  : access === 'unauthorized'
                    ? 'Administrator token rejected'
                    : 'Administrator token required'
            }}
          </h2>
          <p class="mt-1 text-xs leading-relaxed opacity-80">
            {{
              error ||
              'Save the separate administrator token above. The ordinary API token cannot open project configuration.'
            }}
          </p>
          <p v-if="access === 'disabled'" class="mt-3 text-xs">
            Run <code class="rounded bg-black/10 px-1.5 py-0.5">server token admin-generate</code>,
            restart the service, and then paste the generated token above.
          </p>
        </div>
      </div>
    </div>

    <template v-else>
      <div
        v-if="error"
        class="mt-4 flex items-start gap-2 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-xs text-rose-800 dark:border-rose-500/20 dark:bg-rose-500/10 dark:text-rose-200"
      >
        <AlertCircle class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
        <pre class="font-sans whitespace-pre-wrap">{{ error }}</pre>
      </div>

      <div class="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
        <ConfigFileList
          :files="files"
          :selected="selected"
          :loading="loadingFiles"
          @select="selectFile"
          @refresh="refreshFiles"
        />
        <ConfigEditor
          v-model="draft"
          :file="selected"
          :dirty="dirty"
          :loading="loadingContent"
          :saving="saving"
          :reloading="reloading"
          :restarting="restarting"
          :managed="status?.managed ?? false"
          :schema="schema"
          :restart-required="needsRestart"
          :can-restart="canRestart"
          @save="save"
          @reload="reloadSelected"
          @restart="restart"
        />
      </div>
    </template>

    <transition name="toast">
      <div
        v-if="toast"
        class="fixed right-4 bottom-4 z-50 max-w-md rounded-xl px-4 py-3 text-xs font-bold text-white shadow-2xl"
        :class="{
          'bg-emerald-600': toast.tone === 'success',
          'bg-amber-500': toast.tone === 'warning',
          'bg-rose-600': toast.tone === 'error',
        }"
        role="status"
      >
        {{ toast.message }}
      </div>
    </transition>
  </div>
</template>
