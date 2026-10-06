import { computed } from 'vue'
import { useUserView } from '@/composables/useUserView'

export function useResellerCommunications() {
  const view = useUserView()
  const showAnnouncements = computed(() => {
    const customer = view.user?.reseller_customer
    return customer ? customer.announcements_enabled === true : true
  })
  return { showAnnouncements }
}
