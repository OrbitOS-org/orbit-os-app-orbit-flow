<!--
  Copyright (C) 2026 Orbit OS
  Licensed under the GNU Affero General Public License v3.0 or later.
  See LICENSE file for details.

  Settings → Files → Open: looks at a file on the device without
  downloading it. CSV as a table, JSON formatted, other text as it is.
  Only the end of a big file is loaded (the newest lines of a log);
  "Load earlier lines" goes back, and Follow shows new lines as they come.
  Read-only.
-->
<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useApi, type FileView, type ApiError } from '@/composables/useApi'
import { parseCSV } from '@/lib/csv'

const props = defineProps<{ path: string }>()
const emit = defineEmits<{ close: [] }>()

const api = useApi()
const view = ref<FileView | null>(null)
const text = ref('')
const start = ref(0)
const loading = ref(true)
const error = ref<string | null>(null)
const follow = ref(false)
const body = ref<HTMLElement | null>(null)

async function loadLatest(scrollDown: boolean) {
  try {
    const v = await api.viewFile(props.path)
    view.value = v
    text.value = v.text
    start.value = v.start
    error.value = null
    loading.value = false // render the content before scrolling to its end
    if (scrollDown) {
      await nextTick()
      if (body.value) body.value.scrollTop = body.value.scrollHeight
    }
  } catch (err) {
    error.value = api.isApiError(err) ? (err as ApiError).message : 'Could not open the file.'
  } finally {
    loading.value = false
  }
}

async function loadEarlier() {
  if (!start.value) return
  try {
    const v = await api.viewFile(props.path, start.value)
    const el = body.value
    const before = el ? el.scrollHeight : 0
    text.value = v.text + text.value
    start.value = v.start
    follow.value = false // following shows only the end
    await nextTick()
    if (el) el.scrollTop += el.scrollHeight - before
  } catch (err) {
    error.value = api.isApiError(err) ? (err as ApiError).message : 'Could not read the file.'
  }
}

// Follow: reload the end every 3 s, as long as the viewer is open.
let timer: ReturnType<typeof setInterval> | null = null
watch(follow, (on) => {
  if (timer) { clearInterval(timer); timer = null }
  if (on) {
    loadLatest(true)
    timer = setInterval(() => loadLatest(true), 3000)
  }
})

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') { e.stopPropagation(); emit('close') }
}
onMounted(() => {
  loadLatest(true)
  window.addEventListener('keydown', onKey, true)
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
  window.removeEventListener('keydown', onKey, true)
})

// CSV: the column names come from the file's first line, also when only
// its end is loaded.
const table = computed(() => {
  if (view.value?.kind !== 'csv') return null
  const rows = parseCSV(text.value)
  const header = parseCSV(view.value.header ?? '')[0] ?? []
  if (start.value === 0 && rows.length) rows.shift() // the header line itself
  return { header, rows }
})

const pretty = computed(() => {
  if (view.value?.kind !== 'json' || start.value !== 0) return null
  try { return JSON.stringify(JSON.parse(text.value), null, 2) } catch { return null }
})

const lineCount = computed(() => text.value.split('\n').filter((l, i, a) => l !== '' || i < a.length - 1).length)

function size(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-[2px]" @click.self="emit('close')">
    <div class="ios-dialog viewer" role="dialog" aria-modal="true" :aria-label="`File ${path}`">
      <div class="head">
        <div class="min-w-0 flex-1">
          <div class="font-mono text-[14px] text-terminal-text-bright truncate">{{ path }}</div>
          <div v-if="view" class="text-[12px] text-terminal-text-dim tabular-nums">
            {{ size(view.size) }} · {{ new Date(view.modified).toLocaleString() }}
            <template v-if="view.kind !== 'binary'"> · {{ start > 0 ? `last ${lineCount} lines` : `${lineCount} lines` }}</template>
          </div>
        </div>
        <label v-if="view && view.kind !== 'binary'" class="follow" title="Show new lines as they are written">
          <input v-model="follow" type="checkbox" /> Follow
        </label>
        <a class="pill pill-plain" :href="api.fileDownloadUrl(path)" download>Download</a>
        <button class="close" aria-label="Close" @click="emit('close')">×</button>
      </div>

      <div ref="body" class="body">
        <div v-if="loading" class="note">Loading…</div>
        <div v-else-if="error" class="settings-error m-3">{{ error }}</div>
        <div v-else-if="view?.kind === 'binary'" class="note">
          This file is not text (an image, an archive or other binary data). Download it to open it.
        </div>
        <template v-else>
          <button v-if="start > 0" class="earlier" @click="loadEarlier">Load earlier lines</button>
          <table v-if="table" class="csv">
            <thead><tr><th class="num">#</th><th v-for="(h, i) in table.header" :key="i">{{ h }}</th></tr></thead>
            <tbody>
              <tr v-for="(r, i) in table.rows" :key="i">
                <td class="num">{{ start === 0 ? i + 1 : '' }}</td>
                <td v-for="(c, j) in r" :key="j">{{ c }}</td>
              </tr>
            </tbody>
          </table>
          <pre v-else class="text">{{ pretty ?? text }}</pre>
          <div v-if="!text" class="note">The file is empty.</div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.viewer { width: min(1100px, 94vw); height: min(760px, 88vh); display: flex; flex-direction: column; overflow: hidden; }
.head { display: flex; align-items: center; gap: 10px; padding: 12px 14px; border-bottom: 1px solid rgb(var(--c-border)); }
.follow { display: flex; align-items: center; gap: 6px; font-size: 13px; color: rgb(var(--c-text)); cursor: pointer; user-select: none; }
.follow input { accent-color: rgb(var(--c-accent)); }
.close { width: 26px; height: 26px; border-radius: 13px; background: rgb(var(--c-surface-alt)); color: rgb(var(--c-text-dim)); font-size: 16px; line-height: 1; }
.body { flex: 1; min-height: 0; overflow: auto; background: rgb(var(--c-bg)); }
.note { padding: 16px; font-size: 13.5px; color: rgb(var(--c-text-dim)); }
.earlier { display: block; margin: 8px auto; padding: 4px 12px; border-radius: 999px; font-size: 12.5px; color: rgb(var(--c-accent)); background: rgb(var(--c-accent) / 0.1); }
.text { margin: 0; padding: 10px 14px; font-family: var(--font-mono, ui-monospace, monospace); font-size: 12.5px; line-height: 1.5; white-space: pre-wrap; overflow-wrap: anywhere; color: rgb(var(--c-text)); }
.csv { border-collapse: collapse; font-size: 12.5px; font-variant-numeric: tabular-nums; min-width: 100%; }
.csv th { position: sticky; top: 0; background: rgb(var(--c-surface)); text-align: left; font-weight: 600; color: rgb(var(--c-text-bright)); padding: 6px 12px; border-bottom: 1px solid rgb(var(--c-border)); white-space: nowrap; }
.csv td { padding: 4px 12px; border-bottom: 1px solid rgb(var(--c-border) / 0.5); white-space: nowrap; color: rgb(var(--c-text)); }
.csv .num { color: rgb(var(--c-text-dim)); text-align: right; width: 1%; }
</style>
