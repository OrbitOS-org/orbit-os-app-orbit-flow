<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useFlowStore } from '@/stores/flowStore'
import { useUiStore } from '@/stores/uiStore'
import { useAuthStore } from '@/stores/authStore'
import OpenFlowSheet from '@/components/OpenFlowSheet.vue'
import { useApi } from '@/composables/useApi'

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

const auth = useAuthStore()

// ── Flow menu ────────────────────────────────────────────────────
const STATE_LABEL: Record<string, string> = {
  running: 'Running on the device',
  changed: 'Running · changes not deployed',
  new: 'Not deployed yet',
  stopped: 'Stopped on the device',
}
const menuFor = ref<string | null>(null)
const menuPos = ref({ x: 0, y: 0 })
const menuFlow = computed(() => flowStore.flows.find((f) => f.id === menuFor.value) ?? null)
const menuState = computed(() => (menuFor.value ? flowStore.flowState(menuFor.value) : 'running'))
const toast = ref<string | null>(null)
const toastBad = ref(false)
let toastTimer: ReturnType<typeof setTimeout> | undefined

function openMenu(id: string, e: MouseEvent) {
  if (menuFor.value === id) { menuFor.value = null; return }
  const r = (e.currentTarget as HTMLElement).getBoundingClientRect()
  menuPos.value = { x: Math.min(r.left - 8, window.innerWidth - 270), y: r.bottom + 8 }
  menuFor.value = id
}
function showToast(text: string, bad = false) {
  toast.value = text
  toastBad.value = bad
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toast.value = null }, bad ? 5000 : 2500)
}
// Disable / Enable, as in Node-RED: a change like any other, applied on
// the next Deploy together with the rest.
function toggleDisabled() {
  const flow = menuFlow.value
  menuFor.value = null
  if (!flow) return
  flowStore.toggleFlowDisabled(flow.id)
  showToast(flow.disabled
    ? `“${flow.label}” will stop when you deploy`
    : `“${flow.label}” will run when you deploy`)
}
function rename() {
  const id = menuFor.value
  menuFor.value = null
  if (id) ui.openFlowEditProperties(id)
}
function closeFlow() {
  const id = menuFor.value
  menuFor.value = null
  if (id) askDelete(id)
}
function closeMenuOnOutside(e: MouseEvent) {
  const t = e.target as HTMLElement
  if (!t.closest('.flow-menu') && !t.closest('.tab-more')) menuFor.value = null
}
onMounted(() => document.addEventListener('click', closeMenuOnOutside))
onUnmounted(() => document.removeEventListener('click', closeMenuOnOutside))

// Each tab is a flow; "Close…" deletes or saves it after a confirmation. The last
// flow cannot be deleted, and only users who can deploy can delete.
const canDelete = computed(() => auth.can('deploy') && flowStore.flows.length > 1)
const deletingId = ref<string | null>(null)
const deleting = computed(() => flowStore.flows.find((f) => f.id === deletingId.value) ?? null)
const deletingNodes = computed(() => {
  if (!deleting.value) return 0
  return deleting.value.id === flowStore.activeFlowId ? flowStore.nodes.length : deleting.value.nodes.length
})
function askDelete(id: string) {
  closeError.value = null
  deletingId.value = id
}
const api = useApi()
const saving = ref(false)
const closeError = ref<string | null>(null)

// Saves the flow (and the connections it uses) to the app's data folder,
// then removes it from the editor.
async function saveAndClose() {
  if (!deletingId.value) return
  const entry = flowStore.exportFlow(deletingId.value)
  if (!entry) return
  saving.value = true
  closeError.value = null
  try {
    await api.saveLibraryFlow(entry)
    confirmDelete()
  } catch (err) {
    closeError.value = err instanceof Error ? err.message : 'The flow could not be saved'
  } finally {
    saving.value = false
  }
}

function confirmDelete() {
  if (!deletingId.value) return
  const ctx = ui.propertiesContext as { type?: string; flowId?: string } | null
  if (ctx?.type === 'flow-edit' && ctx.flowId === deletingId.value) ui.clearFlowProperties()
  flowStore.removeFlow(deletingId.value)
  deletingId.value = null
}
function onEscDelete(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  if (deletingId.value) deletingId.value = null
  else if (menuFor.value) menuFor.value = null
}
onMounted(() => document.addEventListener('keydown', onEscDelete))
onUnmounted(() => document.removeEventListener('keydown', onEscDelete))

// "+" opens a small menu: a new empty flow, or one from an example.
const addMenuOpen = ref(false)
const examplesOpen = ref(false)

