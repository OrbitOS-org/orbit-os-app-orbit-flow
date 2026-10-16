// Dashboard group manifest. Mirrors internal/nodes/dashboard/init.go.

import type { Component } from 'vue'
import type { NodeGroupManifest } from '../types'

const widget = () => import('./WidgetConfig.vue') as Promise<Component>

export const manifest: NodeGroupManifest = {
  name: 'dashboard',
  category: 'orbit',
  flowEditors: {
    'ui-gauge': widget,
    'ui-chart': widget,
    'ui-text': widget,
    'ui-switch': widget,
    'ui-slider': widget,
    'ui-button': widget,
  },
  configEditors: {
    'ui-page': () => import('./PageConfig.vue') as Promise<Component>,
    'ui-group': () => import('./GroupConfig.vue') as Promise<Component>,
  },
}
