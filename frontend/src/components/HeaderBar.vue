<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useFlowStore } from '@/stores/flowStore'
import { useUiStore } from '@/stores/uiStore'
import { useAuthStore } from '@/stores/authStore'
import type { DeployModeType } from '@/types/flow'
import { useTheme } from '@/composables/useTheme'
import AppMark from '@/components/AppMark.vue'
import { useRoute } from 'vue-router'

const flowStore = useFlowStore()
const uiStore = useUiStore()
const auth = useAuthStore()
const { theme, toggleTheme } = useTheme()
const route = useRoute()
const inSettings = computed(() => route.path.startsWith('/settings'))

async function handleLogout(): Promise<void> {
  await auth.logout()
}

const connectionDotColor = computed(() => uiStore.connectionStatusColor)

const connectionLabel = computed(() => {
  switch (uiStore.connectionStatus) {
    case 'connected':   return 'Online'
    case 'connecting':  return 'Connecting'
    case 'disconnected': return 'Offline'
  }
})

const deployModeLabel: Record<DeployModeType, string> = {
  nodes: 'Deploy',
  flows: 'Deploy flows',
  full: 'Full deploy',
  restart: 'Restart',
}

const deployLabel = computed(() => {
  switch (uiStore.deployStatus) {
    case 'deploying': return 'Deploying'
    case 'deployed':  return 'Deployed'
    case 'failed':    return 'Failed'
    default:          return deployModeLabel[flowStore.deployMode]
  }
})

const deployDisabled = computed(() => uiStore.deployStatus === 'deploying')

const showDeployMenu = ref(false)

const deployModes: { value: DeployModeType; label: string; description: string }[] = [
  { value: 'nodes', label: 'Modified Nodes', description: 'Only changed nodes' },
  { value: 'flows', label: 'Modified Flows', description: 'Entire changed flows' },
  { value: 'full',  label: 'Full Deploy',    description: 'Restart all nodes' },
  { value: 'restart', label: 'Restart',      description: 'Full engine restart' },
]

function handleClickOutside(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (!target.closest('.deploy-menu-container')) {
    showDeployMenu.value = false
  }
}

onMounted(() => document.addEventListener('click', handleClickOutside))
onUnmounted(() => document.removeEventListener('click', handleClickOutside))

async function handleDeploy(): Promise<void> {
  showDeployMenu.value = false
  await flowStore.deploy()
}

function selectMode(mode: DeployModeType): void {
  flowStore.setDeployMode(mode)
  showDeployMenu.value = false
}
</script>

