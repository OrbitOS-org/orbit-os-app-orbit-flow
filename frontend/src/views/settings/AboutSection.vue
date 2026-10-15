<!--
  Copyright (C) 2026 Orbit OS
  Licensed under the GNU Affero General Public License v3.0 or later.
  See LICENSE file for details.

  About: the Orbit OS device (read through the API) and the app.
-->
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@/composables/useApi'
import { useSettingsAction } from '@/views/settings/action'

// Orbit OS device information (GET /system) and the number of node types.
interface SystemInfo {
  deviceName?: string; os?: string; runtime?: string; api?: string; certified?: boolean
  board?: string; soc?: string; cpu?: string; architecture?: string; memoryMB?: number; linux?: string
  uptimeSeconds?: number; cpuPercent?: number; memoryPercent?: number; temperatureC?: number
}
const system = ref<SystemInfo | null>(null)
const systemError = ref<string | null>(null)
const systemLoading = ref(false)
const nodeTypes = ref<number | null>(null)

function uptime(sec?: number): string {
  if (!sec) return '—'
  const d = Math.floor(sec / 86400), h = Math.floor((sec % 86400) / 3600), m = Math.floor((sec % 3600) / 60)
  return d > 0 ? `${d} d ${h} h` : h > 0 ? `${h} h ${m} min` : `${m} min`
}
const pct = (v?: number) => (v === undefined ? '—' : `${v.toFixed(1)} %`)

const infoGroups = computed(() => {
  const s = system.value
  if (!s) return []
  const dash = (v?: string) => v || '—'
  return [
    { title: 'Orbit OS', rows: [
      ['Device name', dash(s.deviceName)],
      ['System', dash(s.os)],
      ['Runtime', dash(s.runtime)],
      ['API', dash(s.api)],
      ['Certified hardware', s.certified === undefined ? '—' : s.certified ? 'Yes' : 'No'],
    ] },
    { title: 'Hardware', rows: [
      ['Board', dash(s.board)],
      ['SoC', dash(s.soc)],
      ['CPU', dash(s.cpu)],
      ['Architecture', dash(s.architecture)],
      ['Memory', s.memoryMB ? `${s.memoryMB} MB` : '—'],
      ['Linux', dash(s.linux)],
    ] },
    { title: 'Now', rows: [
      ['Uptime', uptime(s.uptimeSeconds)],
      ['CPU usage', pct(s.cpuPercent)],
      ['Memory in use', pct(s.memoryPercent)],
      ['Temperature', s.temperatureC === undefined ? '—' : `${s.temperatureC.toFixed(1)} °C`],
    ] },
  ]
})

async function loadSystem() {
  systemLoading.value = true
  systemError.value = null
  try {
    system.value = await useApi().getSystem()
  } catch (err) {
    systemError.value = err instanceof Error ? err.message : 'Device information could not be read'
  } finally {
    systemLoading.value = false
  }
}

onMounted(async () => {
  loadSystem()
  try {
    nodeTypes.value = (await useApi().getNodes()).length
  } catch {
    // Leave it blank.
  }
})

// Version reported by the app (from its metadata.json).
const appVersion = ref('')
onMounted(async () => {
  try {
    const res = await useApi().getSettings()
    const v = (res.settings as Record<string, unknown> | undefined)?.version
    if (typeof v === 'string') appVersion.value = v
  } catch {
    // Leave the version blank.
  }
})

useSettingsAction({ label: 'Refresh', run: loadSystem, primary: false })
</script>

<template>
  <div class="h-full overflow-y-auto p-6">
    <section class="settings-column space-y-6">
        <p v-if="systemError" class="settings-error">{{ systemError }}</p>

        <div v-for="group in infoGroups" :key="group.title">
          <div class="info-title">{{ group.title }}</div>
          <div class="info-list">
            <div v-for="row in group.rows" :key="row[0]" class="info-row">
              <span class="text-terminal-text">{{ row[0] }}</span>
              <span class="text-terminal-text-dim text-right tabular-nums">{{ row[1] }}</span>
            </div>
          </div>
        </div>

        <div>
          <div class="info-title">Orbit Flow</div>
          <div class="info-list">
            <div class="info-row"><span class="text-terminal-text">Version</span><span class="text-terminal-text-dim">{{ appVersion ? `v${appVersion}` : '—' }}</span></div>
            <div class="info-row"><span class="text-terminal-text">Node types</span><span class="text-terminal-text-dim tabular-nums">{{ nodeTypes ?? '—' }}</span></div>
            <div class="info-row">
              <span class="text-terminal-text">License</span>
              <a href="https://www.gnu.org/licenses/agpl-3.0.html" target="_blank" rel="noopener noreferrer" class="text-accent hover:underline">AGPL-3.0-or-later</a>
            </div>
            <div class="info-row">
              <span class="text-terminal-text">Source code</span>
              <a href="https://github.com/OrbitOS-org/orbit-os-app-orbit-flow" target="_blank" rel="noopener noreferrer" class="text-accent hover:underline text-right">github.com/OrbitOS-org/orbit-os-app-orbit-flow</a>
            </div>
            <div class="info-row">
              <span class="text-terminal-text">Based on</span>
              <span class="text-terminal-text-dim text-right">LOOPZE by Dennis Bleul —
                <a href="https://github.com/loopzedev/loopze-edge" target="_blank" rel="noopener noreferrer" class="text-accent hover:underline">github.com/loopzedev/loopze-edge</a>
              </span>
            </div>
            <div class="info-row"><span class="text-terminal-text">Copyright</span><span class="text-terminal-text-dim text-right">© 2026 Dennis Bleul · © 2026 Orbit OS</span></div>
          </div>
          <p class="settings-note mt-2">
            Orbit Flow is free software released under the GNU Affero General Public
            License v3.0 or later. If you run a modified version on a server and
            let users interact with it over a network, you must offer those users
            the corresponding source code.
          </p>
        </div>
      </section>
  </div>
</template>
