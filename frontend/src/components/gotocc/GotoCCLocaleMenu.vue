<!-- Derived from TokenFlux/TokenRouter ab684f817 (v0.7.3) frontend/src/components/common/LocaleSwitcher.vue (header text variant). Copyright (c) 2026 Wesley Liddick & TokenFlux. LGPL-3.0-or-later. GoToCC adaptation. -->
<template>
  <div ref="rootRef" class="relative">
    <!-- 顶栏入口：语言动效图标加当前语言名（窄顶栏只留图标），展开时保持主色 -->
    <button type="button" class="header-nav-link" aria-haspopup="menu" :aria-expanded="open" :data-tip="t('consoleTips.header.language')" data-tip-side="bottom" :aria-label="currentLocale.name" @click="open = !open">
      <LanguagesIcon /><span class="header-nav-label">{{ currentLocale.name }}</span>
    </button>

    <Transition name="dropdown-fade">
      <div v-if="open" class="dropdown right-0 mt-2 w-44 origin-top-right py-0" role="menu">
        <div class="menu-section">
          <button
            v-for="option in availableLocales"
            :key="option.code"
            type="button"
            role="menuitemradio"
            :aria-checked="option.code === locale"
            class="menu-item"
            @click="select(option.code)"
          >
            <span class="text-base leading-none">{{ option.flag }}</span>
            <span class="min-w-0 flex-1">{{ option.name }}</span>
            <Icon v-if="option.code === locale" name="check" size="sm" class="shrink-0 text-primary-600 dark:text-primary-300" />
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { LanguagesIcon } from '@/components/gotocc/navIcons'
import { availableLocales, setLocale } from '@/i18n'

// 顶栏语言入口：与公告、文档、模型广场一样是动效图标加文字，点开是与用户菜单同样的内缩菜单；切换沿用 i18n 的 setLocale。
const { locale, t } = useI18n()
const open = ref(false)
const rootRef = ref<HTMLElement | null>(null)
const currentLocale = computed(() => availableLocales.find((option) => option.code === locale.value)!)

async function select(code: string) {
  open.value = false
  await setLocale(code)
}

function handleClickOutside(event: MouseEvent) {
  if (rootRef.value && !rootRef.value.contains(event.target as Node)) open.value = false
}

onMounted(() => document.addEventListener('click', handleClickOutside))
onBeforeUnmount(() => document.removeEventListener('click', handleClickOutside))
</script>
