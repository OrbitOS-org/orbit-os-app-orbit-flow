<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@/composables/useApi'

const settings = ref({
  flowFile: 'flows.json',
  userDir: 'data',
  httpAdminRoot: '/',
  httpNodeRoot: '/',
  debugMaxLength: 1000,
  mqttReconnectTime: 5000,
  serialReconnectTime: 5000,
})

const saved = ref(false)

function handleSave(): void {
  // TODO: POST settings to /api/v1/settings
  saved.value = true
  setTimeout(() => {
    saved.value = false
  }, 2000)
}

function handleReset(): void {
  // TODO: reload settings from backend
}

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
</script>

<template>
  <div class="h-full w-full overflow-y-auto bg-terminal-bg p-6 font-mono text-terminal-text">
    <!-- Page Header -->
    <div class="mb-6">
      <div class="flex items-center gap-3 mb-2">
        <router-link
          to="/"
          class="text-terminal-text-dim hover:text-accent text-xs uppercase tracking-wider transition-colors duration-100"
        >
          ← Back
        </router-link>
      </div>
      <h1 class="text-accent text-lg font-bold uppercase tracking-widest ">
        ⚙ Settings
      </h1>
      <p class="text-terminal-text-dim text-xs mt-1">
        Runtime configuration for the flow engine
      </p>
      <div class="border-b border-terminal-border mt-3"></div>
    </div>

    <!-- Settings Form -->
    <div class="max-w-2xl space-y-6">
      <!-- Section: General -->
      <section>
        <h2 class="text-xs text-terminal-text-dim uppercase tracking-widest mb-3 flex items-center gap-2">
          <span class="w-2 h-2 bg-accent inline-block"></span>
          General
        </h2>

        <div class="space-y-3 pl-4 border-l border-terminal-border">
          <div class="flex flex-col gap-1">
            <label class="text-[10px] text-terminal-text-dim uppercase tracking-wider">
              Flow File
            </label>
            <input
              v-model="settings.flowFile"
              type="text"
              class="terminal-input w-full text-xs"
            />
          </div>

          <div class="flex flex-col gap-1">
            <label class="text-[10px] text-terminal-text-dim uppercase tracking-wider">
              User Directory
            </label>
            <input
              v-model="settings.userDir"
              type="text"
              class="terminal-input w-full text-xs"
            />
          </div>
        </div>
      </section>

      <!-- Section: HTTP -->
      <section>
        <h2 class="text-xs text-terminal-text-dim uppercase tracking-widest mb-3 flex items-center gap-2">
          <span class="w-2 h-2 bg-accent inline-block"></span>
          HTTP
        </h2>

        <div class="space-y-3 pl-4 border-l border-terminal-border">
          <div class="flex flex-col gap-1">
            <label class="text-[10px] text-terminal-text-dim uppercase tracking-wider">
              Admin Root Path
            </label>
            <input
              v-model="settings.httpAdminRoot"
              type="text"
              class="terminal-input w-full text-xs"
            />
          </div>

          <div class="flex flex-col gap-1">
            <label class="text-[10px] text-terminal-text-dim uppercase tracking-wider">
              Node Root Path
            </label>
            <input
              v-model="settings.httpNodeRoot"
              type="text"
              class="terminal-input w-full text-xs"
            />
          </div>
        </div>
      </section>

      <!-- Section: Debug -->
      <section>
        <h2 class="text-xs text-terminal-text-dim uppercase tracking-widest mb-3 flex items-center gap-2">
          <span class="w-2 h-2 bg-accent inline-block"></span>
          Debug
        </h2>

        <div class="space-y-3 pl-4 border-l border-terminal-border">
          <div class="flex flex-col gap-1">
            <label class="text-[10px] text-terminal-text-dim uppercase tracking-wider">
              Max Debug Message Length
            </label>
            <input
              v-model.number="settings.debugMaxLength"
              type="number"
              min="100"
              max="10000"
              class="terminal-input w-full text-xs"
            />
          </div>
        </div>
      </section>

      <!-- Section: Protocols -->
      <section>
        <h2 class="text-xs text-terminal-text-dim uppercase tracking-widest mb-3 flex items-center gap-2">
          <span class="w-2 h-2 bg-accent inline-block"></span>
          Protocols
        </h2>

        <div class="space-y-3 pl-4 border-l border-terminal-border">
          <div class="flex flex-col gap-1">
            <label class="text-[10px] text-terminal-text-dim uppercase tracking-wider">
              MQTT Reconnect Time (ms)
            </label>
            <input
              v-model.number="settings.mqttReconnectTime"
              type="number"
              min="1000"
              class="terminal-input w-full text-xs"
            />
          </div>

          <div class="flex flex-col gap-1">
            <label class="text-[10px] text-terminal-text-dim uppercase tracking-wider">
              Serial Reconnect Time (ms)
            </label>
            <input
              v-model.number="settings.serialReconnectTime"
              type="number"
              min="1000"
              class="terminal-input w-full text-xs"
            />
          </div>
        </div>
      </section>

      <!-- Actions -->
      <div class="flex items-center gap-3 pt-4 border-t border-terminal-border">
        <button
          class="terminal-btn-primary text-xs uppercase tracking-wider"
          @click="handleSave"
        >
          Save Settings
        </button>
        <button
          class="terminal-btn text-xs uppercase tracking-wider"
          @click="handleReset"
        >
          Reset
        </button>

        <span
          v-if="saved"
          class="text-green-400 text-xs uppercase tracking-wider ml-2 transition-opacity"
        >
          ✓ Saved
        </span>
      </div>

      <!-- System Info: the Orbit OS device, then the app -->
      <section class="mt-8 pt-4 border-t border-terminal-border space-y-5 max-w-xl">
        <div class="flex items-center justify-between">
          <h2 class="text-[15px] font-semibold text-terminal-text-bright">System Information</h2>
          <button class="terminal-btn" :disabled="systemLoading" @click="loadSystem">Refresh</button>
        </div>
        <p v-if="systemError" class="text-xs text-status-error">{{ systemError }}</p>

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
            <div class="info-row"><span class="text-terminal-text">Built with</span><span class="text-terminal-text-dim">Go and Vue</span></div>
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
          <p class="mt-2 px-1 text-[11px] text-terminal-text-dim leading-relaxed">
            Orbit Flow is free software released under the GNU Affero General Public
            License v3.0 or later. If you run a modified version on a server and
            let users interact with it over a network, you must offer those users
            the corresponding source code.
          </p>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.info-title { font-size: 11px; font-weight: 500; letter-spacing: 0.04em; text-transform: uppercase; color: rgb(var(--c-text-dim)); padding: 0 12px 6px; }
.info-list { background: rgb(var(--c-surface)); border-radius: 12px; overflow: hidden; border: 1px solid rgb(var(--c-border) / 0.6); }
.info-row { display: flex; align-items: baseline; justify-content: space-between; gap: 16px; padding: 9px 14px; font-size: 13px; }
.info-row > :first-child { white-space: nowrap; flex: none; }
.info-row > :last-child { min-width: 0; overflow-wrap: anywhere; }
.info-row + .info-row { border-top: 1px solid rgb(var(--c-border) / 0.7); }
</style>
