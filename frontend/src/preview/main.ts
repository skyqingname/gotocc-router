import { createApp } from 'vue'
import { createPinia } from 'pinia'
import i18n, { loadLocaleMessages } from '@/i18n'
import App from './PriorityPreview.vue'
import { installPreviewAdapter } from './fixtures'
import '@/style.css'
import '@/styles/gotocc.css'

async function start() {
  installPreviewAdapter()
  await loadLocaleMessages('zh')
  i18n.global.locale.value = 'zh'
  document.documentElement.classList.add('dark')
  createApp(App).use(createPinia()).use(i18n).mount('#app')
}
void start()
