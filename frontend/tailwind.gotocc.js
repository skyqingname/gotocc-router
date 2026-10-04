// GoToCC 品牌色：取自在用 logo（public/logo.svg）的靛 → 紫渐变。
// 上游 tailwind.config.js 只用 withGotoCCTheme 包一层，品牌色值都在这里维护。

const brandFrom = '#6366f1'
const brandTo = '#8b5cf6'

// 主色：靛色阶，500 是 logo 起点色。白字放在 600 及更深色阶上，对比度满足 WCAG AA。
const primary = {
  50: '#eef2ff',
  100: '#e0e7ff',
  200: '#c7d2fe',
  300: '#a5b4fc',
  400: '#818cf8',
  500: brandFrom,
  600: '#4f46e5',
  700: '#4338ca',
  800: '#3730a3',
  900: '#312e81',
  950: '#1e1b4b'
}

// 辅色：紫色阶，500 是 logo 终点色；与主色组成品牌渐变。
const accent = {
  50: '#f5f3ff',
  100: '#ede9fe',
  200: '#ddd6fe',
  300: '#c4b5fd',
  400: '#a78bfa',
  500: brandTo,
  600: '#7c3aed',
  700: '#6d28d9',
  800: '#5b21b6',
  900: '#4c1d95',
  950: '#2e1065'
}

const glow = (blur, alpha) => `0 0 ${blur}px rgb(99 102 241 / ${alpha})`

export function withGotoCCTheme(config) {
  const { colors, boxShadow, backgroundImage, keyframes } = config.theme.extend
  colors.primary = primary
  colors.accent = accent
  boxShadow.glow = glow(20, 0.25)
  boxShadow['glow-lg'] = glow(40, 0.35)
  backgroundImage['gradient-primary'] = `linear-gradient(135deg, ${brandFrom} 0%, ${brandTo} 100%)`
  keyframes.glow = {
    '0%': { boxShadow: glow(20, 0.25) },
    '100%': { boxShadow: glow(30, 0.4) }
  }
  return config
}