function newFlow() {
  addMenuOpen.value = false
  ui.openFlowCreateProperties()
}
function fromExample() {
  addMenuOpen.value = false
  examplesOpen.value = true
}
function closeMenuOutside(e: MouseEvent) {
  if (!(e.target as HTMLElement).closest('.add-menu')) addMenuOpen.value = false
}
onMounted(() => document.addEventListener('click', closeMenuOutside))
onUnmounted(() => document.removeEventListener('click', closeMenuOutside))

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
    <div
      v-for="(flow, idx) in flowStore.flows"
      :key="flow.id"
      role="tab"
      tabindex="0"
      :aria-selected="flow.id === flowStore.activeFlowId"
      draggable="true"
      class="seg-item group relative flex items-center gap-1.5 whitespace-nowrap cursor-pointer"
      :class="[
        flow.id === flowStore.activeFlowId
          ? 'on'
          : '',
        flow.disabled ? 'opacity-40' : '',
        dragIdx === idx ? 'opacity-30' : '',
        dropIdx === idx && dragIdx !== idx ? 'drop' : '',
      ]"
      @click="switchFlow(flow.id)"
      @keydown.enter.prevent="switchFlow(flow.id)"
      @keydown.space.prevent="switchFlow(flow.id)"
      @dblclick="handleDblClick(flow.id)"
      @dragstart="onDragStart(idx, $event)"
      @dragover="onDragOver(idx, $event)"
      @drop="onDrop(idx)"
      @dragend="onDragEnd"
    >
      <span
        class="state-mark"
        :class="flowStore.flowState(flow.id)"
        :title="STATE_LABEL[flowStore.flowState(flow.id)]"
        aria-hidden="true"
      />
      <span class="truncate max-w-[140px]" :class="{ 'text-terminal-text-dim line-through decoration-1': flow.disabled, 'text-terminal-text-dim': flowStore.flowState(flow.id) === 'stopped' }">{{ flow.label }}</span>
      <span class="sr-only">({{ STATE_LABEL[flowStore.flowState(flow.id)] }})</span>
      <!-- Flow menu: state, stop/start, deploy this flow, rename, close -->
      <button
        class="tab-more"
        :class="flow.id === flowStore.activeFlowId || menuFor === flow.id ? 'opacity-100' : 'opacity-0 group-hover:opacity-100 focus-visible:opacity-100'"
        :aria-label="`Menu of ${flow.label}`"
        :aria-expanded="menuFor === flow.id"
        @click.stop="openMenu(flow.id, $event)"
        @dblclick.stop
        @mousedown.stop
      >
        <svg viewBox="0 0 16 4" class="w-3 h-1" aria-hidden="true"><circle cx="2" cy="2" r="1.6" fill="currentColor" /><circle cx="8" cy="2" r="1.6" fill="currentColor" /><circle cx="14" cy="2" r="1.6" fill="currentColor" /></svg>
      </button>
    </div>

    </div>

    <!-- Add flow: new or from an example -->
    <div class="add-menu relative shrink-0">
      <button
        class="flex items-center justify-center w-7 h-7 rounded-full text-terminal-text-dim hover:text-accent bg-terminal-surface-alt/70 hover:bg-terminal-surface-alt transition-colors duration-100"
        title="Add flow"
        aria-label="Add flow"
        :aria-expanded="addMenuOpen"
        @click="auth.can('deploy') ? (addMenuOpen = !addMenuOpen) : newFlow()"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 5v14m-7-7h14" />
        </svg>
      </button>
      <div v-if="addMenuOpen" class="ios-menu fixed mt-2 w-48 z-50 p-1" role="menu">
        <button class="menu-item" role="menuitem" @click="newFlow">New flow</button>
        <button class="menu-item" role="menuitem" @click="fromExample">Open…</button>
      </div>
    </div>
  </div>
  <OpenFlowSheet v-if="examplesOpen" @close="examplesOpen = false" />

  <!-- Flow menu -->
  <div
    v-if="menuFlow"
    class="flow-menu ios-menu fixed z-50 w-64 p-1"
    :style="{ left: menuPos.x + 'px', top: menuPos.y + 'px' }"
    role="menu"
  >
    <div class="flex items-center gap-2 px-2.5 pt-2 pb-2.5 mb-1 border-b border-terminal-border text-[12.5px] text-terminal-text-dim">
      <span class="state-mark" :class="menuState" aria-hidden="true" />
      {{ STATE_LABEL[menuState] }}
    </div>
    <template v-if="auth.can('deploy')">
      <button class="menu-item" role="menuitem" @click="toggleDisabled">
        {{ menuFlow.disabled ? 'Enable flow' : 'Disable flow' }}<small>on the next deploy</small>
      </button>
      <div class="menu-sep" />
      <button class="menu-item" role="menuitem" @click="rename">Rename…</button>
      <div v-if="canDelete" class="menu-sep" />
      <button v-if="canDelete" class="menu-item danger" role="menuitem" @click="closeFlow">Close…</button>
    </template>
  </div>

  <!-- Result of a flow action -->
  <Transition name="flow-toast">
    <div v-if="toast" class="flow-toast" :class="{ bad: toastBad }" role="status">{{ toast }}</div>
  </Transition>

  <!-- Confirm deleting a flow -->
  <div
    v-if="deleting"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-[2px]"
    @click.self="deletingId = null"
  >
    <div class="ios-dialog w-[340px] max-w-[90vw] p-5 text-center" role="dialog" aria-modal="true" aria-labelledby="del-title">
      <h2 id="del-title">Close “{{ deleting.label }}”?</h2>
      <p class="mt-2 text-[13px] text-terminal-text-dim">
        Its {{ deletingNodes }} node{{ deletingNodes === 1 ? '' : 's' }} stop running when you deploy.
        Save it to reopen it later with + → Open…
      </p>
      <p v-if="closeError" class="mt-3 settings-error">{{ closeError }}</p>
      <div class="mt-4 grid gap-2">
        <button class="pill pill-primary py-2" :disabled="saving" @click="saveAndClose">{{ saving ? 'Saving…' : 'Save and close' }}</button>
        <button class="pill py-2 bg-status-error/10 text-status-error font-semibold" :disabled="saving" @click="confirmDelete">Delete</button>
        <button class="pill pill-plain py-2" :disabled="saving" @click="deletingId = null">Cancel</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* iOS segmented control */
