// I2C group manifest. Mirrors internal/nodes/i2c/init.go.

import type { Component } from 'vue'
import type { NodeGroupManifest } from '../types'

export const manifest: NodeGroupManifest = {
  name: 'i2c',
  category: 'orbit',
  flowEditors: {
    'i2c-transfer': () => import('./I2CTransferConfig.vue') as Promise<Component>,
  },
  configEditors: {
    'i2c-bus': () => import('./I2CBusConfig.vue') as Promise<Component>,
  },
}
