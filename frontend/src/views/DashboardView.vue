<!--
  Copyright (C) 2026 Orbit OS
  Licensed under the GNU Affero General Public License v3.0 or later.
  See LICENSE file for details.

  The dashboard: the ui-* widgets of the running flows, by page and group,
  updated live. Every account can see and use it; editing the flows stays
  in the editor.
-->
<script setup lang="ts">
import { computed, inject, onMounted, onUnmounted, reactive, ref, type Ref } from 'vue'
import AppMark from '@/components/AppMark.vue'
import DashboardWidget from '@/components/dashboard/DashboardWidget.vue'
import { useApi, type DashboardLayout, type DashboardWidget as Widget } from '@/composables/useApi'
import { useAuthStore } from '@/stores/authStore'
import type { DashboardUpdate } from '@/types/events'

type Pt = { t: number; v: number }

const api = useApi()
const auth = useAuthStore()
const onDashboard = inject<(cb: (u: DashboardUpdate) => void) => () => void>('onDashboard')
const wsStatus = inject<Ref<string>>('wsStatus', ref('connected'))

const layout = ref<DashboardLayout>({ pages: [], groups: [], widgets: [] })
const values = reactive<Record<string, unknown>>({})
const series = reactive<Record<string, Record<string, Pt[]>>>({})
const loading = ref(true)
const error = ref('')
const now = ref(Date.now())
const activePage = ref('')

const PAGE_DEFAULT = '__page'
const GROUP_DEFAULT = '__group'

async function load() {
  try {
    const l = await api.getDashboard()
    layout.value = l
    for (const w of l.widgets) {
      values[w.id] = w.value
      if (w.series) series[w.id] = JSON.parse(JSON.stringify(w.series))
    }
    error.value = ''
  } catch (e) {
    error.value = api.isApiError(e) ? e.message : 'Could not load the dashboard.'
  } finally {
    loading.value = false
  }
}

function keepMs(id: string): number {
  const w = layout.value.widgets.find((x) => x.id === id)
  const s = w?.props?.keepSeconds
  return (typeof s === 'number' ? s : 3600) * 1000
}

function applyUpdate(u: DashboardUpdate) {
  if (u.layout) load()
  for (const [id, v] of Object.entries(u.values ?? {})) values[id] = v
  for (const id of u.clears ?? []) series[id] = {}
  for (const [id, byName] of Object.entries(u.points ?? {})) {
    const target = (series[id] ??= {})
    const cutoff = Date.now() - keepMs(id)
    for (const [name, pts] of Object.entries(byName)) {
      const list = (target[name] ?? []).concat(pts).filter((p) => p.t >= cutoff)
      target[name] = list.length > 2000 ? list.slice(-2000) : list
    }
  }
}

let off: (() => void) | undefined
let clock: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  load()
  off = onDashboard?.(applyUpdate)
  clock = setInterval(() => { now.value = Date.now() }, 1000)
})
onUnmounted(() => {
  off?.()
  if (clock) clearInterval(clock)
})

// Pages and groups in order, with a page and a group for widgets whose
// group or page is missing.
const pages = computed(() => {
  const list = [...layout.value.pages]
  const pageIds = new Set(list.map((p) => p.id))
  const groupIds = new Set(layout.value.groups.map((g) => g.id))
  const orphanGroups = layout.value.groups.some((g) => !pageIds.has(g.page))
  const orphanWidgets = layout.value.widgets.some((w) => !groupIds.has(w.group))
  if (!list.length || orphanGroups || orphanWidgets) {
    list.push({ id: PAGE_DEFAULT, name: list.length ? 'Other' : 'Dashboard', order: 1e9 })
  }
  return list
})
const currentPage = computed(() => pages.value.find((p) => p.id === activePage.value) ?? pages.value[0])

const groupsOfPage = computed(() => {
  const pageId = currentPage.value?.id
  const pageIds = new Set(layout.value.pages.map((p) => p.id))
  const groups = layout.value.groups.filter((g) =>
    g.page === pageId || (pageId === PAGE_DEFAULT && !pageIds.has(g.page)))
  const groupIds = new Set(layout.value.groups.map((g) => g.id))
  const result = groups.map((g) => ({ ...g, widgets: layout.value.widgets.filter((w) => w.group === g.id) }))
  if (pageId === PAGE_DEFAULT) {
    const loose = layout.value.widgets.filter((w) => !groupIds.has(w.group))
    if (loose.length) result.push({ id: GROUP_DEFAULT, page: PAGE_DEFAULT, name: '', width: 12, order: 1e9, widgets: loose })
  }
  return result
})

const empty = computed(() => !loading.value && !error.value && layout.value.widgets.length === 0)
const canEdit = computed(() => auth.role !== 'viewer')

async function act(w: Widget, value: unknown) {
  if (w.type === 'ui-switch' || w.type === 'ui-slider') values[w.id] = value
  try {
    await api.dashboardAction(w.id, value)
  } catch (e) {
    error.value = api.isApiError(e) ? e.message : 'The action did not reach the flow.'
  }
}
</script>

