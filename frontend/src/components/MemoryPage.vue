<template>
  <div class="mx-auto max-w-full px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Memory graph" title="Memory" color="violet">
      <template #stats>
        <StatCard :value="fragments" label="Fragments" variant="dark" color="violet" />
        <StatCard :value="edgeCount" label="Edges" variant="primary" color="violet" />
      </template>
      <template #actions>
        <Button
          variant="secondary"
          size="sm"
          :disabled="maintaining"
          :loading="maintaining"
          loading-text="Maintaining..."
          @click="runMaintain"
        >
          <span class="flex items-center gap-1.5">
            <RefreshCw class="h-4 w-4" />
            Run maintenance
          </span>
        </Button>
        <Button variant="primary" size="sm" @click="createNew">
          <span class="flex items-center gap-1.5">
            <Plus class="h-4 w-4" />
            New fragment
          </span>
        </Button>
        <Button variant="ghost" size="sm" @click="reload">
          <span class="flex items-center gap-1.5">
            <RefreshCw class="h-4 w-4" />
            Refresh
          </span>
        </Button>
      </template>
    </PageHeader>

    <div class="mt-4 flex flex-wrap items-center gap-3">
      <InputField v-model="searchQuery" placeholder="Filter fragments…" color="violet" flex />
      <SelectField v-model="kindFilter" class="w-44">
        <option value="">All kinds</option>
        <option v-for="k in kindOptions" :key="k" :value="k">{{ k }}</option>
      </SelectField>
      <span v-if="selectedId" class="text-xs font-semibold text-slate-500 dark:text-slate-400">
        Selected: <code class="text-violet-600 dark:text-violet-400">{{ selectedId }}</code>
      </span>
      <span
        v-else-if="isNewFragment"
        class="text-xs font-semibold text-slate-500 dark:text-slate-400"
      >
        Editing: <code class="text-violet-600 dark:text-violet-400">new fragment</code> (id assigned
        on save)
      </span>
    </div>

    <ErrorBanner v-if="error" :message="error" :on-retry="reload" />
    <LoadingSpinner v-if="isLoading" message="Loading memory graph…" color="violet" />

    <div v-else-if="nodes.length === 0" class="mt-6">
      <EmptyState
        title="No memory yet"
        description="The graph is empty. Ask the agent to remember something, or create a fragment."
        icon="search"
        color="violet"
      />
    </div>

    <div v-else class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-5">
      <!-- Graph pane -->
      <div class="lg:col-span-3">
        <div
          class="relative overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-900"
        >
          <div
            class="flex items-center justify-between border-b border-slate-100 px-3 py-2 dark:border-slate-800"
          >
            <span class="text-xs font-bold tracking-wide text-slate-400 uppercase">
              Graph view ({{ nodes.length }} nodes)
            </span>
            <div class="flex items-center gap-1">
              <button
                class="rounded-md p-1 text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800"
                type="button"
                @click="zoomOut"
              >
                −
              </button>
              <button
                class="rounded-md p-1 text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800"
                type="button"
                @click="zoomIn"
              >
                +
              </button>
              <button
                class="rounded-md px-1.5 py-1 text-[10px] font-bold text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800"
                type="button"
                title="Fit all graph nodes in view"
                @click="fitGraph"
              >
                Fit
              </button>
            </div>
          </div>
          <div
            ref="canvasRef"
            class="memory-canvas relative h-[480px] cursor-grab overflow-hidden bg-slate-50 active:cursor-grabbing dark:bg-slate-950"
            @mousedown="onPanStart"
            @mousemove="onPanMove"
            @mouseup="onPanEnd"
            @mouseleave="onPanEnd"
          >
            <div
              class="absolute inset-0 origin-top-left"
              :style="{ transform: `translate(${pan.x}px, ${pan.y}px) scale(${scale})` }"
            >
              <svg :width="graphSize.width" :height="graphSize.height" class="select-none">
                <!-- edges -->
                <line
                  v-for="(e, i) in edges"
                  :key="'e' + i"
                  :x1="pos(e.from).x"
                  :y1="pos(e.from).y"
                  :x2="pos(e.to).x"
                  :y2="pos(e.to).y"
                  :stroke="edgeColor(e.rel)"
                  stroke-width="1.5"
                  stroke-opacity="0.5"
                />
                <!-- nodes -->
                <g
                  v-for="n in visibleNodes"
                  :key="n.id"
                  :transform="`translate(${pos(n.id).x}, ${pos(n.id).y})`"
                  class="cursor-pointer"
                  @mousedown.stop="startDrag(n.id, $event)"
                  @click.stop="select(n.id)"
                >
                  <circle
                    :r="radius(n.kind)"
                    :fill="nodeFill(n.kind)"
                    :stroke="selectedId === n.id ? '#fff' : nodeStroke(n.kind)"
                    :stroke-width="selectedId === n.id ? 3 : 1.5"
                    class="drop-shadow-sm"
                  />
                  <text
                    :x="radius(n.kind) + 6"
                    y="4"
                    class="fill-slate-700 dark:fill-slate-200"
                    font-size="11"
                    font-weight="600"
                  >
                    {{ label(n) }}
                  </text>
                </g>
              </svg>
            </div>
          </div>
          <div
            class="flex flex-wrap gap-1.5 border-t border-slate-100 px-3 py-2 dark:border-slate-800"
          >
            <span
              v-for="k in presentKinds"
              :key="k"
              class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-bold tracking-wide text-white uppercase"
              :style="{ background: nodeFill(k) }"
            >
              {{ k }}
            </span>
          </div>
        </div>
      </div>

      <!-- Detail / edit pane -->
      <div class="lg:col-span-2">
        <div
          v-if="!editingFragment"
          class="rounded-2xl border border-dashed border-slate-300 p-6 text-center dark:border-slate-700"
        >
          <p class="text-sm font-semibold text-slate-500 dark:text-slate-400">
            Select a fragment to view or edit it.
          </p>
        </div>
        <div
          v-else
          class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900"
        >
          <div class="mb-3 flex items-start justify-between gap-2">
            <div>
              <h3 class="text-sm font-extrabold text-slate-800 dark:text-slate-100">
                {{ isNewFragment ? 'new fragment' : editingFragment.id }}
              </h3>
              <p class="text-xs text-slate-400">
                {{
                  isNewFragment ? 'The id is derived from the title on save.' : 'Editing fragment'
                }}
              </p>
            </div>
            <div class="flex items-center gap-1">
              <Button variant="danger" size="sm" :disabled="saving" @click="deleteFragment">
                Delete
              </Button>
            </div>
          </div>

          <div class="space-y-3">
            <div>
              <label class="mb-1 block text-xs font-bold text-slate-500">Title</label>
              <InputField v-model="form.title" color="violet" />
            </div>
            <div class="grid grid-cols-2 gap-2">
              <div>
                <label class="mb-1 block text-xs font-bold text-slate-500">Kind</label>
                <SelectField v-model="form.kind">
                  <option v-for="k in kindOptions" :key="k" :value="k">{{ k }}</option>
                </SelectField>
              </div>
              <div>
                <label class="mb-1 block text-xs font-bold text-slate-500">Status</label>
                <SelectField v-model="form.status">
                  <option value="active">active</option>
                  <option value="archived">archived</option>
                  <option value="superseded">superseded</option>
                </SelectField>
              </div>
            </div>
            <div>
              <label class="mb-1 block text-xs font-bold text-slate-500">Parent (child_of)</label>
              <InputField v-model="form.parent" color="violet" placeholder="mem_inbox" />
            </div>
            <div>
              <label class="mb-1 block text-xs font-bold text-slate-500"
                >Tags (comma separated)</label
              >
              <InputField
                v-model="form.tagsText"
                color="violet"
                placeholder="memory, architecture"
              />
            </div>
            <div>
              <label class="mb-1 block text-xs font-bold text-slate-500">Summary</label>
              <TextAreaField
                v-model="form.summary"
                :rows="2"
                placeholder="Short summary (≤280 chars)"
              />
            </div>
            <div>
              <label class="mb-1 block text-xs font-bold text-slate-500">Body (markdown)</label>
              <TextAreaField v-model="form.body" :rows="8" placeholder="Full markdown body" />
            </div>

            <div
              v-if="!isNewFragment && editingFragment.links.length"
              class="rounded-xl border border-slate-100 p-3 dark:border-slate-800"
            >
              <p class="mb-2 text-xs font-bold tracking-wide text-slate-400 uppercase">Links</p>
              <div class="space-y-1">
                <div
                  v-for="(l, i) in editingFragment.links"
                  :key="i"
                  class="flex items-center justify-between gap-2 rounded-md bg-slate-50 px-2 py-1 dark:bg-slate-800"
                >
                  <span class="text-xs">
                    <code class="text-violet-600 dark:text-violet-400">{{ l.rel }}</code>
                    → <code>{{ l.to }}</code>
                  </span>
                  <button
                    v-if="l.rel !== 'child_of'"
                    class="text-rose-500 hover:text-rose-700"
                    type="button"
                    @click="removeLink(l)"
                  >
                    <X class="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            </div>

            <div
              v-if="!isNewFragment"
              class="rounded-xl border border-slate-100 p-3 dark:border-slate-800"
            >
              <p class="mb-2 text-xs font-bold tracking-wide text-slate-400 uppercase">Add link</p>
              <div class="flex items-center gap-1.5">
                <SelectField v-model="newLinkRel" class="w-28">
                  <option value="relates">relates</option>
                  <option value="depends_on">depends_on</option>
                  <option value="supersedes">supersedes</option>
                  <option value="about">about</option>
                  <option value="contradicts">contradicts</option>
                  <option value="source">source</option>
                </SelectField>
                <InputField
                  v-model="newLinkTo"
                  class="min-w-0 flex-1"
                  placeholder="target id (e.g. mem_x)"
                  color="violet"
                  flex
                />
                <Button variant="secondary" size="sm" @click="addLink">Add</Button>
              </div>
            </div>

            <ErrorBanner v-if="formError" :message="formError" @retry="clearFormError" />
            <div class="flex items-center gap-2 pt-1">
              <Button
                variant="primary"
                size="sm"
                :loading="saving"
                :disabled="saving"
                @click="save"
              >
                Save changes
              </Button>
              <span v-if="savedMsg" class="text-xs font-semibold text-emerald-600">{{
                savedMsg
              }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue';
import { Plus, RefreshCw, X } from '@lucide/vue';
import PageHeader from './ui/PageHeader.vue';
import StatCard from './ui/StatCard.vue';
import Button from './ui/Button.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import EmptyState from './ui/EmptyState.vue';
import InputField from './ui/InputField.vue';
import TextAreaField from './ui/TextAreaField.vue';
import SelectField from './ui/SelectField.vue';
import {
  getAIMemoryGraph,
  getAIMemoryStats,
  getAIMemoryFragment,
  writeAIMemory,
  maintainAIMemory,
} from '../lib/api/memory';
import type {
  AIMemoryGraphNode,
  AIMemoryGraphEdge,
  AIMemoryFragment,
  AIMemoryLink,
} from '@browser-server/shared-types';

const W = 1200;
const H = 900;
const COL_W = 220;
const ROW_H = 64;
const ORPHAN_ROW_H = 48;

const kindOptions = [
  'persona',
  'project',
  'component',
  'decision',
  'task',
  'preference',
  'fact',
  'person',
  'event',
  'glossary',
  'index',
  'note',
  'stub',
];

const nodes = ref<AIMemoryGraphNode[]>([]);
const edges = ref<AIMemoryGraphEdge[]>([]);
const fragments = ref(0);
const isLoading = ref(false);
const error = ref<string | null>(null);
const maintaining = ref(false);
const searchQuery = ref('');
const kindFilter = ref('');

// positions keyed by node id
const positions = ref<Record<string, { x: number; y: number }>>({});

const selectedId = ref<string | null>(null);
const editingFragment = ref<AIMemoryFragment | null>(null);
const form = ref({
  title: '',
  kind: 'note',
  status: 'active',
  parent: 'mem_inbox',
  tagsText: '',
  summary: '',
  body: '',
});
const newLinkRel = ref('relates');
const newLinkTo = ref('');
const saving = ref(false);
const savedMsg = ref('');
const isNewFragment = ref(false);
let savedMsgTimer: ReturnType<typeof setTimeout> | null = null;
const formError = ref<string | null>(null);

const pan = ref({ x: 0, y: 0 });
const scale = ref(1);
const canvasRef = ref<HTMLElement | null>(null);
const panning = ref(false);
const panStart = ref({ x: 0, y: 0, px: 0, py: 0 });

const edgeCount = computed(() => edges.value.length);

// The graph can grow beyond the original design-time SVG dimensions. Keep the
// SVG large enough for every laid-out node (and its label) so overflow never
// silently hides a fragment before pan/zoom gets a chance to reveal it.
const graphSize = computed(() => {
  const positionsList = Object.values(positions.value);
  const maxX = Math.max(0, ...positionsList.map((p) => p.x));
  const maxY = Math.max(0, ...positionsList.map((p) => p.y));
  return { width: Math.max(W, maxX + 180), height: Math.max(H, maxY + ROW_H) };
});

const presentKinds = computed(() => {
  const set = new Set(nodes.value.map((n) => n.kind));
  return kindOptions.filter((k) => set.has(k));
});

const visibleNodes = computed(() => {
  const q = searchQuery.value.trim().toLowerCase();
  const k = kindFilter.value;
  if (!q && !k) return nodes.value;
  return nodes.value.filter((n) => {
    if (k && n.kind !== k) return false;
    if (q && !(n.id + ' ' + n.title + ' ' + n.summary).toLowerCase().includes(q)) return false;
    return true;
  });
});

const pos = (id: string) => {
  const p = positions.value[id];
  return p || { x: 0, y: 0 };
};

const label = (n: AIMemoryGraphNode) => {
  const t = n.title || n.id;
  return t.length > 22 ? t.slice(0, 20) + '…' : t;
};

const radius = (kind: string) => (kind === 'persona' || kind === 'index' ? 16 : 12);

function nodeFill(kind: string): string {
  const map: Record<string, string> = {
    persona: '#8b5cf6',
    project: '#6366f1',
    component: '#0ea5e9',
    decision: '#f59e0b',
    task: '#10b981',
    preference: '#ec4899',
    fact: '#14b8a6',
    person: '#f97316',
    event: '#f43f5e',
    glossary: '#94a3b8',
    index: '#64748b',
    note: '#22d3ee',
    stub: '#cbd5e1',
  };
  return map[kind] || '#22d3ee';
}
function nodeStroke(kind: string): string {
  return kind === 'persona' ? '#6d28d9' : kind === 'index' ? '#475569' : '#ffffff';
}
function edgeColor(rel: string): string {
  const map: Record<string, string> = {
    child_of: '#94a3b8',
    relates: '#818cf8',
    depends_on: '#f59e0b',
    supersedes: '#10b981',
    about: '#0ea5e9',
    contradicts: '#f43f5e',
    source: '#a78bfa',
  };
  return map[rel] || '#94a3b8';
}

// Layered layout: BFS from mem_root assigning columns by depth, rows by index.
// The backend now returns the full child_of closure regardless of depth, so
// orphans should be rare; anything unreachable is still shown in its own
// column so the viewer never silently drops fragments.
function layout() {
  const byId = new Map(nodes.value.map((n) => [n.id, n]));
  const childMap = new Map<string, string[]>();
  for (const e of edges.value) {
    if (e.rel !== 'child_of') continue;
    if (!childMap.has(e.from)) childMap.set(e.from, []);
    childMap.get(e.from)!.push(e.to);
  }
  const depth = new Map<string, number>();
  const queue: string[] = ['mem_root'];
  if (byId.has('mem_root')) {
    depth.set('mem_root', 0);
    while (queue.length) {
      const cur = queue.shift()!;
      const d = depth.get(cur)!;
      for (const child of childMap.get(cur) || []) {
        if (!byId.has(child) || depth.has(child)) continue;
        depth.set(child, d + 1);
        queue.push(child);
      }
    }
  }
  // Orphans are every node we could not reach from mem_root.
  const orphanCol: string[] = [];
  for (const n of nodes.value) {
    if (!depth.has(n.id)) orphanCol.push(n.id);
  }
  const cols: string[][] = [];
  for (const n of nodes.value) {
    const d = depth.get(n.id);
    if (d === undefined) continue;
    while (cols.length <= d) cols.push([]);
    cols[d]!.push(n.id);
  }
  if (orphanCol.length) cols.push(orphanCol);

  const p: Record<string, { x: number; y: number }> = {};
  cols.forEach((col, c) => {
    const x = c * COL_W + COL_W / 2;
    const rowH = c === cols.length - 1 && orphanCol.length ? ORPHAN_ROW_H : ROW_H;
    const offsetY = ROW_H;
    col.forEach((id, idx) => {
      p[id] = { x, y: offsetY + idx * rowH };
    });
  });
  positions.value = p;
}

async function reload() {
  isLoading.value = true;
  error.value = null;
  try {
    const [g, st] = await Promise.all([getAIMemoryGraph(), getAIMemoryStats()]);
    nodes.value = g.nodes;
    edges.value = g.edges;
    fragments.value = st.fragments;
    layout();
    await nextTick();
    fitGraph();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load memory graph';
  } finally {
    isLoading.value = false;
  }
}

async function select(id: string) {
  selectedId.value = id;
  isNewFragment.value = false;
  editingFragment.value = null;
  formError.value = null;
  savedMsg.value = '';
  try {
    const f = await getAIMemoryFragment(id);
    editingFragment.value = f;
    form.value = {
      title: f.title,
      kind: f.kind || 'note',
      status: f.status || 'active',
      parent: f.parent || 'mem_inbox',
      tagsText: (f.tags || []).join(', '),
      summary: f.summary || '',
      body: f.body || '',
    };
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to load fragment';
  }
}

function createNew() {
  editingFragment.value = {
    id: '',
    kind: 'note',
    title: '',
    summary: '',
    body: '',
    tags: [],
    status: 'active',
    pinned: false,
    parent: 'mem_inbox',
    links: [],
  };
  form.value = {
    title: '',
    kind: 'note',
    status: 'active',
    parent: 'mem_inbox',
    tagsText: '',
    summary: '',
    body: '',
  };
  selectedId.value = null;
  isNewFragment.value = true;
}

function flashSaved(msg: string) {
  savedMsg.value = msg;
  if (savedMsgTimer) clearTimeout(savedMsgTimer);
  savedMsgTimer = setTimeout(() => {
    savedMsg.value = '';
  }, 3000);
}

async function save() {
  saving.value = true;
  formError.value = null;
  savedMsg.value = '';
  const op = {
    op: 'upsert' as const,
    title: form.value.title || 'Untitled fragment',
    kind: form.value.kind,
    summary: form.value.summary,
    body: form.value.body,
    tags: form.value.tagsText
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean),
    parent: form.value.parent || 'mem_inbox',
    status: form.value.status,
  };
  try {
    const res = await writeAIMemory([op]);
    const createdId = res.results?.[0]?.id || selectedId.value;
    await reload();
    if (createdId) await select(createdId);
    flashSaved(res.results?.[0]?.created ? 'Created' : 'Saved');
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Save failed';
  } finally {
    saving.value = false;
  }
}

