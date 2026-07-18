<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { faviconUrl } from '@browser-server/shared-utils'
import type { GraphLayout, PositionedNode } from './buildGraph'

const props = defineProps<{
  layout: GraphLayout
  selectedId: string | null
}>()

const emit = defineEmits<{
  select: [node: PositionedNode]
  background: []
  dropBookmark: [bookmarkId: number, folderPath: string]
}>()

const viewport = ref<SVGSVGElement | null>(null)
const pan = ref({ x: 0, y: 0 })
const zoom = ref(1)
const minZoom = 0.25
const maxZoom = 3

type PanDrag = { mode: 'pan'; startX: number; startY: number; originX: number; originY: number }
type NodeDrag = {
  mode: 'node'
  node: PositionedNode
  pointerId: number
  current: { x: number; y: number }
  hoverFolderId: string | null
}
type DragState = PanDrag | NodeDrag | null

const drag = ref<DragState>(null)

const transform = computed(() => `translate(${pan.value.x} ${pan.value.y}) scale(${zoom.value})`)

const hoverFolderId = computed(() => (drag.value?.mode === 'node' ? drag.value.hoverFolderId : null))

function clientToGraph(clientX: number, clientY: number): { x: number; y: number } {
  const rect = viewport.value?.getBoundingClientRect()
  if (!rect) return { x: 0, y: 0 }
  return {
    x: (clientX - rect.left - pan.value.x) / zoom.value,
    y: (clientY - rect.top - pan.value.y) / zoom.value,
  }
}

function onBackgroundPointerDown(event: PointerEvent) {
  if (event.button !== 0) return
  emit('background')
  drag.value = {
    mode: 'pan',
    startX: event.clientX,
    startY: event.clientY,
    originX: pan.value.x,
    originY: pan.value.y,
  }
  ;(event.currentTarget as Element).setPointerCapture(event.pointerId)
}

function onNodePointerDown(event: PointerEvent, node: PositionedNode) {
  if (event.button !== 0) return
  if (node.node.type === 'root') {
    emit('select', node)
    return
  }
  event.stopPropagation()
  emit('select', node)
  if (node.node.type !== 'bookmark') return
  drag.value = {
    mode: 'node',
    node,
    pointerId: event.pointerId,
    current: { x: node.x, y: node.y },
    hoverFolderId: null,
  }
  ;(event.currentTarget as Element).setPointerCapture(event.pointerId)
}

function onPointerMove(event: PointerEvent) {
  const state = drag.value
  if (!state) return
  if (state.mode === 'pan') {
    pan.value = {
      x: state.originX + (event.clientX - state.startX),
      y: state.originY + (event.clientY - state.startY),
    }
    return
  }
  const g = clientToGraph(event.clientX, event.clientY)
  state.current = g
  let found: string | null = null
  for (const n of props.layout.nodes) {
    if (n.node.type !== 'folder') continue
    if (n.node.id === state.node.node.id) continue
    const dx = g.x - n.x
    const dy = g.y - n.y
    if (Math.hypot(dx, dy) < 36) {
      found = n.node.id
      break
    }
  }
  state.hoverFolderId = found
}

function endDrag() {
  const state = drag.value
  if (!state) return
  if (state.mode === 'node' && state.hoverFolderId) {
    const folder = props.layout.nodes.find((n) => n.node.id === state.hoverFolderId)
    const bookmarkId = state.node.node.bookmark?.id
    if (folder && bookmarkId != null) {
      emit('dropBookmark', bookmarkId, folder.node.path)
    }
  }
  drag.value = null
}

function onWheel(event: WheelEvent) {
  event.preventDefault()
  const rect = viewport.value?.getBoundingClientRect()
  if (!rect) return
  const factor = Math.exp(-event.deltaY * 0.0015)
  const nextZoom = Math.min(maxZoom, Math.max(minZoom, zoom.value * factor))
  const cx = event.clientX - rect.left
  const cy = event.clientY - rect.top
  pan.value = {
    x: cx - (cx - pan.value.x) * (nextZoom / zoom.value),
    y: cy - (cy - pan.value.y) * (nextZoom / zoom.value),
  }
  zoom.value = nextZoom
}

function zoomBy(factor: number) {
  const rect = viewport.value?.getBoundingClientRect()
  const cx = rect ? rect.width / 2 : 0
  const cy = rect ? rect.height / 2 : 0
  const nextZoom = Math.min(maxZoom, Math.max(minZoom, zoom.value * factor))
  pan.value = {
    x: cx - (cx - pan.value.x) * (nextZoom / zoom.value),
    y: cy - (cy - pan.value.y) * (nextZoom / zoom.value),
  }
  zoom.value = nextZoom
}

