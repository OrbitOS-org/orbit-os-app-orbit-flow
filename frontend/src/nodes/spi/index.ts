// SPI group manifest. Mirrors internal/nodes/spi/init.go.

import type { Component } from 'vue'
import type { NodeGroupManifest } from '../types'

export const manifest: NodeGroupManifest = {
  name: 'spi',
  category: 'orbit',
  flowEditors: {
    'spi-transfer': () => import('./SpiTransferConfig.vue') as Promise<Component>,
  },
  configEditors: {
    'spi-device': () => import('./SpiDeviceConfig.vue') as Promise<Component>,
  },
}
