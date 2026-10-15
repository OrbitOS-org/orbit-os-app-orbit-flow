import { watch } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

import { useAuthStore } from '@/stores/authStore'
import { basePath } from '@/runtime'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'FlowEditor',
    component: () => import('@/views/FlowEditor.vue'),
  },
  {
    // One Settings area; each section is a child route.
    path: '/settings',
    name: 'settings',
    component: () => import('@/views/SettingsLayout.vue'),
    children: [
      { path: 'users', name: 'settings-users', component: () => import('@/views/UsersView.vue'), meta: { requiresAdmin: true } },
      { path: 'certs', name: 'settings-certs', component: () => import('@/views/CertsView.vue') },
      { path: 'files', name: 'settings-files', component: () => import('@/views/settings/FilesSection.vue') },
      { path: 'appearance', name: 'settings-appearance', component: () => import('@/views/settings/AppearanceSection.vue') },
      { path: 'about', name: 'settings-about', component: () => import('@/views/settings/AboutSection.vue') },
    ],
  },
  // Old addresses.
  { path: '/users', redirect: '/settings/users' },
  { path: '/certs', redirect: '/settings/certs' },
  {
    path: '/design-preview',
    name: 'DesignPreview',
    component: () => import('@/views/DesignPreview.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(basePath),
  routes,
})

// Block admin-only routes for non-admins. The store is populated before
// the very first navigation finishes (App.vue runs auth.init() at mount,
// and route loading is async), but to be safe we redirect to "/" if no
// user or wrong role.
router.beforeEach(async (to) => {
  if (to.meta.requiresAdmin) {
    const auth = useAuthStore()
    // On a fresh page load the session is still being read; wait for it,
    // so reloading an admin page does not bounce to the editor.
    if (auth.loading) {
      await new Promise<void>((resolve) => {
        const stop = watch(() => auth.loading, (l) => { if (!l) { stop(); resolve() } }, { immediate: true })
      })
    }
    if (auth.role !== 'admin') {
      return { path: '/' }
    }
  }
  return true
})

export default router
