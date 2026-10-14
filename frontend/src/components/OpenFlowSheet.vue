<!--
  Copyright (C) 2026 Orbit OS
  Licensed under the GNU Affero General Public License v3.0 or later.
  See LICENSE file for details.

  "Open…": the flows saved with "Save and close" (in the app's data folder)
  and the examples that ship with the app. The chosen one is added as a
  new, undeployed flow.
-->
<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useApi, type ExampleSummary, type LibrarySummary, type Example } from '@/composables/useApi'
import { useFlowStore } from '@/stores/flowStore'

const emit = defineEmits<{ close: [] }>()
const api = useApi()
const flowStore = useFlowStore()

const saved = ref<LibrarySummary[]>([])
const examples = ref<ExampleSummary[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const busy = ref<string | null>(null)
const confirmingDelete = ref<string | null>(null)

// Examples and saved flows are separate tabs; the sheet opens on Saved
// when there is something saved.
const tab = ref<'examples' | 'saved'>('examples')
const query = ref('')
const CATEGORIES = ['All', 'Getting started', 'Device', 'Industrial', 'Data', 'Projects']
const category = ref('All')

const q = computed(() => query.value.trim().toLowerCase())
const shownSaved = computed(() => saved.value.filter((f) => !q.value || f.name.toLowerCase().includes(q.value)))
const shownExamples = computed(() =>
  examples.value.filter(
    (e) =>
      (category.value === 'All' || e.category === category.value) &&
      (!q.value || `${e.name} ${e.description} ${e.needs ?? ''}`.toLowerCase().includes(q.value)),
  ),
)
const exampleGroups = computed(() =>
  CATEGORIES.slice(1)
    .map((c) => ({ title: c, items: shownExamples.value.filter((e) => e.category === c) }))
    .concat([{ title: 'Other', items: shownExamples.value.filter((e) => !CATEGORIES.includes(e.category)) }])
    .filter((g) => g.items.length > 0),
)

onMounted(async () => {
  document.addEventListener('keydown', onKey)
  try {
    const [lib, ex] = await Promise.all([api.listLibrary(), api.listExamples()])
    saved.value = lib
    examples.value = ex
    if (lib.length > 0) tab.value = 'saved'
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'The list could not be loaded'
  } finally {
    loading.value = false
  }
})
onUnmounted(() => document.removeEventListener('keydown', onKey))

function onKey(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  if (confirmingDelete.value) confirmingDelete.value = null
  else emit('close')
}

async function open(key: string, load: () => Promise<Example>) {
  busy.value = key
  try {
    flowStore.importExample(await load())
    emit('close')
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'The flow could not be added'
  } finally {
    busy.value = null
  }
}

async function removeSaved(id: string) {
  try {
    await api.deleteLibraryFlow(id)
    saved.value = saved.value.filter((s) => s.id !== id)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'The saved flow could not be deleted'
  } finally {
    confirmingDelete.value = null
  }
}

function savedOn(iso: string): string {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? '' : d.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-[2px]" @click.self="emit('close')">
    <div class="sheet" role="dialog" aria-modal="true" aria-labelledby="open-title">
      <header class="flex items-center justify-between px-4 pt-4 pb-1">
        <h2 id="open-title" class="text-[17px] font-semibold text-terminal-text-bright">Open</h2>
        <button
          class="w-7 h-7 flex items-center justify-center rounded-full bg-terminal-surface-alt/80 text-terminal-text-dim hover:text-terminal-text"
          aria-label="Close"
          @click="emit('close')"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </header>
      <div class="px-4 pb-2 space-y-3">
        <div class="seg flex" role="group" aria-label="Show">
          <button :aria-pressed="tab === 'examples'" @click="tab = 'examples'">Examples<span class="n">{{ examples.length }}</span></button>
          <button :aria-pressed="tab === 'saved'" @click="tab = 'saved'">Saved<span class="n">{{ saved.length }}</span></button>
        </div>
        <label class="ios-search">
          <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.2" stroke-linecap="round">
            <circle cx="11" cy="11" r="6.5" /><path d="m20 20-4.2-4.2" />
          </svg>
          <input v-model="query" type="text" placeholder="Search" aria-label="Search" />
        </label>
      </div>
      <div v-if="tab === 'examples'" class="chips" role="group" aria-label="Category">
        <button v-for="c in CATEGORIES" :key="c" class="chip" :aria-pressed="category === c" @click="category = c">{{ c }}</button>
      </div>

      <div class="px-3 pb-4 h-[52vh] overflow-y-auto space-y-4">
        <div v-if="loading" class="px-2 py-6 text-center text-xs text-terminal-text-dim">Loading…</div>
        <div v-if="error" class="settings-error">{{ error }}</div>

        <template v-if="!loading && tab === 'saved'">
          <div v-if="shownSaved.length === 0" class="list px-4 py-6 text-center text-[13px] text-terminal-text-dim">
            {{ saved.length === 0 ? 'No saved flows. Close a flow with “Save and close” to keep it here.' : 'No saved flows match.' }}
          </div>
          <div v-else class="list">
            <div v-for="f in shownSaved" :key="f.id" class="row">
              <button class="flex-1 min-w-0 text-left" :disabled="busy !== null" @click="open('lib:' + f.id, () => api.getLibraryFlow(f.id))">
                <span class="block text-[15px] text-terminal-text-bright truncate">{{ f.name }}</span>
                <span class="block text-[12.5px] text-terminal-text-dim mt-0.5">
                  {{ f.nodes }} node{{ f.nodes === 1 ? '' : 's' }}<span v-if="savedOn(f.savedAt)"> · saved {{ savedOn(f.savedAt) }}</span>
                </span>
              </button>
              <template v-if="confirmingDelete === f.id">
                <button class="pill pill-plain" @click="confirmingDelete = null">Keep</button>
                <button class="pill bg-status-error text-white font-semibold" @click="removeSaved(f.id)">Delete</button>
              </template>
              <button
                v-else
                class="w-7 h-7 rounded-full flex items-center justify-center text-terminal-text-dim hover:text-status-error hover:bg-status-error/10"
                :aria-label="`Delete saved flow ${f.name}`"
                title="Delete"
                @click="confirmingDelete = f.id"
              >
                <svg viewBox="0 0 24 24" class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <path d="M4 7h16M10 11v6M14 11v6M6 7l1 12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-12M9 7V4h6v3" />
                </svg>
              </button>
            </div>
          </div>
        </template>

        <template v-if="!loading && tab === 'examples'">
          <div v-if="exampleGroups.length === 0" class="list px-4 py-6 text-center text-[13px] text-terminal-text-dim">No examples match.</div>
          <div v-for="g in exampleGroups" :key="g.title">
            <div class="info-title">{{ g.title }}</div>
            <div class="list">
              <button
                v-for="ex in g.items"
                :key="ex.id"
                class="row w-full text-left"
                :disabled="busy !== null"
                @click="open('ex:' + ex.id, () => api.getExample(ex.id))"
              >
                <span class="min-w-0 flex-1 grid gap-[3px]">
                  <span class="text-[15px] text-terminal-text-bright">{{ ex.name }}</span>
                  <span class="text-[12.5px] text-terminal-text-dim leading-snug">{{ ex.description }}</span>
                  <span class="tag" :class="ex.needs ? 'hw' : 'free'">{{ ex.needs ? `Needs: ${ex.needs}` : 'No hardware' }}</span>
                </span>
                <svg class="w-2 h-3.5 shrink-0 text-terminal-text-dim/70" viewBox="0 0 8 13" aria-hidden="true">
                  <path d="M1.5 1.5 6.5 6.5 1.5 11.5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
                </svg>
              </button>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sheet {
  width: 520px; max-width: calc(100vw - 32px);
  background: rgb(var(--c-bg)); border-radius: 16px;
  box-shadow: 0 20px 50px rgb(0 0 0 / 0.3);
}
.list { background: rgb(var(--c-surface)); border-radius: 12px; overflow: hidden; }
.row { display: flex; align-items: center; gap: 10px; padding: 10px 14px; }
.row + .row { border-top: 1px solid rgb(var(--c-border)); }
button.row:hover:not(:disabled), .row:hover { background: rgb(var(--c-surface-alt) / 0.5); }
.row :disabled, .row:disabled { cursor: progress; }
.seg { gap: 2px; padding: 2px; border-radius: 9px; background: rgb(var(--c-surface-alt) / 0.8); }
.seg button { flex: 1; padding: 5px 10px; border-radius: 7px; font-size: 13px; color: rgb(var(--c-text)); }
.seg button[aria-pressed="true"] { background: rgb(var(--c-surface)); font-weight: 600; box-shadow: 0 1px 3px rgb(0 0 0 / 0.12); }
:root[data-theme="dark"] .seg button[aria-pressed="true"] { background: rgb(99 99 102); }
.seg .n { margin-left: 5px; font-weight: 400; color: rgb(var(--c-text-dim)); }
.ios-search { display: flex; align-items: center; gap: 7px; background: rgb(var(--c-surface-alt) / 0.8); border-radius: 10px; padding: 7px 10px; color: rgb(var(--c-text-dim)); }
.ios-search input { flex: 1; min-width: 0; border: 0; outline: none; background: transparent; font-size: 15px; color: rgb(var(--c-text)); }
.ios-search:focus-within { box-shadow: 0 0 0 2px rgb(var(--c-accent) / 0.5); }
.chips { display: flex; gap: 6px; overflow-x: auto; scrollbar-width: none; padding: 0 16px 10px; }
.chip { border-radius: 9999px; padding: 4px 12px; font-size: 13px; font-weight: 500; white-space: nowrap; background: rgb(var(--c-surface-alt) / 0.8); color: rgb(var(--c-text)); }
.chip[aria-pressed="true"] { background: rgb(var(--c-accent)); color: #fff; }
.tag { justify-self: start; margin-top: 2px; padding: 1px 8px; border-radius: 9999px; font-size: 11.5px; font-weight: 500; }
.tag.free { background: rgb(var(--c-success) / 0.14); color: rgb(var(--c-success)); }
.tag.hw { background: rgb(var(--c-warning) / 0.16); color: rgb(154 91 0); }
:root[data-theme="dark"] .tag.hw { color: rgb(255 179 64); }
:root:not([data-theme="dark"]) .tag.free { color: rgb(36 138 61); }
</style>
