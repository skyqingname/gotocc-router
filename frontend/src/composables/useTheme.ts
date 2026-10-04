import { ref } from 'vue'

type ViewTransition = {
  ready: Promise<void>
}

type ViewTransitionDocument = {
  startViewTransition?: (callback: () => void) => ViewTransition
}

// 未保存主题即跟随系统，与 main.ts 启动时的判断一致。
export type ThemeMode = 'light' | 'dark' | 'system'

const themeStorageKey = 'theme'
const themeRippleDuration = 640
const systemDark = window.matchMedia('(prefers-color-scheme: dark)')

function readThemeMode(): ThemeMode {
  const savedTheme = localStorage.getItem(themeStorageKey)
  return savedTheme === 'dark' || savedTheme === 'light' ? savedTheme : 'system'
}

function resolveIsDark(mode: ThemeMode): boolean {
  return mode === 'system' ? systemDark.matches : mode === 'dark'
}

const themeMode = ref<ThemeMode>(readThemeMode())
const isDark = ref(resolveIsDark(themeMode.value))

function applyTheme(nextIsDark: boolean) {
  isDark.value = nextIsDark
  document.documentElement.classList.toggle('dark', nextIsDark)
}

// 跟随系统时，系统深浅色切换即时生效。
systemDark.addEventListener('change', (event) => {
  if (themeMode.value === 'system') applyTheme(event.matches)
})

export function initTheme() {
  applyTheme(resolveIsDark(themeMode.value))
}

function persistTheme(mode: ThemeMode) {
  themeMode.value = mode
  if (mode === 'system') localStorage.removeItem(themeStorageKey)
  else localStorage.setItem(themeStorageKey, mode)
  applyTheme(resolveIsDark(mode))
}

function supportsAnimatedTheme(event?: MouseEvent): event is MouseEvent {
  const viewTransitionDocument = document as unknown as ViewTransitionDocument
  return Boolean(
    event &&
      viewTransitionDocument.startViewTransition &&
      !window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

function getRippleRadius(x: number, y: number): number {
  return Math.hypot(
    Math.max(x, window.innerWidth - x),
    Math.max(y, window.innerHeight - y)
  )
}

function animateThemeRipple(transition: ViewTransition, x: number, y: number) {
  void transition.ready
    .then(() => {
      const endRadius = getRippleRadius(x, y)
      const options: KeyframeAnimationOptions & { pseudoElement: string } = {
        duration: themeRippleDuration,
        easing: 'cubic-bezier(0.65, 0, 0.35, 1)',
        pseudoElement: '::view-transition-new(root)',
      }

      // 用新主题截图做圆形裁剪，让颜色从点击位置像水波一样扩散。
      document.documentElement.animate(
        {
          clipPath: [
            `circle(0px at ${x}px ${y}px)`,
            `circle(${endRadius}px at ${x}px ${y}px)`,
          ],
        },
        options
      )
    })
    .catch(() => undefined)
}

export function setThemeMode(mode: ThemeMode, event?: MouseEvent) {
  if (resolveIsDark(mode) === isDark.value || !supportsAnimatedTheme(event)) {
    persistTheme(mode)
    return
  }

  const { clientX, clientY } = event
  const viewTransitionDocument = document as unknown as ViewTransitionDocument
  const transition = viewTransitionDocument.startViewTransition?.(() => {
    persistTheme(mode)
  })

  if (transition) {
    animateThemeRipple(transition, clientX, clientY)
  } else {
    persistTheme(mode)
  }
}

export function setTheme(nextIsDark: boolean, event?: MouseEvent) {
  if (nextIsDark === isDark.value) return
  setThemeMode(nextIsDark ? 'dark' : 'light', event)
}

export function useTheme() {
  return {
    isDark,
    themeMode,
    setThemeMode,
    setTheme,
    toggleTheme: (event?: MouseEvent) => setTheme(!isDark.value, event),
  }
}
