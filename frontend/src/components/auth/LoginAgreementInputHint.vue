<template>
  <div
    class="relative"
    @focusin="active = true"
    @focusout="handleFocusOut"
    @click="active = true"
    @keydown.esc="active = false"
  >
    <slot :hint-id="visible ? hintId : undefined" />
    <div
      v-if="visible"
      :id="hintId"
      role="status"
      class="absolute left-4 top-full z-20 mt-2 max-w-[calc(100%-2rem)] rounded-md border border-gray-500 bg-white shadow-lg"
    >
      <span
        aria-hidden="true"
        class="absolute -top-[7px] left-5 h-3 w-3 rotate-45 border-l border-t border-gray-500 bg-white"
      />
      <button
        type="button"
        aria-haspopup="dialog"
        class="relative flex items-center gap-2.5 rounded-md px-3 py-2.5 text-left text-sm font-medium text-gray-900 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500"
        @click.stop="emit('open')"
      >
        <span
          aria-hidden="true"
          class="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-sm bg-orange-600 text-lg font-bold leading-none text-white"
        >!</span>
        <span>{{ t('legal.loginAgreementPrompt.inputHint') }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ blocked: boolean }>()
const emit = defineEmits<{ open: [] }>()
const { t } = useI18n()
const hintId = useId()
const active = ref(false)
const visible = computed(() => props.blocked && active.value)

function handleFocusOut(event: FocusEvent): void {
  active.value = (event.currentTarget as HTMLElement).contains(event.relatedTarget as Node | null)
}
</script>
