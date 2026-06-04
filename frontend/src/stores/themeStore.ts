// 单主题模式 - 只使用 void 主题
export type Theme = 'void'

export const useThemeStore = {
  getState: () => ({ theme: 'void' as const })
}

export function initTheme() {
  // 不需要初始化，默认主题已在 globals.css :root 中定义
}
