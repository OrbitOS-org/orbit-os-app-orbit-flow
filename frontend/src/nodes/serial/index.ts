// Serial group manifest. Mirrors internal/nodes/serial/init.go.

import type { Component } from 'vue'
import type { NodeGroupManifest } from '../types'

export const manifest: NodeGroupManifest = {
  name: 'serial',
  category: 'orbit',
  flowEditors: {
    'serial-in': () => import('./SerialInConfig.vue') as Promise<Component>,
    'serial-out': () => import('./SerialOutConfig.vue') as Promise<Component>,
    'serial-request': () => import('./SerialRequestConfig.vue') as Promise<Component>,
  },
  configEditors: {
    'serial-port': () => import('./SerialPortConfig.vue') as Promise<Component>,
  },
}