function fit() {
  const rect = viewport.value?.getBoundingClientRect()
  if (!rect) return
  const { minX, minY, maxX, maxY } = props.layout.bounds
  const w = maxX - minX
  const h = maxY - minY
  if (!isFinite(w) || !isFinite(h) || w <= 0 || h <= 0) {
    pan.value = { x: rect.width / 2, y: rect.height / 2 }
    zoom.value = 1
    return
  }
  const pad = 80
  const scale = Math.min((rect.width - pad * 2) / w, (rect.height - pad * 2) / h)
  zoom.value = Math.min(maxZoom, Math.max(minZoom, scale))
  const cx = (minX + maxX) / 2
  const cy = (minY + maxY) / 2
  pan.value = {
    x: rect.width / 2 - cx * zoom.value,
    y: rect.height / 2 - cy * zoom.value,
  }
}

defineExpose({ fit, zoomIn: () => zoomBy(1.2), zoomOut: () => zoomBy(1 / 1.2), reset: fit })

onBeforeUnmount(() => {
  drag.value = null
})
</script>

<template>
  <svg
    ref="viewport"
    class="h-full w-full touch-none select-none bg-slate-950"
    :class="drag?.mode === 'pan' ? 'cursor-grabbing' : 'cursor-grab'"
    @pointerdown="onBackgroundPointerDown"
    @pointermove="onPointerMove"
    @pointerup="endDrag"
    @pointercancel="endDrag"
    @wheel.passive.prevent="onWheel"
  >
    <g :transform="transform">
      <path
        v-for="(edge, i) in layout.edges"
        :key="`edge-${i}`"
        :d="`M ${edge.from.x} ${edge.from.y} Q ${edge.from.x} ${(edge.from.y + edge.to.y) / 2} ${edge.to.x} ${edge.to.y}`"
        fill="none"
        stroke="rgb(51 65 85 / 0.7)"
        stroke-width="1.5"
      />

      <g
        v-for="n in layout.nodes"
        :key="n.node.id"
        :transform="`translate(${n.x} ${n.y})`"
        :class="[
          'cursor-pointer',
          n.node.type === 'bookmark' && drag?.mode === 'node' && drag.node.node.id === n.node.id ? 'opacity-40' : '',
        ]"
        @pointerdown.stop="onNodePointerDown($event, n)"
      >
        <template v-if="n.node.type === 'root'">
          <circle r="26" :class="selectedId === n.node.id ? 'fill-rose-500' : 'fill-slate-800'" stroke="rgb(244 63 94)" stroke-width="2" />
          <text text-anchor="middle" dominant-baseline="central" class="pointer-events-none fill-white text-[11px] font-semibold">
            {{ n.node.name }}
          </text>
        </template>

        <template v-else-if="n.node.type === 'folder'">
          <rect
            x="-46" y="-18" width="92" height="36" rx="10"
            :class="hoverFolderId === n.node.id ? 'fill-amber-500/30' : selectedId === n.node.id ? 'fill-amber-500/20' : 'fill-slate-800'"
            :stroke="hoverFolderId === n.node.id ? 'rgb(251 191 36)' : 'rgb(71 85 105)'"
            :stroke-width="hoverFolderId === n.node.id ? 2.5 : 1.5"
          />
          <text x="0" y="-2" text-anchor="middle" class="pointer-events-none fill-amber-200 text-[11px] font-semibold">
            {{ n.node.name.length > 14 ? n.node.name.slice(0, 13) + '…' : n.node.name }}
          </text>
          <text x="0" y="11" text-anchor="middle" class="pointer-events-none fill-slate-500 text-[8px]">
            {{ n.node.leafCount }}
          </text>
        </template>

        <template v-else>
          <rect
            x="-58" y="-14" width="116" height="28" rx="14"
            :class="selectedId === n.node.id ? 'fill-rose-500/25' : 'fill-slate-900'"
            :stroke="selectedId === n.node.id ? 'rgb(244 63 94)' : 'rgb(51 65 85)'"
            stroke-width="1.5"
          />
          <image
            v-if="n.node.bookmark"
            :href="faviconUrl(n.node.bookmark.url)"
            x="-52" y="-9" width="18" height="18"
            preserveAspectRatio="xMidYMid meet"
            @error="($event.target as SVGImageElement).style.display = 'none'"
          />
          <text x="-30" y="0" dominant-baseline="central" class="pointer-events-none fill-slate-100 text-[10px]">
            {{ n.node.name.length > 18 ? n.node.name.slice(0, 17) + '…' : n.node.name }}
          </text>
        </template>
      </g>
    </g>

    <g
      v-if="drag?.mode === 'node'"
      :transform="`translate(${drag.current.x * zoom + pan.x} ${drag.current.y * zoom + pan.y})`"
      pointer-events="none"
    >
      <rect x="-58" y="-14" width="116" height="28" rx="14" class="fill-rose-500/80" stroke="rgb(244 63 94)" stroke-width="2" />
      <text x="0" y="0" text-anchor="middle" dominant-baseline="central" class="pointer-events-none fill-white text-[10px]">
        {{ drag.node.node.name.length > 18 ? drag.node.node.name.slice(0, 17) + '…' : drag.node.node.name }}
      </text>
    </g>
  </svg>
</template>
