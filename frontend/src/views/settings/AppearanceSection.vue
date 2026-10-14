<!--
  Copyright (C) 2026 Orbit OS
  Licensed under the GNU Affero General Public License v3.0 or later.
  See LICENSE file for details.
-->
<script setup lang="ts">
import { useTheme } from '@/composables/useTheme'

const { theme, setTheme } = useTheme()
</script>

<template>
  <div class="h-full overflow-y-auto p-6">
    <div class="space-y-2">
      <div class="info-title">Theme</div>
      <div class="info-list">
        <div class="info-row">
          <span class="text-terminal-text">Editor theme</span>
          <div class="seg" role="group" aria-label="Editor theme">
            <button :aria-pressed="theme === 'light'" @click="setTheme('light')">Light</button>
            <button :aria-pressed="theme === 'dark'" @click="setTheme('dark')">Dark</button>
          </div>
        </div>
      </div>
      <p class="px-3 text-xs text-terminal-text-dim">Saved in this browser. The button in the header switches it too.</p>
    </div>
  </div>
</template>

<style scoped>
.seg { display: inline-flex; gap: 2px; padding: 2px; border-radius: 9px; background: rgb(var(--c-surface-alt) / 0.8); }
.seg button { padding: 4px 14px; border-radius: 7px; font-size: 13px; color: rgb(var(--c-text)); }
.seg button[aria-pressed="true"] { background: rgb(var(--c-surface)); font-weight: 600; box-shadow: 0 1px 3px rgb(0 0 0 / 0.12); }
:root[data-theme="dark"] .seg button[aria-pressed="true"] { background: rgb(99 99 102); }
</style>
