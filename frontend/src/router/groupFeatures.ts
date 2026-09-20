import type { Router } from 'vue-router'

/** Register before installing the router so existing auth/admin guards cover
 * direct initial navigation as well as links from the group management page. */
export function registerGroupFeatureRoutes(router: Router): void {
  router.addRoute({
    path: '/admin/group-features',
    name: 'AdminGroupFeatures',
    component: () => import('@/views/admin/GroupFeaturesView.vue'),
    meta: { requiresAuth: true, requiresAdmin: true, title: 'Group time-window rates' }
  })
  router.addRoute({
    path: '/admin/video-models',
    name: 'AdminVideoModels',
    component: () => import('@/views/admin/VideoModelsView.vue'),
    meta: { requiresAuth: true, requiresAdmin: true, title: 'Video channel models' }
  })
}
