// PWM group manifest. Mirrors internal/nodes/pwm/init.go.

import type { Component } from 'vue'
import type { NodeGroupManifest } from '../types'

export const manifest: NodeGroupManifest = {
  name: 'pwm',
  category: 'orbit',
  flowEditors: {
    'pwm-out': () => import('./PwmOutConfig.vue') as Promise<Component>,
  },
}
