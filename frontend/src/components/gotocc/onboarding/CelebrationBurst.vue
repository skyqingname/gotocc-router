<template>
  <div class="burst" aria-hidden="true">
    <span v-for="index in count" :key="index" class="burst-dot" :style="{ '--i': index, '--n': count }"></span>
    <span class="burst-ring"></span>
    <span class="burst-medal"><Icon name="check" size="lg" :stroke-width="2.6" /></span>
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
import consoleGuide from '../../../../../console-guide.json'

// 全部完成时的庆祝：勋章弹出，一圈小光点向外散开一次后淡出。减少动态效果时只显示勋章。
const count = consoleGuide.onboarding.celebration_particles
</script>

<style scoped>
.burst { position: relative; display: grid; height: 96px; width: 96px; place-items: center; }
.burst-medal {
  position: relative; z-index: 1; display: grid; height: 64px; width: 64px; place-items: center;
  border-radius: 9999px; color: white;
  background: linear-gradient(135deg, theme('colors.primary.500'), theme('colors.accent.500'));
  box-shadow: 0 10px 30px -10px rgb(99 102 241 / 60%), inset 0 1px 0 rgb(255 255 255 / 35%);
  animation: burst-pop 620ms var(--motion-ease-fluid) both;
}
.burst-ring {
  position: absolute; inset: 16px; border-radius: 9999px; border: 2px solid rgb(139 92 246 / 45%);
  animation: burst-ring 900ms var(--motion-ease-fluid) 120ms both;
}
.burst-dot {
  position: absolute; left: 50%; top: 50%; height: 6px; width: 6px; margin: -3px 0 0 -3px; border-radius: 9999px;
  background: theme('colors.primary.400');
  opacity: 0;
  --angle: calc(360deg / var(--n) * var(--i));
  animation: burst-dot 900ms var(--motion-ease-fluid) 160ms both;
}
.burst-dot:nth-child(3n) { background: theme('colors.accent.400'); }
.burst-dot:nth-child(4n) { background: theme('colors.amber.400'); }
@keyframes burst-pop {
  0% { transform: scale(.3); opacity: 0; }
  60% { transform: scale(1.08); opacity: 1; }
  100% { transform: scale(1); }
}
@keyframes burst-ring {
  from { transform: scale(.6); opacity: .9; }
  to { transform: scale(1.5); opacity: 0; }
}
@keyframes burst-dot {
  0% { transform: rotate(var(--angle)) translateY(-14px) scale(1); opacity: 0; }
  20% { opacity: 1; }
  100% { transform: rotate(var(--angle)) translateY(-48px) scale(.4); opacity: 0; }
}
@media (prefers-reduced-motion: reduce) {
  .burst-medal { animation: none; }
  .burst-ring, .burst-dot { display: none; }
}
</style>
