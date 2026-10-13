// GPIO group manifest. Mirrors internal/nodes/gpio/init.go.

import type { Component } from 'vue'
import type { NodeGroupManifest } from '../types'

export const manifest: NodeGroupManifest = {
  name: 'gpio',
  category: 'orbit',
  flowEditors: {
    'gpio-read':  () => import('./GpioReadConfig.vue') as Promise<Component>,
    'gpio-write': () => import('./GpioWriteConfig.vue') as Promise<Component>,
  },
  configEditors: {
    'gpio-pin': () => import('./GpioPinConfig.vue') as Promise<Component>,
  },
}
