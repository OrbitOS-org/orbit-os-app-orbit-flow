// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// The Settings area shows each section's main action (e.g. "New user") at
// the top right of its navigation bar. A section registers it here.

import { inject, onUnmounted, type InjectionKey, type Ref } from 'vue'

export interface SettingsAction {
  label: string
  run: () => void
  primary?: boolean
}

export const SETTINGS_ACTION: InjectionKey<Ref<SettingsAction | null>> = Symbol('settings-action')

/** Shows action in the Settings bar while the calling section is open. */
export function useSettingsAction(action: SettingsAction | null): void {
  const slot = inject(SETTINGS_ACTION, null)
  if (!slot) return
  slot.value = action
  onUnmounted(() => {
    if (slot.value === action) slot.value = null
  })
}
