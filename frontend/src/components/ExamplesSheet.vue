<!--
  Copyright (C) 2026 Orbit OS
  Licensed under the GNU Affero General Public License v3.0 or later.
  See LICENSE file for details.

  Lists the example flows that ship with the app and adds the chosen one
  as a new, undeployed flow.
-->
<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useApi, type ExampleSummary } from '@/composables/useApi'
import { useFlowStore } from '@/stores/flowStore'

const emit = defineEmits<{ close: [] }>()
const api = useApi()
const flowStore = useFlowStore()

const examples = ref<ExampleSummary[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const busy = ref<string | null>(null)

onMounted(async () => {
  document.addEventListener('keydown', onKey)
  try {
    examples.value = await api.listExamples()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Examples could not be loaded'
  } finally {
    loading.value = false
  }
})
onUnmounted(() => document.removeEventListener('keydown', onKey))

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}

async function choose(id: string) {
  busy.value = id
  try {
    flowStore.importExample(await api.getExample(id))
    emit('close')
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'The example could not be added'
  } finally {
    busy.value = null
  }
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-[2px]" @click.self="emit('close')">
    <div class="sheet" role="dialog" aria-modal="true" aria-labelledby="examples-title">
      <header class="flex items-center justify-between px-4 pt-4 pb-2">
        <h2 id="examples-title" class="text-[17px] font-semibold text-terminal-text-bright">Examples</h2>
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
        The example is added as a new flow. Check its settings, then deploy.
      </p>

      <div class="px-3 pb-4 max-h-[60vh] overflow-y-auto">
        <div v-if="loading" class="px-2 py-6 text-center text-xs text-terminal-text-dim">Loading…</div>
        <div v-else-if="error" class="px-2 py-3 text-xs text-status-error">{{ error }}</div>
        <div v-else-if="examples.length === 0" class="px-2 py-6 text-center text-xs text-terminal-text-dim">No examples.</div>
        <div v-else class="list">
          <button
            v-for="ex in examples"
            :key="ex.id"
            class="row"
            :disabled="busy !== null"
            @click="choose(ex.id)"
          >
            <span class="min-w-0 flex-1 text-left">
              <span class="block text-[14px] text-terminal-text-bright">{{ ex.name }}</span>
              <span class="block text-xs text-terminal-text-dim mt-0.5">{{ ex.description }}</span>
            </span>
            <svg v-if="busy !== ex.id" class="w-2 h-3.5 shrink-0 text-terminal-text-dim/70" viewBox="0 0 8 13" aria-hidden="true">
              <path d="M1.5 1.5 6.5 6.5 1.5 11.5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
            </svg>
            <span v-else class="text-xs text-terminal-text-dim">Adding…</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sheet {
  width: 440px; max-width: calc(100vw - 32px);
  background: rgb(var(--c-bg)); border-radius: 16px;
  box-shadow: 0 20px 50px rgb(0 0 0 / 0.3);
}
.list { background: rgb(var(--c-surface)); border-radius: 12px; overflow: hidden; }
.row { width: 100%; display: flex; align-items: center; gap: 12px; padding: 11px 14px; cursor: pointer; }
.row + .row { border-top: 1px solid rgb(var(--c-border)); }
.row:hover:not(:disabled) { background: rgb(var(--c-surface-alt) / 0.6); }
.row:disabled { cursor: progress; }
</style>