async function deleteFragment() {
  if (!editingFragment.value) return;
  const id = editingFragment.value.id;
  if (!window.confirm(`Delete fragment ${id}? This moves it to the archive (keeps history).`))
    return;
  saving.value = true;
  formError.value = null;
  try {
    await writeAIMemory([{ op: 'delete', id, cascade: true }]);
    editingFragment.value = null;
    selectedId.value = null;
    isNewFragment.value = false;
    await reload();
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Delete failed';
  } finally {
    saving.value = false;
  }
}

async function addLink() {
  const to = newLinkTo.value.trim();
  if (!to || !editingFragment.value) return;
  saving.value = true;
  formError.value = null;
  try {
    await writeAIMemory([
      { op: 'link', from: editingFragment.value.id, rel: newLinkRel.value, to },
    ]);
    newLinkTo.value = '';
    await reload();
    await select(editingFragment.value.id);
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Link failed';
  } finally {
    saving.value = false;
  }
}

async function removeLink(l: AIMemoryLink) {
  if (!editingFragment.value) return;
  saving.value = true;
  formError.value = null;
  try {
    await writeAIMemory([{ op: 'unlink', from: editingFragment.value.id, rel: l.rel, to: l.to }]);
    await reload();
    await select(editingFragment.value.id);
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Unlink failed';
  } finally {
    saving.value = false;
  }
}

