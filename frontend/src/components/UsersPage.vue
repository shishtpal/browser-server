<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useModal } from '@browser-server/shared-modal';
import {
  ArrowRight,
  Check,
  Database,
  LoaderCircle,
  Mail,
  Plus,
  Search,
  ShieldCheck,
  Trash2,
  UserRound,
  Users,
} from '@lucide/vue';
import { useUser } from '../composables/useUser';
import { useUsers } from '../composables/useUsers';
import Button from './ui/Button.vue';
import EmptyState from './ui/EmptyState.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import InputField from './ui/InputField.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import PageHeader from './ui/PageHeader.vue';
import StatCard from './ui/StatCard.vue';

const {
  usersList,
  isLoading,
  isCreating,
  deletingId,
  error,
  successMsg,
  newUsername,
  newEmail,
  loadUsers,
  addUser,
  removeUser,
} = useUsers();
const { currentUserId, setUser, clearUser, fetchUsers: refreshSharedUsers } = useUser();
const { confirmDelete } = useModal();

const searchQuery = ref('');
const createPanel = ref<HTMLElement | null>(null);

const usersWithEmail = computed(() => usersList.value.filter((user) => Boolean(user.email)).length);
const activeUser = computed(() => usersList.value.find((user) => user.id === currentUserId.value));
const filteredUsers = computed(() => {
  const query = searchQuery.value.trim().toLowerCase();
  if (!query) return usersList.value;
  return usersList.value.filter(
    (user) =>
      user.username.toLowerCase().includes(query) ||
      user.email?.toLowerCase().includes(query) ||
      String(user.id) === query.replace(/^#/, ''),
  );
});

const avatarPalettes = [
  'from-amber-400 to-orange-500 shadow-orange-500/20',
  'from-violet-500 to-indigo-500 shadow-indigo-500/20',
  'from-cyan-500 to-blue-500 shadow-cyan-500/20',
  'from-emerald-500 to-teal-500 shadow-emerald-500/20',
  'from-rose-500 to-pink-500 shadow-rose-500/20',
];

function initials(name: string): string {
  return (
    name
      .trim()
      .split(/\s+/)
      .slice(0, 2)
      .map((part) => part[0]?.toUpperCase())
      .join('') || 'U'
  );
}

function avatarClass(id: number): string {
  return avatarPalettes[Math.abs(id) % avatarPalettes.length] ?? avatarPalettes[0] ?? '';
}

function openCreatePanel() {
  createPanel.value?.scrollIntoView({ behavior: 'smooth', block: 'center' });
}

async function handleCreate() {
  const created = await addUser();
  if (!created) return;
  await refreshSharedUsers();
  if (!currentUserId.value) setUser(created.id);
}

function activateUser(id: number) {
  setUser(id);
}

async function confirmRemove(id: number) {
  const user = usersList.value.find((item) => item.id === id);
  const confirmed = await confirmDelete(
    `Delete “${user?.username ?? 'this workspace'}”?`,
    'This removes the workspace identity. Records using this numeric user ID may remain in individual data stores, so back up anything important first.',
  );
  if (!confirmed) return;
  const removed = await removeUser(id);
  if (!removed) return;
  if (currentUserId.value === id) clearUser();
  await refreshSharedUsers();
}

onMounted(loadUsers);
</script>

<template>
  <div class="mx-auto w-full max-w-[1500px] px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Workspace identities" title="Users" color="amber">
      <template #stats>
        <StatCard :value="usersList.length" label="Workspaces" variant="dark" color="amber" />
        <StatCard :value="usersWithEmail" label="With email" variant="primary" color="amber" />
        <StatCard
          v-if="activeUser"
          :value="activeUser.username"
          label="Active"
          variant="secondary"
          color="emerald"
        />
      </template>
      <template #controls>
        <Button variant="gradient-amber" size="sm" @click="openCreatePanel">
          <span class="flex items-center gap-1.5">
            <Plus class="h-4 w-4" :stroke-width="2.4" aria-hidden="true" />
            New workspace
          </span>
        </Button>
      </template>
    </PageHeader>

    <section
      class="relative mb-5 overflow-hidden rounded-2xl border border-amber-200/70 bg-gradient-to-br from-amber-50 via-white to-orange-50 p-4 sm:p-5 dark:border-amber-500/15 dark:from-amber-500/10 dark:via-slate-900 dark:to-orange-500/5"
    >
      <div class="relative flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex max-w-3xl items-start gap-3">
          <span
            class="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-amber-500 text-white shadow-lg shadow-amber-500/20"
          >
            <ShieldCheck class="h-5 w-5" aria-hidden="true" />
          </span>
          <div>
            <h2 class="text-sm font-extrabold text-slate-900 dark:text-white">
              Workspaces, not login accounts
            </h2>
            <p class="mt-1 text-xs leading-relaxed text-slate-600 dark:text-slate-300">
              A user keeps todos, bookmarks, history, wallet entries, and quiz activity separated by
              numeric ID. Authentication is still controlled by the server API token.
            </p>
          </div>
        </div>
        <div
          class="flex shrink-0 items-center gap-2 text-[11px] font-bold text-slate-500 dark:text-slate-400"
        >
          <Database class="h-4 w-4 text-amber-500" aria-hidden="true" />
          Local data boundary
        </div>
      </div>
    </section>

    <div
      v-if="successMsg"
      class="mb-4 flex items-center gap-2 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-xs font-bold text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-300"
      role="status"
    >
      <Check class="h-4 w-4" aria-hidden="true" />
      {{ successMsg }}
    </div>

    <ErrorBanner v-if="error" :message="error" :on-retry="loadUsers" />

    <div class="grid grid-cols-1 gap-5 lg:grid-cols-[340px_minmax(0,1fr)] lg:items-start">
      <aside
        ref="createPanel"
        class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm lg:sticky lg:top-16 dark:border-white/10 dark:bg-slate-900"
      >
        <div class="border-b border-slate-100 p-4 dark:border-white/10">
          <div class="flex items-center gap-3">
            <span
              class="grid h-9 w-9 place-items-center rounded-xl bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-300"
            >
              <UserRound class="h-4 w-4" aria-hidden="true" />
            </span>
            <div>
              <h2 class="text-sm font-extrabold text-slate-900 dark:text-white">
                Create a workspace
              </h2>
              <p class="text-[11px] text-slate-500 dark:text-slate-400">
                Add a distinct local data owner.
              </p>
            </div>
          </div>
        </div>

        <form class="space-y-4 p-4" @submit.prevent="handleCreate">
          <div>
            <label
              class="mb-1.5 block text-xs font-bold text-slate-600 dark:text-slate-300"
              for="user-name"
            >
              Workspace name
            </label>
            <InputField
              id="user-name"
              v-model="newUsername"
              type="text"
              placeholder="e.g. Personal"
              required
              color="amber"
            />
            <p class="mt-1.5 text-[10px] leading-relaxed text-slate-400">
              Use a short recognizable label; this appears in workspace selectors.
            </p>
          </div>
          <div>
            <label
              class="mb-1.5 block text-xs font-bold text-slate-600 dark:text-slate-300"
              for="user-email"
            >
              Email <span class="font-medium text-slate-400">(optional)</span>
            </label>
            <InputField
              id="user-email"
              v-model="newEmail"
              type="email"
              placeholder="you@example.com"
              color="amber"
            />
          </div>
          <Button
            type="submit"
            variant="gradient-amber"
            size="md"
            class="w-full"
            :disabled="!newUsername.trim()"
            :loading="isCreating"
            loading-text="Creating..."
          >
            <span class="flex items-center justify-center gap-2">
              Create workspace
              <ArrowRight class="h-4 w-4" aria-hidden="true" />
            </span>
          </Button>
        </form>
      </aside>

      <section class="min-w-0">
        <div
          class="mb-3 flex flex-col gap-3 rounded-2xl border border-slate-200 bg-white p-3 shadow-sm sm:flex-row sm:items-center sm:justify-between dark:border-white/10 dark:bg-slate-900"
        >
          <div>
            <h2 class="text-sm font-extrabold text-slate-900 dark:text-white">
              Workspace directory
            </h2>
            <p class="text-[11px] text-slate-500 dark:text-slate-400">
              {{ filteredUsers.length }} of {{ usersList.length }} shown
            </p>
          </div>
          <label class="relative block w-full sm:max-w-xs">
            <span class="sr-only">Search workspaces</span>
            <Search
              class="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-slate-400"
              aria-hidden="true"
            />
            <input
              v-model="searchQuery"
              type="search"
              placeholder="Search name, email, or ID"
              class="w-full rounded-xl border border-slate-200 bg-slate-50 py-2 pr-3 pl-9 text-xs font-semibold text-slate-700 transition outline-none focus:border-amber-400 focus:ring-4 focus:ring-amber-100 dark:border-white/10 dark:bg-slate-950 dark:text-slate-200 dark:focus:ring-amber-900/30"
            />
          </label>
        </div>

        <LoadingSpinner v-if="isLoading" message="Loading workspaces..." color="amber" />

        <EmptyState
          v-else-if="usersList.length === 0"
          title="No workspaces yet"
          description="Create the first local workspace using the form."
          icon="users"
          color="amber"
        />

        <EmptyState
          v-else-if="filteredUsers.length === 0"
          title="No matching workspaces"
          :description="`Nothing matches “${searchQuery}”. Try a different search.`"
          icon="search"
          color="amber"
        />

        <div v-else class="grid grid-cols-1 gap-3 xl:grid-cols-2">
          <article
            v-for="user in filteredUsers"
            :key="user.id"
            class="group rounded-2xl border bg-white p-4 shadow-sm transition hover:-translate-y-0.5 hover:shadow-lg dark:bg-slate-900"
            :class="
              currentUserId === user.id
                ? 'border-emerald-300 ring-2 ring-emerald-100 dark:border-emerald-500/40 dark:ring-emerald-500/10'
                : 'border-slate-200 hover:border-amber-300 dark:border-white/10 dark:hover:border-amber-500/30'
            "
          >
            <div class="flex items-start gap-3">
              <span
                class="grid h-11 w-11 shrink-0 place-items-center rounded-xl bg-gradient-to-br text-sm font-black text-white shadow-lg"
                :class="avatarClass(user.id)"
              >
                {{ initials(user.username) }}
              </span>
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <h3 class="truncate text-sm font-extrabold text-slate-900 dark:text-white">
                    {{ user.username }}
                  </h3>
                  <span
                    v-if="currentUserId === user.id"
                    class="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2 py-0.5 text-[9px] font-bold tracking-wide text-emerald-700 uppercase dark:bg-emerald-500/10 dark:text-emerald-300"
                  >
                    <Check class="h-2.5 w-2.5" aria-hidden="true" /> Active
                  </span>
                </div>
                <a
                  v-if="user.email"
                  :href="`mailto:${user.email}`"
                  class="mt-1 flex min-w-0 items-center gap-1.5 text-xs text-slate-500 transition hover:text-amber-600 dark:text-slate-400 dark:hover:text-amber-300"
                >
                  <Mail class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                  <span class="truncate">{{ user.email }}</span>
                </a>
                <p v-else class="mt-1 text-xs text-slate-400">No email attached</p>
              </div>
              <span class="font-mono text-[10px] font-bold text-slate-400">#{{ user.id }}</span>
            </div>

            <div
              class="mt-4 flex items-center justify-between gap-2 border-t border-slate-100 pt-3 dark:border-white/10"
            >
              <button
                v-if="currentUserId !== user.id"
                type="button"
                class="inline-flex items-center gap-1.5 rounded-lg bg-slate-900 px-3 py-2 text-[11px] font-bold text-white transition hover:bg-amber-500 dark:bg-white dark:text-slate-900 dark:hover:bg-amber-400"
                @click="activateUser(user.id)"
              >
                Use workspace <ArrowRight class="h-3.5 w-3.5" aria-hidden="true" />
              </button>
              <span
                v-else
                class="inline-flex items-center gap-1.5 text-[11px] font-bold text-emerald-600 dark:text-emerald-300"
              >
                <Users class="h-3.5 w-3.5" aria-hidden="true" /> Current workspace
              </span>
              <button
                type="button"
                class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-2 text-[11px] font-bold text-rose-600 transition hover:bg-rose-50 disabled:opacity-40 dark:text-rose-300 dark:hover:bg-rose-500/10"
                :disabled="deletingId !== null"
                @click="confirmRemove(user.id)"
              >
                <LoaderCircle
                  v-if="deletingId === user.id"
                  class="h-3.5 w-3.5 animate-spin"
                  aria-hidden="true"
                />
                <Trash2 v-else class="h-3.5 w-3.5" aria-hidden="true" />
                Delete
              </button>
            </div>
          </article>
        </div>
      </section>
    </div>
  </div>
</template>