<template>
  <div class="dash">
    <header class="top">
      <div class="brand">
        <AppMark class="mark" />
        <b v-if="pages.length <= 1">{{ currentPage?.name || 'Dashboard' }}</b>
      </div>
      <nav v-if="pages.length > 1" class="tabs" aria-label="Pages">
        <button
          v-for="p in pages"
          :key="p.id"
          type="button"
          :class="{ on: p.id === currentPage?.id }"
          @click="activePage = p.id"
        >{{ p.name }}</button>
      </nav>
      <span class="spacer" />
      <span class="live" :class="{ off: wsStatus !== 'connected' }">{{ wsStatus === 'connected' ? 'Live' : 'Reconnecting…' }}</span>
      <router-link v-if="canEdit" to="/" class="edit">Flows</router-link>
    </header>

    <main class="body">
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <p v-if="loading" class="note">Loading…</p>
      <div v-else-if="empty" class="emptystate">
        <h2>No widgets yet</h2>
        <p>
          Add dashboard nodes (Gauge, Chart, Text, Switch, Slider, Button) to a flow, choose a group for each,
          and deploy. They appear here, live.
        </p>
        <router-link v-if="canEdit" to="/" class="btn">Open the editor</router-link>
      </div>

      <div v-else class="groups">
        <section
          v-for="g in groupsOfPage"
          :key="g.id"
          class="group"
          :style="{ '--span': g.width }"
          :aria-label="g.name || 'Widgets'"
        >
          <h3 v-if="g.name">{{ g.name }}</h3>
          <div class="grid">
            <div v-for="w in g.widgets" :key="w.id" class="cell" :style="{ '--span': w.width }">
              <DashboardWidget
                :widget="w"
                :value="values[w.id]"
                :series="series[w.id] ?? {}"
                :now="now"
                @action="act(w, $event)"
              />
            </div>
            <p v-if="!g.widgets.length" class="note">No widgets in this group are running.</p>
          </div>
        </section>
      </div>
    </main>
  </div>
</template>

<style scoped>
.dash { height: 100%; display: flex; flex-direction: column; background: rgb(var(--c-bg)); color: rgb(var(--c-text)); overflow: hidden; }
.top { display: flex; align-items: center; gap: 12px; padding: 10px max(16px, env(safe-area-inset-right, 0px)) 10px max(16px, env(safe-area-inset-left, 0px)); background: rgb(var(--c-surface)); border-bottom: 1px solid rgb(var(--c-border)); }
.brand { display: flex; align-items: center; gap: 8px; min-width: 0; }
.brand b { font-size: 16px; color: rgb(var(--c-text-bright)); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.mark { width: 20px; height: 20px; color: rgb(var(--c-accent)); flex: none; }
.tabs { display: flex; gap: 4px; overflow-x: auto; min-width: 0; scrollbar-width: none; }
.tabs::-webkit-scrollbar { display: none; }
.tabs button { flex: none; border: 0; background: transparent; color: rgb(var(--c-text-dim)); font-size: 13.5px; padding: 6px 12px; border-radius: 999px; cursor: pointer; }
.tabs button.on { background: rgb(var(--c-accent)); color: #fff; font-weight: 600; }
.spacer { flex: 1; min-width: 4px; }
.live { flex: none; font-size: 12.5px; color: rgb(var(--c-success)); display: inline-flex; align-items: center; gap: 6px; }
.live::before { content: ""; width: 7px; height: 7px; border-radius: 50%; background: currentColor; }
.live.off { color: rgb(var(--c-warning)); }
.edit { flex: none; font-size: 13px; font-weight: 600; color: rgb(var(--c-accent)); text-decoration: none; padding: 6px 12px; border-radius: 999px; background: rgb(var(--c-accent) / 0.1); }
.body { flex: 1; overflow: auto; padding: 16px max(16px, env(safe-area-inset-right, 0px)) 24px max(16px, env(safe-area-inset-left, 0px)); }
.groups { display: grid; grid-template-columns: repeat(12, minmax(0, 1fr)); gap: 16px; max-width: 1400px; margin: 0 auto; }
.group { grid-column: span var(--span, 12); display: grid; gap: 8px; align-content: start; min-width: 0; }
.group h3 { margin: 0; padding-left: 4px; font-size: 12px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; color: rgb(var(--c-text-dim)); }
.grid { display: grid; grid-template-columns: repeat(12, minmax(0, 1fr)); gap: 10px; align-items: start; }
.cell { grid-column: span var(--span, 4); min-width: 0; }
.note { color: rgb(var(--c-text-dim)); font-size: 13px; grid-column: 1 / -1; }
.error { color: rgb(var(--c-error)); font-size: 13px; margin: 0 0 12px; }
.emptystate { max-width: 460px; margin: 12vh auto 0; text-align: center; display: grid; gap: 10px; justify-items: center; }
.emptystate h2 { margin: 0; font-size: 20px; color: rgb(var(--c-text-bright)); }
.emptystate p { margin: 0; color: rgb(var(--c-text-dim)); line-height: 1.5; }
.btn { font-size: 14px; font-weight: 600; color: #fff; background: rgb(var(--c-accent)); padding: 9px 16px; border-radius: 10px; text-decoration: none; }
@media (max-width: 760px) {
  .group, .cell { grid-column: span 12; }
}
</style>
