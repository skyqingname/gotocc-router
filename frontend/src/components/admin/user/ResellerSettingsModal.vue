<template>
  <BaseDialog :show="show" :title="tr('站长设置', 'Reseller settings')" width="normal" @close="$emit('close')">
    <div v-if="loading" class="py-10 text-center text-sm text-gray-500">{{ tr('读取设置…', 'Loading…') }}</div>
    <form v-else-if="profile && user" id="reseller-settings" class="space-y-6" @submit.prevent="save">
      <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800"><p class="font-medium">{{ user.username || user.email }}</p><p class="mt-1 text-sm text-gray-500">{{ user.email }}</p></div>
      <label class="flex cursor-pointer items-start gap-3"><input v-model="enabled" type="checkbox" class="mt-1" /><span><span class="font-medium">{{ tr('开放站长中心', 'Enable reseller center') }}</span><span class="mt-1 block text-xs leading-5 text-gray-500">{{ tr('允许此用户邀请客户、管理直属客户、设置客户价格并查看收益。关闭后禁止新的站长注册绑定及管理操作；已生效价格和历史记录保留。', 'Allow this user to invite customers, manage direct customers, set pricing and view earnings. Disabling stops new reseller invitations and management; existing prices and history remain.') }}</span></span></label>
      <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
    </form>
    <p v-else-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
    <template #footer><button type="button" class="btn btn-secondary" @click="$emit('close')">{{ tr('取消', 'Cancel') }}</button><button type="submit" form="reseller-settings" class="btn btn-primary" :disabled="loading || saving || !profile">{{ saving ? tr('保存中…', 'Saving…') : tr('保存站长设置', 'Save settings') }}</button></template>
  </BaseDialog>
</template>
<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { resellerAPI, type ResellerProfile } from '@/api/reseller'
import { useResellerAccess } from '@/composables/useResellerAccess'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
const props = defineProps<{ show: boolean; user: { id: number; username: string; email: string } | null }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'saved'): void }>()
const { locale } = useI18n(); const tr = (zh: string,en: string) => locale.value.startsWith('zh') ? zh : en
const auth = useAuthStore(); const app = useAppStore(); const access = useResellerAccess()
const profile = ref<ResellerProfile | null>(null); const loading = ref(false); const saving = ref(false); const error = ref(''); const enabled = ref(false)
watch(() => [props.show, props.user?.id], async () => {
  if (!props.show || !props.user) return
  loading.value = true; error.value = ''; profile.value = null
  try { const data = await resellerAPI.adminGet(props.user.id); profile.value = data; enabled.value = data.enabled }
  catch (e) { error.value = extractApiErrorMessage(e, tr('读取失败', 'Load failed')) }
  finally { loading.value = false }
}, { immediate: true })
async function save() {
  if (!props.user) return
  saving.value = true; error.value = ''
  try { await resellerAPI.adminSave(props.user.id, enabled.value); if (auth.user?.id === props.user.id) await access.load(true); app.showSuccess(tr('站长设置已保存', 'Reseller settings saved')); emit('saved'); emit('close') }
  catch(e) { error.value = extractApiErrorMessage(e, tr('保存失败', 'Save failed')) }
  finally { saving.value = false }
}
</script>
