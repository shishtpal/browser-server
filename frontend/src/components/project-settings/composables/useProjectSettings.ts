import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import {
  AdminAPIError,
  getAdminConfigFile,
  getAdminConfigSchema,
  getAdminStatus,
  isServerOnline,
  listAdminConfigFiles,
  putAdminConfigFile,
  reloadAdminConfigFile,
  restartServer,
  type AdminConfigFile,
  type AdminConfigSchema,
  type AdminStatus,
} from '../../../lib/api';
import { hasAdminToken } from '../../../lib/auth';

export type AdminAccess =
  | 'missing-token'
  | 'checking'
  | 'ready'
  | 'unauthorized'
  | 'disabled'
  | 'error';
export type ToastTone = 'success' | 'warning' | 'error';

export function useProjectSettings() {
  const access = ref<AdminAccess>('checking');
  const status = ref<AdminStatus | null>(null);
  const files = ref<AdminConfigFile[]>([]);
  const selected = ref<AdminConfigFile | null>(null);
  const draft = ref('');
  const original = ref('');
  const schema = ref<AdminConfigSchema | null>(null);
  const loadingFiles = ref(false);
  const loadingContent = ref(false);
  const saving = ref(false);
  const reloading = ref(false);
  const restarting = ref(false);
  const error = ref('');
  const needsRestart = ref(false);
  const toast = ref<{ message: string; tone: ToastTone } | null>(null);
  let toastTimer: number | undefined;
  let selectionRequest = 0;

  const dirty = computed(() => draft.value !== original.value);
  const canRestart = computed(() => Boolean(status.value?.managed && needsRestart.value));

  function showToast(message: string, tone: ToastTone = 'success') {
    toast.value = { message, tone };
    if (toastTimer) window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(() => (toast.value = null), 5000);
  }

  function describeError(caught: unknown): string {
    if (caught instanceof AdminAPIError) return caught.message;
    return caught instanceof Error ? caught.message : 'Unexpected administrator API error.';
  }

  function classifyAccess(caught: unknown) {
    if (caught instanceof AdminAPIError && caught.status === 401) {
      access.value = 'unauthorized';
      error.value = 'The saved administrator token is invalid.';
    } else if (
      caught instanceof AdminAPIError &&
      caught.status === 403 &&
      caught.code === 'admin_disabled'
    ) {
      access.value = 'disabled';
      error.value =
        'The server has no administrator token loaded. Generate one and restart the server.';
    } else {
      access.value = 'error';
      error.value = describeError(caught);
    }
  }

  async function connect() {
    selectionRequest += 1;
    error.value = '';
    status.value = null;
    files.value = [];
    selected.value = null;
    schema.value = null;
    loadingContent.value = false;
    needsRestart.value = false;
    if (!hasAdminToken()) {
      access.value = 'missing-token';
      return;
    }
    access.value = 'checking';
    try {
      status.value = await getAdminStatus();
      access.value = 'ready';
      await loadFiles();
    } catch (caught) {
      classifyAccess(caught);
    }
  }

  async function loadFiles(preferredName?: string) {
    loadingFiles.value = true;
    error.value = '';
    try {
      files.value = await listAdminConfigFiles();
      const name = preferredName ?? selected.value?.name;
      const next = files.value.find((file) => file.name === name) ?? files.value[0] ?? null;
      if (next) await selectFile(next);
    } catch (caught) {
      classifyAccess(caught);
    } finally {
      loadingFiles.value = false;
    }
  }

  async function selectFile(file: AdminConfigFile) {
    const requestID = ++selectionRequest;
    if (dirty.value && selected.value?.name !== file.name) {
      showToast('Unsaved editor changes were discarded.', 'warning');
    }
    selected.value = file;
    needsRestart.value = false;
    error.value = '';
    schema.value = null;
    if (!file.exists) {
      const starter = '{\n  \n}\n';
      draft.value = starter;
      original.value = starter;
      return;
    }
    loadingContent.value = true;
    try {
      const content = await getAdminConfigFile(file.name);
      if (requestID !== selectionRequest) return;
      draft.value = content.content;
      original.value = content.content;
      schema.value = null;
      try {
        const schemaResponse = await getAdminConfigSchema(file.name);
        if (requestID === selectionRequest) schema.value = schemaResponse;
      } catch {
        // Schema loading is best-effort: fall back to code mode if it fails.
        if (requestID === selectionRequest) schema.value = null;
      }
    } catch (caught) {
      if (requestID === selectionRequest) error.value = describeError(caught);
    } finally {
      if (requestID === selectionRequest) loadingContent.value = false;
    }
  }

  async function save() {
    if (!selected.value || saving.value) return;
    saving.value = true;
    error.value = '';
    try {
      const result = await putAdminConfigFile(selected.value.name, draft.value);
      const restartRequired =
        result.reload === 'restart_required' || Boolean(result.restart_required);
      needsRestart.value = restartRequired;
      showToast(
        restartRequired
          ? 'Saved — server restart required.'
          : result.reload === 'hot_reloaded'
            ? 'Saved and hot-reloaded.'
            : 'Saved.',
        restartRequired ? 'warning' : 'success',
      );
      if (result.warning) showToast(result.warning, 'warning');
      await loadFiles(selected.value.name);
      if (restartRequired) needsRestart.value = true;
    } catch (caught) {
      error.value = describeError(caught);
      showToast(error.value, 'error');
    } finally {
      saving.value = false;
    }
  }

  async function reloadSelected() {
    if (!selected.value || selected.value.class !== 'leaf' || reloading.value) return;
    reloading.value = true;
    error.value = '';
    try {
      const result = await reloadAdminConfigFile(selected.value.name);
      needsRestart.value = Boolean(result.restart_required);
      showToast(
        result.warning ?? 'Configuration hot-reloaded.',
        result.warning ? 'warning' : 'success',
      );
    } catch (caught) {
      error.value = describeError(caught);
      showToast(error.value, 'error');
    } finally {
      reloading.value = false;
    }
  }

  async function restart() {
    if (!status.value?.managed || restarting.value) return;
    restarting.value = true;
    error.value = '';
    const previousUptime = status.value.uptime_seconds;
    try {
      await restartServer();
      showToast('Restart requested. Waiting for the managed service…', 'warning');
      await new Promise((resolve) => window.setTimeout(resolve, 1200));

      let backOnline = false;
      for (let attempt = 0; attempt < 45; attempt += 1) {
        if (await isServerOnline()) {
          try {
            const current = await getAdminStatus();
            // A lower uptime proves a restart. After several seconds, accept
            // the healthy authenticated process as a fallback for very young
            // servers whose before/after uptimes are nearly equal.
            if (current.uptime_seconds < previousUptime || attempt >= 3) {
              status.value = current;
              backOnline = true;
              break;
            }
          } catch {
            // The process may be accepting health checks before all routes are ready.
          }
        }
        await new Promise((resolve) => window.setTimeout(resolve, 2000));
      }
      if (!backOnline) throw new Error('The server did not return before the restart timeout.');
      needsRestart.value = false;
      await loadFiles(selected.value?.name);
      showToast('Server restarted successfully.');
    } catch (caught) {
      error.value = describeError(caught);
      showToast(error.value, 'error');
    } finally {
      restarting.value = false;
    }
  }

  function handleTokenChange() {
    void connect();
  }

  onMounted(() => {
    window.addEventListener('api-admin-token-changed', handleTokenChange);
    void connect();
  });
  onBeforeUnmount(() => {
    window.removeEventListener('api-admin-token-changed', handleTokenChange);
    if (toastTimer) window.clearTimeout(toastTimer);
  });

  return {
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
    connect,
    loadFiles,
    selectFile,
    save,
    reloadSelected,
    restart,
  };
}