<template>
  <header
    class="ios-bar flex items-center justify-between h-12 px-4 border-b border-terminal-border select-none shrink-0 relative z-30"
  >
    <!-- Left: Logo / App Name -->
    <div class="flex items-center gap-3">
      <button
        class="text-terminal-text-dim hover:text-terminal-text transition-colors duration-100 p-1 -ml-1 rounded hover:bg-terminal-surface-alt"
        title="Toggle node palette"
        @click="uiStore.toggleLeftPanel()"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>

      <div class="flex items-center gap-2">
        <AppMark class="w-[18px] h-[18px] text-accent" />
        <span class="text-terminal-text-bright text-[15px] font-semibold">Orbit Flow</span>
      </div>

      <span class="text-terminal-text-dim text-[10px] tracking-wider hidden sm:inline font-medium uppercase">
        Flow Automation
      </span>
    </div>

    <!-- Right: Actions & Status -->
    <div class="flex items-center gap-2">
      <!-- Connection status -->
      <div
        class="flex items-center gap-1.5 px-2 py-1 rounded"
        :title="`Status: ${connectionLabel}`"
      >
        <span
          class="block w-2 h-2 rounded-full"
          :style="{ backgroundColor: connectionDotColor }"
        />
        <span class="text-[10px] text-terminal-text-dim tracking-wider hidden sm:inline font-medium">
          {{ connectionLabel }}
        </span>
      </div>

      <!-- Separator -->
      <div class="w-px h-5 bg-terminal-border mx-1" />

      <!-- Settings link -->
      <router-link
        to="/settings"
        class="p-1.5 rounded transition-all duration-100"
        :class="inSettings ? 'text-accent bg-accent/10' : 'text-terminal-text-dim hover:text-terminal-text hover:bg-terminal-surface-alt'"
        title="Settings"
        aria-label="Settings"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 15a3 3 0 100-6 3 3 0 000 6z" />
          <path stroke-linecap="round" stroke-linejoin="round" d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 01-2.83 2.83l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z" />
        </svg>
      </router-link>

      <!-- Light / dark theme -->
      <button
        class="p-1.5 rounded text-terminal-text-dim hover:text-terminal-text hover:bg-terminal-surface-alt transition-all duration-100"
        :title="theme === 'dark' ? 'Light theme' : 'Dark theme'"
        :aria-label="theme === 'dark' ? 'Switch to the light theme' : 'Switch to the dark theme'"
        @click="toggleTheme()"
      >
        <svg v-if="theme === 'dark'" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
          <circle cx="12" cy="12" r="4" />
          <path stroke-linecap="round" d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
        </svg>
        <svg v-else xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
          <path stroke-linecap="round" stroke-linejoin="round" d="M21 12.8A9 9 0 1111.2 3a7 7 0 009.8 9.8z" />
        </svg>
      </button>

      <!-- Toggle Terminal Log -->
      <button
        class="p-1.5 rounded transition-all duration-100"
        :class="uiStore.logsPanelOpen
          ? 'text-accent bg-accent/10'
          : 'text-terminal-text-dim hover:text-terminal-text hover:bg-terminal-surface-alt'"
        title="Log"
        @click="uiStore.toggleLogsPanel()"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
          <path stroke-linecap="round" stroke-linejoin="round" d="M5 4h14a2 2 0 012 2v12a2 2 0 01-2 2H5a2 2 0 01-2-2V6a2 2 0 012-2z" />
          <path stroke-linecap="round" stroke-linejoin="round" d="M7 11l3-3-3-3M11 14h4" />
        </svg>
      </button>

      <!-- Toggle Properties panel -->
      <button
        class="p-1.5 rounded transition-all duration-100"
        :class="uiStore.propertiesPanelOpen
          ? 'text-accent bg-accent/10'
          : 'text-terminal-text-dim hover:text-terminal-text hover:bg-terminal-surface-alt'"
        title="Toggle properties"
        @click="uiStore.togglePropertiesPanel()"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
          <path stroke-linecap="round" stroke-linejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
        </svg>
      </button>

      <!-- Toggle Debug panel -->
      <button
        class="p-1.5 rounded transition-all duration-100"
        :class="uiStore.infoPanelOpen
          ? 'text-accent bg-accent/10'
          : 'text-terminal-text-dim hover:text-terminal-text hover:bg-terminal-surface-alt'"
        title="Toggle information"
        @click="uiStore.toggleInfoPanel()"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </button>

      <!-- Separator -->
      <div class="w-px h-5 bg-terminal-border mx-1" />

      <!-- User identity + logout -->
      <div v-if="auth.user" class="flex items-center gap-1.5">
        <span class="text-[10px] text-terminal-text-dim font-mono hidden sm:inline" :title="auth.user.role">
          {{ auth.user.username }}
        </span>
        <button
          class="p-1.5 rounded text-terminal-text-dim hover:text-terminal-text hover:bg-terminal-surface-alt transition-all duration-100"
          title="Sign out"
          @click="handleLogout"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
            <path stroke-linecap="round" stroke-linejoin="round" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
          </svg>
        </button>
      </div>

      <!-- Read-only badge for viewers (replaces the deploy button) -->
      <div
        v-if="auth.user && !auth.can('deploy')"
        class="px-2 py-1 border border-terminal-border rounded-sm text-[10px] uppercase tracking-wider text-terminal-text-dim font-mono"
        title="You can view flows but not modify or deploy them."
      >
        Read-only
      </div>

      <!-- Separator -->
      <div v-if="auth.user" class="w-px h-5 bg-terminal-border mx-1" />

      <!-- Deploy split-button -->
      <div v-if="auth.can('deploy')" class="deploy-menu-container relative">
        <div class="flex items-center">
          <!-- Main deploy button -->
          <button
            class="h-8 flex items-center gap-1.5 pl-3.5 pr-3 text-[13px] font-semibold transition-all duration-150 rounded-l-full"
            :disabled="deployDisabled"
            :class="{
              'opacity-50 cursor-not-allowed': deployDisabled,
              'bg-status-success text-white': uiStore.deployStatus === 'deployed',
              'bg-status-error text-white': uiStore.deployStatus === 'failed',
              'bg-accent text-white hover:bg-accent-dim': uiStore.deployStatus !== 'deployed' && uiStore.deployStatus !== 'failed',
            }"
            title="Deploy flows"
            @click="handleDeploy"
          >
            <!-- Spinning icon while deploying -->
            <svg
              v-if="uiStore.deployStatus === 'deploying'"
              xmlns="http://www.w3.org/2000/svg"
              class="w-3.5 h-3.5 animate-spin"
              fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"
            >
              <path stroke-linecap="round" d="M12 2v4m0 12v4m-7-7H3m18 0h-2M6.34 6.34L4.93 4.93m12.73 12.73l1.41 1.41M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41" />
            </svg>
            <!-- Checkmark when deployed -->
            <svg
              v-else-if="uiStore.deployStatus === 'deployed'"
              xmlns="http://www.w3.org/2000/svg"
              class="w-3.5 h-3.5"
              fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M5 12l5 5L20 7" />
            </svg>
            <!-- X when failed -->
            <svg
              v-else-if="uiStore.deployStatus === 'failed'"
              xmlns="http://www.w3.org/2000/svg"
              class="w-3.5 h-3.5"
              fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
            <!-- Upload/deploy icon (idle) -->
            <svg
              v-else
              xmlns="http://www.w3.org/2000/svg"
              class="w-3.5 h-3.5"
              fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M7 16l-4-4m0 0l4-4m-4 4h18M17 8l4 4m0 0l-4 4" />
            </svg>
            <span>{{ deployLabel }}</span>
            <span
              v-if="flowStore.changedFlowCount > 0 && uiStore.deployStatus !== 'deploying' && uiStore.deployStatus !== 'deployed' && uiStore.deployStatus !== 'failed'"
              class="rounded-full bg-white/25 px-1.5 text-[11.5px] font-medium tabular-nums"
              :title="`${flowStore.changedFlowCount} flow(s) with changes not deployed`"
            >{{ flowStore.changedFlowCount }} {{ flowStore.changedFlowCount === 1 ? 'flow' : 'flows' }}</span>
          </button>

          <!-- Dropdown toggle -->
          <button
            class="h-8 flex items-center pl-1.5 pr-2.5 transition-all duration-150 rounded-r-full border-l border-l-white/25"
            :disabled="deployDisabled"
            :class="{
              'opacity-50 cursor-not-allowed': deployDisabled,
              'bg-status-success text-white': uiStore.deployStatus === 'deployed',
              'bg-status-error text-white': uiStore.deployStatus === 'failed',
              'bg-accent text-white hover:bg-accent-dim': uiStore.deployStatus !== 'deployed' && uiStore.deployStatus !== 'failed',
            }"
            title="Select deploy mode"
            @click.stop="showDeployMenu = !showDeployMenu"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </button>
        </div>

        <!-- Dropdown menu -->
        <div
          v-if="showDeployMenu"
          class="ios-menu absolute right-0 top-full mt-2 w-60 z-50 overflow-hidden"
        >
          <button
            v-for="mode in deployModes"
            :key="mode.value"
            class="w-full flex items-center gap-2.5 px-3 py-2.5 text-left text-xs hover:bg-terminal-surface-alt transition-colors duration-100"
            @click="selectMode(mode.value)"
          >
            <span
              class="w-2 h-2 rounded-full border border-terminal-text-dim shrink-0 transition-colors"
              :class="{ 'bg-accent border-accent': flowStore.deployMode === mode.value }"
            />
            <div>
              <div class="text-terminal-text font-medium">{{ mode.label }}</div>
              <div class="text-terminal-text-dim text-[10px] mt-0.5">{{ mode.description }}</div>
            </div>
          </button>
        </div>
      </div>
    </div>
    <!-- iOS-style banner after a deploy -->
    <Transition name="banner">
      <div
        v-if="uiStore.deployStatus === 'deployed' || uiStore.deployStatus === 'failed'"
        class="ios-banner"
        role="status"
      >
        <span class="mark" :class="uiStore.deployStatus === 'failed' ? 'bad' : ''">{{ uiStore.deployStatus === 'failed' ? '!' : '✓' }}</span>
        <span>{{ uiStore.deployStatus === 'failed' ? 'Deploy failed — see the log' : 'Deployed' }}</span>
      </div>
    </Transition>
  </header>