.ios-seg { display: inline-flex; gap: 2px; padding: 2px; border-radius: 9px; background: rgb(var(--c-surface-alt) / 0.8); max-width: 100%; }
.seg-item { padding: 4px 12px; border-radius: 7px; font-size: 13px; color: rgb(var(--c-text)); transition: background 0.15s; }
.seg-item:hover:not(.on) { background: rgb(var(--c-surface) / 0.5); }
.seg-item.on { background: rgb(var(--c-surface)); font-weight: 600; box-shadow: 0 1px 3px rgb(0 0 0 / 0.12), 0 0 0 0.5px rgb(0 0 0 / 0.04); }
:root[data-theme="dark"] .seg-item.on { background: rgb(99 99 102); }
.state-mark { width: 9px; height: 9px; border-radius: 9999px; flex: none; display: inline-block; }
.state-mark.running { background: rgb(var(--c-success)); }
.state-mark.changed { margin-inline: 3px; background: rgb(var(--c-success)); box-shadow: 0 0 0 2px rgb(var(--c-surface)), 0 0 0 3.5px rgb(var(--c-accent)); }
.state-mark.new { border: 1.6px dashed rgb(var(--c-accent)); }
.state-mark.stopped { border: 1.8px solid rgb(var(--c-text-dim)); }
.tab-more { width: 18px; height: 18px; margin-right: -4px; border-radius: 9999px; display: grid; place-items: center; color: rgb(var(--c-text-dim)); transition: opacity 0.12s, background-color 0.12s; }
.tab-more:hover { background: rgb(var(--c-text-dim) / 0.2); color: rgb(var(--c-text)); }
.menu-item small { font-size: 11.5px; color: rgb(var(--c-text-dim)); }
.menu-item:hover small { color: rgb(255 255 255 / 0.8); }
.menu-item.danger { color: rgb(var(--c-error)); }
.menu-item.danger:hover { background: rgb(var(--c-error)); color: #fff; }
.menu-item:disabled { opacity: 0.5; cursor: progress; }
.menu-sep { height: 1px; margin: 4px 6px; background: rgb(var(--c-border)); }
.flow-toast { position: fixed; left: 50%; bottom: 24px; transform: translateX(-50%); z-index: 60; padding: 9px 16px; border-radius: 14px; font-size: 13px; background: var(--bar-bg); backdrop-filter: saturate(180%) blur(20px); -webkit-backdrop-filter: saturate(180%) blur(20px); border: 1px solid rgb(var(--c-border)); color: rgb(var(--c-text)); box-shadow: 0 8px 24px rgb(0 0 0 / 0.16); }
.flow-toast.bad { color: rgb(var(--c-error)); }
.flow-toast-enter-active, .flow-toast-leave-active { transition: opacity 0.2s, transform 0.25s; }
.flow-toast-enter-from, .flow-toast-leave-to { opacity: 0; transform: translate(-50%, 12px); }
.menu-item { display: flex; justify-content: space-between; align-items: center; width: 100%; text-align: left; padding: 7px 10px; border-radius: 8px; font-size: 13px; color: rgb(var(--c-text)); }
.menu-item:hover { background: rgb(var(--c-accent)); color: #fff; }
.seg-item.drop { box-shadow: inset 2px 0 0 rgb(var(--c-accent)); }
</style>
