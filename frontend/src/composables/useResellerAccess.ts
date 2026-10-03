import { ref } from 'vue'
import { resellerAPI } from '@/api/reseller'
import { useAuthStore } from '@/stores/auth'
const enabled = ref(false)
let loadedUser: number | null = null
export function useResellerAccess() {
  async function load(force = false): Promise<boolean> {
    const auth = useAuthStore()
    const userID = auth.user?.id ?? null
    if (!userID) { enabled.value = false; loadedUser = null; return false }
    if (!force && loadedUser === userID && auth.user?.reseller_customer !== undefined) return enabled.value
    const result = await resellerAPI.access()
    if (auth.user?.id === userID) { enabled.value = result.enabled; loadedUser = userID
      auth.user.reseller_customer = result.customer
      if (result.customer) { auth.user.balance = result.customer.credit_balance; auth.user.frozen_balance = result.customer.frozen_credit } }
    return result.enabled
  }
  return { enabled, load }
}
