<!--
  Copyright (C) 2026 Orbit OS
  Licensed under the GNU Affero General Public License v3.0 or later.
  See LICENSE file for details.

  Settings → Files: the files the flows read and write. On Orbit OS there is
  no terminal into the app's folder, so this is where users see, download
  and delete them. Everyone who can open the editor downloads; only
  administrators delete.
-->
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useApi, type AppFile, type ApiError } from '@/composables/useApi'
import { useAuthStore } from '@/stores/authStore'
import { useSettingsAction } from '@/views/settings/action'

const api = useApi()
const auth = useAuthStore()

const files = ref<AppFile[]>([])
const used = ref(0)
const free = ref(0)
const loading = ref(true)
const error = ref<string | null>(null)

const canDelete = computed(() => auth.can('manageUsers'))
const onlyFiles = computed(() => files.value.filter((f) => !f.isDir))

async function load() {
  loading.value = true
  error.value = null
  try {
    const res = await api.listFiles()
    files.value = res.files
    used.value = res.used
    free.value = res.free
  } catch (err) {
    error.value = api.isApiError(err) ? (err as ApiError).message : 'Could not list the files.'
  } finally {
    loading.value = false
  }
}
onMounted(load)
useSettingsAction({ label: 'Refresh', run: load, primary: false })

const deleting = ref<AppFile | null>(null)
const deleteError = ref<string | null>(null)
async function confirmDelete() {
  if (!deleting.value) return
  deleteError.value = null
  try {
    await api.deleteFile(deleting.value.path)
    deleting.value = null
    await load()
  } catch (err) {
    deleteError.value = api.isApiError(err) ? (err as ApiError).message : 'Could not delete the file.'
  }
}

function size(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`
  return `${(n / 1024 / 1024 / 1024).toFixed(1)} GB`
}

function when(iso: string): string {
  const d = new Date(iso)
  const today = new Date()
  const time = d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  if (d.toDateString() === today.toDateString()) return `today ${time}`
  return `${d.toLocaleDateString([], { day: 'numeric', month: 'short' })} ${time}`
}

// Share of the disk the files use, for the bar (at least a sliver when used).
const share = computed(() => {
  const total = used.value + free.value
  if (!total || !used.value) return 0
  return Math.max(1, Math.round((used.value / total) * 100))
})
</script>

<template>
  <div class="h-full flex flex-col bg-terminal-bg text-terminal-text overflow-hidden">
    <div class="flex-1 overflow-auto p-6">
      <div class="settings-column space-y-2">
        <div v-if="error" class="settings-error">{{ error }}</div>
        <div v-if="loading" class="px-3 text-[15px] text-terminal-text-dim">Loading…</div>

        <div v-else-if="onlyFiles.length === 0" class="settings-empty">
          <svg viewBox="0 0 24 24" class="w-9 h-9 text-terminal-text-dim/60" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round" aria-hidden="true">
            <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
          </svg>
          <div class="text-[16px] font-semibold text-terminal-text-bright">No files yet</div>
          <p class="text-[13.5px] text-terminal-text-dim max-w-md">
            Files written by Write File appear here, for example a CSV log. Try the example "Log readings to CSV".
          </p>
        </div>

        <template v-else>
          <div class="info-title">The app's files</div>
          <div class="info-list">
            <div v-for="f in onlyFiles" :key="f.path" class="info-row">
              <div class="min-w-0 file-name">
                <div class="font-mono text-[13.5px] text-terminal-text truncate">{{ f.path }}</div>
                <div class="text-[12.5px] text-terminal-text-dim tabular-nums">{{ size(f.size) }} · {{ when(f.modified) }}</div>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <a class="pill pill-plain" :href="api.fileDownloadUrl(f.path)" download>Download</a>
                <button v-if="canDelete" class="pill pill-bad" @click="deleting = f; deleteError = null">Delete</button>
              </div>
            </div>
          </div>
        </template>

        <div v-if="!loading" class="usage">
          <span class="tabular-nums">{{ size(used) }} used</span>
          <span class="bar" aria-hidden="true"><i :style="{ width: share + '%' }" /></span>
          <span class="tabular-nums">{{ size(free) }} free on the device</span>
        </div>
        <p class="settings-note">
          The flows read and write only these files: Write File, Read File and Watch cannot reach anything else
          on the device. They stay on the device; download a copy here.
        </p>
      </div>
    </div>

    <div
      v-if="deleting"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-[2px]"
      @click.self="deleting = null"
    >
      <div class="ios-dialog w-[340px] max-w-[90vw] p-5 text-center" role="dialog" aria-modal="true" aria-labelledby="del-file-title">
        <h2 id="del-file-title">Delete “{{ deleting.path }}”?</h2>
        <p class="mt-2 text-[13px] text-terminal-text-dim">
          The file is removed from the device. A flow that writes to it creates it again.
        </p>
        <div v-if="deleteError" class="settings-error mt-3">{{ deleteError }}</div>
        <div class="mt-4 grid gap-2">
          <button class="pill py-2 bg-status-error/10 text-status-error font-semibold" @click="confirmDelete">Delete</button>
          <button class="pill pill-plain py-2" @click="deleting = null">Cancel</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* The file name may be long: let it shrink (info-row keeps the first child whole). */
.info-row > .file-name { flex: 1 1 auto; white-space: normal; }
.usage { display: flex; align-items: center; gap: 10px; padding: 8px 14px 0; font-size: 12.5px; color: rgb(var(--c-text-dim)); }
.usage .bar { flex: 1; height: 6px; border-radius: 3px; background: rgb(var(--c-surface-alt)); overflow: hidden; }
.usage .bar i { display: block; height: 100%; background: rgb(var(--c-accent)); }
</style>
