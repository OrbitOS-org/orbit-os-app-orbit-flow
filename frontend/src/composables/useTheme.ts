// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

import { ref } from 'vue'

export type Theme = 'dark' | 'light'

// Also read by the inline script in index.html, which applies the saved
// theme before the page paints.
const STORAGE_KEY = 'orbit-flow-theme'

function initialTheme(): Theme {
  return document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light'
}

const theme = ref<Theme>(initialTheme())

function setTheme(next: Theme): void {
  theme.value = next
  if (next === 'dark') {
    document.documentElement.dataset.theme = 'dark'
  } else {
    delete document.documentElement.dataset.theme
  }
  try {
    localStorage.setItem(STORAGE_KEY, next)
  } catch {
    // The choice just does not survive a reload.
  }
}

function toggleTheme(): void {
  setTheme(theme.value === 'dark' ? 'light' : 'dark')
}

/** The editor's colour theme: light by default, dark by choice. */
export function useTheme() {
  return { theme, setTheme, toggleTheme }
}