</template>

<style scoped>
.ios-banner {
  position: fixed; left: 50%; top: 12px; transform: translateX(-50%); z-index: 60;
  display: flex; align-items: center; gap: 10px; padding: 9px 16px 9px 10px;
  background: var(--bar-bg); backdrop-filter: saturate(180%) blur(20px); -webkit-backdrop-filter: saturate(180%) blur(20px);
  border: 1px solid rgb(var(--c-border)); border-radius: 16px; font-size: 13px; font-weight: 500;
  color: rgb(var(--c-text)); box-shadow: 0 8px 24px rgb(0 0 0 / 0.18);
}
.ios-banner .mark {
  width: 22px; height: 22px; border-radius: 7px; display: grid; place-items: center;
  background: rgb(var(--c-success)); color: #fff; font-size: 13px; font-weight: 700;
}
.ios-banner .mark.bad { background: rgb(var(--c-error)); }
.banner-enter-active, .banner-leave-active { transition: transform 0.35s cubic-bezier(0.2, 0.9, 0.3, 1.2), opacity 0.25s; }
.banner-enter-from, .banner-leave-to { transform: translate(-50%, -140%); opacity: 0; }
@media (prefers-reduced-motion: reduce) { .banner-enter-active, .banner-leave-active { transition: none; } }
</style>
