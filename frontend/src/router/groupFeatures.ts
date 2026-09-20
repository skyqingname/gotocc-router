import type { Router } from 'vue-router'

/** Register before installing the router so its existing auth/admin guards also
 * cover a direct initial navigation to this page. */
export function registerGroupFeatureRoutes(router: Router): void {
  router.addRoute({
    path: '/admin/group-features',
    name: 'AdminGroupFeatures',
    component: () => import('@/views/admin/GroupFeaturesView.vue'),
    meta: { requiresAuth: true, requiresAdmin: true, title: 'Group time-window rates' }
  })
}
