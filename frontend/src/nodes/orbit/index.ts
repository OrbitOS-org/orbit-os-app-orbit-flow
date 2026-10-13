// OrbitOS group manifest. Mirrors internal/nodes/orbit/init.go.
//
// Nodes here call an OrbitOS SDK manager directly (via internal/orbitsdk on
// the backend) rather than doing generic network/filesystem I/O, so they
// only work when this app is actually running on OrbitOS.

import type { Component } from 'vue'
import type { NodeGroupManifest } from '../types'

export const manifest: NodeGroupManifest = {
  name: 'orbit',
  category: 'orbit',
  flowEditors: {
    'mobile-notify':    () => import('./MobileNotifyConfig.vue') as Promise<Component>,
    'system-metrics':   () => import('./SystemMetricsConfig.vue') as Promise<Component>,
    'device-identity':  () => import('./DeviceIdentityConfig.vue') as Promise<Component>,
  },
}
