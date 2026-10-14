<!--
  Copyright (C) 2026 Orbit OS
  Licensed under the GNU Affero General Public License v3.0 or later.
  See LICENSE file for details.

  One Settings area in the style of iOS Settings: sections on the left,
  the open section on the right. "‹ Editor" (or Esc) always leads back.
-->
<script setup lang="ts">
import { computed, onMounted, onUnmounted, provide, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'
import { SETTINGS_ACTION, type SettingsAction } from '@/views/settings/action'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const action = ref<SettingsAction | null>(null)
provide(SETTINGS_ACTION, action)

interface Section { name: string; title: string; color: string; icon: string; show: boolean }
const sections = computed<Section[][]>(() => [
  [
    { name: 'settings-users', title: 'Users', color: '#007aff', show: auth.can('manageUsers'),
      icon: 'M9 11.2a3.2 3.2 0 1 0 0-6.4 3.2 3.2 0 0 0 0 6.4ZM3.5 19a5.5 5.5 0 0 1 11 0M16 5.5a3 3 0 0 1 0 5.6M18 19a5 5 0 0 0-2.5-4.3' },
    { name: 'settings-certs', title: 'Certificates', color: '#34c759', show: auth.can('manageCerts'),
      icon: 'M12 3 5 6v5c0 4.5 3 8 7 10 4-2 7-5.5 7-10V6z' },
  ].filter((s) => s.show),
  [
    { name: 'settings-appearance', title: 'Appearance', color: '#5f45d8', show: true,
      icon: 'M12 20a8 8 0 1 0 0-16 8 8 0 0 0 0 16ZM12 4v16' },
    { name: 'settings-about', title: 'About', color: '#8e8e93', show: true,
      icon: 'M12 20.5a8.5 8.5 0 1 0 0-17 8.5 8.5 0 0 0 0 17ZM12 11v5M12 8h.01' },
  ],
].filter((g) => g.length > 0))

const current = computed(() => sections.value.flat().find((s) => s.name === route.name))

// /settings on its own opens the first section this user can see.
function openDefault() {
  if (route.name === 'settings') {
    router.replace({ name: sections.value[0][0].name })
  }
}
onMounted(openDefault)
watch(() => route.name, openDefault)

function back() {
  router.push('/')
}

function onKey(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  const t = e.target as HTMLElement | null
  if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.tagName === 'SELECT' || t.isContentEditable)) return
  // A dialog open inside a section closes first.
  if (document.querySelector('[role="dialog"], .fixed.inset-0')) return
  back()
}
onMounted(() => window.addEventListener('keydown', onKey))
onUnmounted(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <div class="h-full flex flex-col bg-terminal-bg text-terminal-text">
    <!-- Navigation bar -->
    <div class="ios-bar grid grid-cols-[1fr_auto_1fr] items-center px-3 py-2 border-b border-terminal-border shrink-0">
      <button class="back justify-self-start" aria-label="Back to the editor" @click="back">
        <svg viewBox="0 0 12 20" class="w-[11px] h-[19px]" aria-hidden="true">
          <path d="M10 2 2 10l8 8" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
        Editor
      </button>
      <h1 class="text-[16px] font-semibold text-terminal-text-bright">{{ current?.title ?? 'Settings' }}</h1>
      <div class="justify-self-end">
        <button
          v-if="action"
          :class="action.primary === false
            ? 'bg-terminal-surface-alt/80 text-terminal-text hover:bg-terminal-surface-alt'
            : 'bg-accent text-white hover:bg-accent-dim'"
          class="rounded-full px-4 py-1.5 text-[13px] font-semibold transition-colors"
          @click="action.run()"
        >{{ action.label }}</button>
      </div>
    </div>

    <div class="flex-1 min-h-0 flex">
      <!-- Sections -->
      <nav class="w-[250px] shrink-0 border-r border-terminal-border overflow-y-auto p-3 space-y-4" aria-label="Settings sections">
        <div v-for="(group, gi) in sections" :key="gi" class="settings-group">
          <router-link
            v-for="s in group"
            :key="s.name"
            :to="{ name: s.name }"
            class="settings-item"
            :class="{ on: route.name === s.name }"
            :aria-current="route.name === s.name ? 'page' : undefined"
          >
            <span class="tile" :style="{ background: s.color }">
              <svg viewBox="0 0 24 24" class="w-[17px] h-[17px]" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path :d="s.icon" /></svg>
            </span>
            <span class="flex-1">{{ s.title }}</span>
            <svg class="chev" viewBox="0 0 8 13" aria-hidden="true"><path d="M1.5 1.5 6.5 6.5 1.5 11.5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" /></svg>
          </router-link>
        </div>
      </nav>

      <!-- Open section -->
      <div class="flex-1 min-w-0 overflow-hidden">
        <router-view />
      </div>
    </div>
  </div>
</template>

<style scoped>
.back { display: flex; align-items: center; gap: 4px; color: rgb(var(--c-accent)); font-size: 16px; padding: 2px 4px; }
.back:hover { opacity: 0.75; }
.settings-group { background: rgb(var(--c-surface)); border-radius: 11px; overflow: hidden; }
.settings-item { display: flex; align-items: center; gap: 11px; padding: 8px 12px; font-size: 14px; color: rgb(var(--c-text)); }
.settings-item + .settings-item { border-top: 1px solid rgb(var(--c-border) / 0.7); }
.settings-item:hover:not(.on) { background: rgb(var(--c-surface-alt) / 0.6); }
.settings-item.on { background: rgb(var(--c-accent)); color: #fff; }
.settings-item.on .chev { color: rgb(255 255 255 / 0.75); }
.tile { width: 29px; height: 29px; border-radius: 7px; display: grid; place-items: center; color: #fff; flex: none; }
.chev { width: 8px; height: 13px; color: rgb(var(--c-text-dim) / 0.7); flex: none; }
</style>
