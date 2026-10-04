import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import defaults from '../../../reseller-defaults.json'

export function isPlatformFundingPage(path: string): boolean {
  const pathname = path.split('?')[0]!
  return defaults.customer_hidden_routes.some(route => pathname === route || pathname.startsWith(`${route}/`))
    || defaults.customer_hidden_route_prefixes.some(prefix => pathname.startsWith(prefix))
}

export function useResellerCustomer() {
  const auth = useAuthStore()
  const customer = computed(() => auth.user?.reseller_customer ?? null)
  const isCustomer = computed(() => customer.value !== null)
  const canVisit = (path: string) => !isCustomer.value || !isPlatformFundingPage(path)
  return { customer, isCustomer, canVisit }
}
