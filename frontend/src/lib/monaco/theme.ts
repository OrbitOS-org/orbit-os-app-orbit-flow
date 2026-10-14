// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// Code editor themes. They follow the page theme (useTheme): the surface
// colours are read from the page's variables, so there is one palette.

import { watch } from 'vue'
import * as monaco from 'monaco-editor'
import { useTheme, type Theme } from '@/composables/useTheme'

const DARK = 'orbit-flow-dark'
const LIGHT = 'orbit-flow-light'

/** Reads a "--c-name" variable ("R G B") as "#rrggbb". */
function pageColor(name: string): string {
  const raw = getComputedStyle(document.documentElement).getPropertyValue(`--c-${name}`)
  const parts = raw.trim().split(/\s+/).map(Number)
  if (parts.length !== 3 || parts.some(Number.isNaN)) return '#808080'
  return '#' + parts.map((n) => n.toString(16).padStart(2, '0')).join('')
}

const SYNTAX: Record<Theme, Record<string, string>> = {
  dark: {
    keyword: 'ff7b72', string: 'a5d6ff', number: '79c0ff', type: 'ffa657',
    function: 'd2a8ff', property: '79c0ff', operator: 'ff7b72',
  },
  light: {
    keyword: 'cf222e', string: '0a3069', number: '0550ae', type: '953800',
    function: '8250df', property: '0550ae', operator: 'cf222e',
  },
}

function define(name: string, mode: Theme): void {
  const c = pageColor
  const s = SYNTAX[mode]
  const text = c('text').slice(1)
  monaco.editor.defineTheme(name, {
    base: mode === 'dark' ? 'vs-dark' : 'vs',
    inherit: true,
    rules: [
      { token: 'comment', foreground: c('text-dim').slice(1), fontStyle: 'italic' },
      { token: 'keyword', foreground: s.keyword },
      { token: 'string', foreground: s.string },
      { token: 'number', foreground: s.number },
      { token: 'regexp', foreground: s.string },
      { token: 'type', foreground: s.type },
      { token: 'class', foreground: s.type },
      { token: 'function', foreground: s.function },
      { token: 'variable', foreground: text },
      { token: 'constant', foreground: s.number },
      { token: 'parameter', foreground: text },
      { token: 'property', foreground: s.property },
      { token: 'operator', foreground: s.operator },
      { token: 'delimiter', foreground: text },
      { token: 'delimiter.bracket', foreground: text },
    ],
    colors: {
      'editor.background': c('bg'),
      'editor.foreground': c('text'),
      'editor.lineHighlightBackground': c('surface') + '40',
      'editor.selectionBackground': c('accent') + '33',
      'editor.inactiveSelectionBackground': c('accent') + '1a',
      'editorCursor.foreground': c('accent'),
      'editorLineNumber.foreground': c('border-light'),
      'editorLineNumber.activeForeground': c('text-dim'),
      'editorGutter.background': c('bg'),
      'editorWidget.background': c('surface'),
      'editorWidget.border': c('border'),
      'editorSuggestWidget.background': c('surface'),
      'editorSuggestWidget.border': c('border'),
      'editorSuggestWidget.selectedBackground': c('accent') + '33',
      'editorSuggestWidget.highlightForeground': c('accent'),
      'editorHoverWidget.background': c('surface'),
      'editorHoverWidget.border': c('border'),
      'input.background': c('bg'),
      'input.border': c('border'),
      'input.foreground': c('text'),
      'focusBorder': c('accent'),
      'list.hoverBackground': c('surface'),
      'list.activeSelectionBackground': c('accent') + '33',
      'scrollbarSlider.background': c('border-light') + '33',
      'scrollbarSlider.hoverBackground': c('border-light') + '66',
      'scrollbarSlider.activeBackground': c('border-light') + 'aa',
      'editorBracketMatch.background': c('accent') + '33',
      'editorBracketMatch.border': c('accent') + '66',
      'editorIndentGuide.background': c('surface-alt'),
      'editorIndentGuide.activeBackground': c('border'),
    },
  })
}

const defined = new Set<Theme>()

function themeName(mode: Theme): string {
  if (!defined.has(mode)) {
    // The variables hold the palette of the theme on the page, so a theme
    // can only be defined while the page shows it.
    define(mode === 'dark' ? DARK : LIGHT, mode)
    defined.add(mode)
  }
  return mode === 'dark' ? DARK : LIGHT
}

let watching = false

/**
 * Name of the code editor theme for the current page theme. Every open
 * editor follows when the page theme changes.
 */
export function editorTheme(): string {
  const { theme } = useTheme()
  if (!watching) {
    watching = true
    watch(theme, (mode) => monaco.editor.setTheme(themeName(mode)), { flush: 'post' })
  }
  return themeName(theme.value)
}
