<script setup lang="ts">
import { ref } from 'vue'
import { useFlowStore } from '@/stores/flowStore'
import { useUiStore } from '@/stores/uiStore'

const flowStore = useFlowStore()
const ui = useUiStore()

function switchFlow(flowId: string) {
  if (flowId === flowStore.activeFlowId) return
  flowStore.syncCanvasToActiveFlow()
  flowStore.setActiveFlow(flowId)
  ui.clearFlowProperties()
}

function handleDblClick(flowId: string) {
  ui.openFlowEditProperties(flowId)
}

function handleAddClick() {
  ui.openFlowCreateProperties()
}

// ── Drag & Drop ──────────────────────────────────────────────────
const dragIdx = ref<number | null>(null)
const dropIdx = ref<number | null>(null)

function onDragStart(idx: number, e: DragEvent) {
  dragIdx.value = idx
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', String(idx))
  }
}

function onDragOver(idx: number, e: DragEvent) {
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  dropIdx.value = idx
}

function onDrop(idx: number) {
  if (dragIdx.value !== null && dragIdx.value !== idx) {
    flowStore.reorderFlows(dragIdx.value, idx)
  }
  dragIdx.value = null
  dropIdx.value = null
}

function onDragEnd() {
  dragIdx.value = null
  dropIdx.value = null
}
</script>

<template>
  <div class="flex items-center gap-2 h-11 px-3 bg-terminal-surface border-b border-terminal-border select-none shrink-0 overflow-x-auto">
    <div class="ios-seg">
    <!-- Flow Tabs -->
    <button
      v-for="(flow, idx) in flowStore.flows"
      :key="flow.id"
      draggable="true"
      class="seg-item relative flex items-center gap-1.5 whitespace-nowrap cursor-pointer"
      :class="[
        flow.id === flowStore.activeFlowId
          ? 'on'
          : '',
        flow.disabled ? 'opacity-40' : '',
        dragIdx === idx ? 'opacity-30' : '',
        dropIdx === idx && dragIdx !== idx ? 'drop' : '',
      ]"
      @click="switchFlow(flow.id)"
      @dblclick="handleDblClick(flow.id)"
      @dragstart="onDragStart(idx, $event)"
      @dragover="onDragOver(idx, $event)"
      @drop="onDrop(idx)"
      @dragend="onDragEnd"
    >
      <span class="truncate max-w-[120px]">{{ flow.label }}</span>
      <span
        v-if="flowStore.isFlowDirty(flow.id)"
        class="w-1.5 h-1.5 rounded-full bg-accent"
        style="box-shadow: 0 0 4px rgb(var(--c-accent) / 0.5)"
      />
    </button>

    </div>

    <!-- Add Flow Button -->
    <button
      class="flex items-center justify-center w-7 h-7 shrink-0 rounded-full text-terminal-text-dim hover:text-accent bg-terminal-surface-alt/70 hover:bg-terminal-surface-alt transition-colors duration-100"
      title="New flow"
      @click="handleAddClick"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 5v14m-7-7h14" />
      </svg>
    </button>
  </div>
</template>

<style scoped>
/* iOS segmented control */
.ios-seg { display: inline-flex; gap: 2px; padding: 2px; border-radius: 9px; background: rgb(var(--c-surface-alt) / 0.8); max-width: 100%; }
.seg-item { padding: 4px 12px; border-radius: 7px; font-size: 13px; color: rgb(var(--c-text)); transition: background 0.15s; }
.seg-item:hover:not(.on) { background: rgb(var(--c-surface) / 0.5); }
.seg-item.on { background: rgb(var(--c-surface)); font-weight: 600; box-shadow: 0 1px 3px rgb(0 0 0 / 0.12), 0 0 0 0.5px rgb(0 0 0 / 0.04); }
:root[data-theme="dark"] .seg-item.on { background: rgb(99 99 102); }
.seg-item.drop { box-shadow: inset 2px 0 0 rgb(var(--c-accent)); }
</style>
