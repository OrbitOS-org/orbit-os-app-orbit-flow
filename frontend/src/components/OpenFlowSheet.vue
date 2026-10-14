<!--
  Copyright (C) 2026 Orbit OS
  Licensed under the GNU Affero General Public License v3.0 or later.
  See LICENSE file for details.

  "Open…": the flows saved with "Save and close" (in the app's data folder)
  and the examples that ship with the app. The chosen one is added as a
  new, undeployed flow.
-->
<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
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

onMounted(async () => {
  document.addEventListener('keydown', onKey)
  try {
    const [lib, ex] = await Promise.all([api.listLibrary(), api.listExamples()])
    saved.value = lib
    examples.value = ex
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
      <p class="px-4 pb-3 text-xs text-terminal-text-dim">
        The flow is added as a new tab. Check its settings, then deploy.
      </p>

      <div class="px-3 pb-4 max-h-[64vh] overflow-y-auto space-y-4">
        <div v-if="loading" class="px-2 py-6 text-center text-xs text-terminal-text-dim">Loading…</div>
        <div v-if="error" class="settings-error">{{ error }}</div>

        <template v-if="!loading">
          <div>
            <div class="info-title">Saved flows</div>
            <div v-if="saved.length === 0" class="list px-4 py-3 text-[13px] text-terminal-text-dim">
              None yet. Close a flow with “Save and close” to keep it here.
            </div>
            <div v-else class="list">
              <div v-for="f in saved" :key="f.id" class="row">
                <button class="flex-1 min-w-0 text-left" :disabled="busy !== null" @click="open('lib:' + f.id, () => api.getLibraryFlow(f.id))">
                  <span class="block text-[14px] text-terminal-text-bright truncate">{{ f.name }}</span>
                  <span class="block text-xs text-terminal-text-dim mt-0.5">
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
          </div>

          <div>
            <div class="info-title">Examples</div>
            <div v-if="examples.length === 0" class="list px-4 py-3 text-[13px] text-terminal-text-dim">No examples.</div>
            <div v-else class="list">
              <button
                v-for="ex in examples"
                :key="ex.id"
                class="row w-full text-left"
                :disabled="busy !== null"
                @click="open('ex:' + ex.id, () => api.getExample(ex.id))"
              >
                <span class="min-w-0 flex-1">
                  <span class="block text-[14px] text-terminal-text-bright">{{ ex.name }}</span>
                  <span class="block text-xs text-terminal-text-dim mt-0.5">{{ ex.description }}</span>
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
  width: 460px; max-width: calc(100vw - 32px);
  background: rgb(var(--c-bg)); border-radius: 16px;
  box-shadow: 0 20px 50px rgb(0 0 0 / 0.3);
}
.list { background: rgb(var(--c-surface)); border-radius: 12px; overflow: hidden; }
.row { display: flex; align-items: center; gap: 10px; padding: 10px 14px; }
.row + .row { border-top: 1px solid rgb(var(--c-border)); }
button.row:hover:not(:disabled), .row:hover { background: rgb(var(--c-surface-alt) / 0.5); }
.row :disabled, .row:disabled { cursor: progress; }
</style>