async function runMaintain() {
  maintaining.value = true;
  try {
    await maintainAIMemory();
    await reload();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Maintenance failed';
  } finally {
    maintaining.value = false;
  }
}

function clearFormError() {
  formError.value = null;
}

// pan / zoom
function zoomIn() {
  scale.value = Math.min(2.5, scale.value + 0.25);
}
function zoomOut() {
  scale.value = Math.max(0.1, scale.value - 0.25);
}
function fitGraph() {
  const canvas = canvasRef.value;
  if (!canvas) return;
  const padding = 24;
  const fit = Math.min(
    1,
    (canvas.clientWidth - padding * 2) / graphSize.value.width,
    (canvas.clientHeight - padding * 2) / graphSize.value.height,
  );
  scale.value = Math.max(0.1, fit);
  pan.value = {
    x: Math.max(padding, (canvas.clientWidth - graphSize.value.width * scale.value) / 2),
    y: Math.max(padding, (canvas.clientHeight - graphSize.value.height * scale.value) / 2),
  };
}
function onPanStart(e: MouseEvent) {
  panning.value = true;
  panStart.value = { x: e.clientX, y: e.clientY, px: pan.value.x, py: pan.value.y };
}
function onPanMove(e: MouseEvent) {
  if (!panning.value) return;
  pan.value = {
    x: panStart.value.px + (e.clientX - panStart.value.x),
    y: panStart.value.py + (e.clientY - panStart.value.y),
  };
}
function onPanEnd() {
  panning.value = false;
}

// node drag (offset within the scaled group)
function startDrag(id: string, e: MouseEvent) {
  e.preventDefault();
  e.stopPropagation();
  const start = { x: e.clientX, y: e.clientY };
  const orig = { ...pos(id) };
  const move = (ev: MouseEvent) => {
    const dx = (ev.clientX - start.x) / scale.value;
    const dy = (ev.clientY - start.y) / scale.value;
    positions.value = { ...positions.value, [id]: { x: orig.x + dx, y: orig.y + dy } };
  };
  const up = () => {
    window.removeEventListener('mousemove', move);
    window.removeEventListener('mouseup', up);
  };
  window.addEventListener('mousemove', move);
  window.addEventListener('mouseup', up);
}

onMounted(reload);
</script>
