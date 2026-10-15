<script setup lang="ts">
// The "File" field of the file nodes: a name inside the app's files
// ("logs/readings.csv"), typed or picked with Browse from what is there.
// The file nodes cannot reach anything else on the device.
import { computed, ref } from 'vue'
import { useNodeProperty } from '@/composables/useNodeProperty'
import { useApi, type AppFile } from '@/composables/useApi'
import FormField from '@/components/ui/FormField.vue'

const props = withDefaults(defineProps<{
  label?: string
  /** "file": only files can be picked; "any": folders too. */
  pick?: 'file' | 'any'
  hint?: string
}>(), { label: 'File', pick: 'file', hint: '' })

const api = useApi()
const path = useNodeProperty<string>('path', '')

const open = ref(false)
const loading = ref(false)
const error = ref('')
const files = ref<AppFile[]>([])

const shown = computed(() => files.value.filter((f) => props.pick === 'any' || !f.isDir))

async function browse() {
  open.value = !open.value
  if (!open.value) return
  loading.value = true
  error.value = ''
  try {
    files.value = (await api.listFiles()).files
  } catch {
    error.value = 'Could not list the files'
  } finally {
    loading.value = false
  }
}

function choose(f: AppFile) {
  path.value = f.path
  open.value = false
}

function size(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

// Absolute paths or ".." cannot work on Orbit OS: say so as it is typed.
const outside = computed(() => path.value.startsWith('/') || path.value.split('/').includes('..'))

// Written as a constant: "{{" inside the template would be read by Vue.
const TEMPLATE_HINT = '{{topic}}'
</script>

<template>
  <FormField :label="label" :error="outside ? 'Use a name inside the app\'s files, e.g. logs/readings.csv' : ''">
    <div class="file-field">
      <span class="pre">files /</span>
      <input v-model="path" class="flex-1 min-w-0" placeholder="logs/readings.csv" spellcheck="false" />
      <button type="button" class="browse" :aria-expanded="open" @click="browse">Browse</button>
    </div>
    <div v-if="open" class="picker">
      <div v-if="loading" class="empty">Loading…</div>
      <div v-else-if="error" class="empty">{{ error }}</div>
      <div v-else-if="shown.length === 0" class="empty">No files yet. Type a name: Write File creates it.</div>
      <button v-for="f in shown" v-else :key="f.path" type="button" class="row" :class="{ on: f.path === path }" @click="choose(f)">
        <span class="ico">{{ f.isDir ? '▸' : '▤' }}</span>
        <span class="name">{{ f.path }}</span>
        <span class="meta">{{ f.isDir ? 'folder' : size(f.size) }}</span>
      </button>
    </div>
    <div class="text-[10px] text-terminal-text-dim leading-tight">
      {{ hint || 'Inside the app\'s files.' }} Empty: uses msg.filename. {{ TEMPLATE_HINT }} and other message properties work in the name.
    </div>
  </FormField>
</template>

<style scoped>
.file-field { display: flex; align-items: center; gap: 6px; background: rgb(var(--c-surface-alt) / 0.7); border-radius: 10px; padding: 6px 6px 6px 10px; }
.file-field .pre { color: rgb(var(--c-text-dim) / 0.8); font-family: var(--font-mono, ui-monospace, monospace); font-size: 12px; white-space: nowrap; }
.file-field input { background: transparent; border: 0; outline: none; color: rgb(var(--c-text)); font-family: var(--font-mono, ui-monospace, monospace); font-size: 12.5px; }
.browse { border: 0; border-radius: 7px; padding: 3px 9px; font-size: 11.5px; font-weight: 600; color: rgb(var(--c-accent)); background: rgb(var(--c-accent) / 0.1); }
.browse:hover { background: rgb(var(--c-accent) / 0.18); }
.picker { margin-top: 6px; max-height: 220px; overflow-y: auto; border: 1px solid rgb(var(--c-border)); border-radius: 10px; background: rgb(var(--c-surface)); }
.picker .row { width: 100%; display: flex; align-items: center; gap: 8px; padding: 7px 10px; text-align: left; font-size: 12.5px; color: rgb(var(--c-text)); }
.picker .row + .row { border-top: 1px solid rgb(var(--c-border) / 0.6); }
.picker .row:hover { background: rgb(var(--c-surface-alt)); }
.picker .row.on { background: rgb(var(--c-accent) / 0.12); }
.picker .ico { width: 14px; color: rgb(var(--c-text-dim)); }
.picker .name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: var(--font-mono, ui-monospace, monospace); }
.picker .meta { font-size: 11px; color: rgb(var(--c-text-dim)); }
.picker .empty { padding: 10px; font-size: 12px; color: rgb(var(--c-text-dim)); }
</style>
