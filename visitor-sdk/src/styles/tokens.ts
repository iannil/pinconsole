// pinconsole visitor-sdk tokens
// 来源:docs/design-system.md §5.1 + §6.2
//
// SDK 注入第三方页面,需要:
//   1. CSS vars 加 --pinconsole-* 前缀(避免宿主冲突)
//   2. 所有注入元素加 data-pinconsole-* 属性(便于识别 / 测试 / 排查)
//   3. 所有样式不继承宿主(all: initial reset + 显式 token)
//
// 用法:
//   import { injectTokenStyles } from './styles/tokens';
//   injectTokenStyles();  // 一次,SDK 初始化时
//   el.style.cssText = 'background: var(--pinconsole-color-accent-default); ...';

const TOKENS_CSS = `
:root {
  /* ===== 字体 family ===== */
  --pinconsole-font-sans: "IBM Plex Sans", "PingFang SC", "Microsoft YaHei",
    "Hiragino Sans GB", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  --pinconsole-font-mono: "IBM Plex Mono", ui-monospace, "SF Mono", monospace;

  /* ===== Type scale ===== */
  --pinconsole-text-xs: 12px;
  --pinconsole-text-sm: 13px;
  --pinconsole-text-base: 14px;
  --pinconsole-text-md: 15px;
  --pinconsole-text-lg: 17px;
  --pinconsole-text-xl: 20px;

  /* ===== 间距 ===== */
  --pinconsole-space-card: 16px;
  --pinconsole-space-component: 12px;
  --pinconsole-space-field: 8px;

  /* ===== 圆角 ===== */
  --pinconsole-radius-sm: 6px;
  --pinconsole-radius-md: 8px;
  --pinconsole-radius-lg: 12px;
  --pinconsole-radius-xl: 16px;
  --pinconsole-radius-pill: 9999px;

  /* ===== 动效 ===== */
  --pinconsole-duration-fast: 120ms;
  --pinconsole-duration-base: 180ms;
  --pinconsole-duration-slow: 240ms;
  --pinconsole-easing: cubic-bezier(0.4, 0, 0.2, 1);

  /* ===== Light mode(默认) ===== */
  --pinconsole-color-bg-canvas: #faf8f5;
  --pinconsole-color-bg-surface: #ffffff;
  --pinconsole-color-bg-subtle: #f5f1ec;
  --pinconsole-color-border-default: #e7e5e4;
  --pinconsole-color-border-strong: #d6d3d1;
  --pinconsole-color-text-primary: #1c1917;
  --pinconsole-color-text-secondary: #57534e;
  --pinconsole-color-text-muted: #78716c;

  --pinconsole-color-accent-default: #0f766e;
  --pinconsole-color-accent-hover: #115e59;
  --pinconsole-color-accent-active: #134e4a;
  --pinconsole-color-accent-subtle: #ccfbf1;
  --pinconsole-color-accent-on: #ffffff;

  --pinconsole-color-success: #15803d;
  --pinconsole-color-warning: #b45309;
  --pinconsole-color-danger: #b91c1c;
  --pinconsole-color-info: #0e7490;

  --pinconsole-shadow-sm: 0 2px 4px rgba(28, 25, 23, 0.05),
    0 1px 2px rgba(28, 25, 23, 0.04);
  --pinconsole-shadow-md: 0 4px 8px rgba(28, 25, 23, 0.06),
    0 2px 4px rgba(28, 25, 23, 0.04);
  --pinconsole-shadow-lg: 0 8px 16px rgba(28, 25, 23, 0.08),
    0 4px 8px rgba(28, 25, 23, 0.05);

  --pinconsole-focus-ring: 0 0 0 3px rgba(15, 118, 110, 0.25);

  --pinconsole-scrim: rgba(28, 25, 23, 0.4);
}

@media (prefers-color-scheme: dark) {
  :root {
    --pinconsole-color-bg-canvas: #1c1917;
    --pinconsole-color-bg-surface: #292524;
    --pinconsole-color-bg-subtle: #44403c;
    --pinconsole-color-border-default: #57534e;
    --pinconsole-color-border-strong: #78716c;
    --pinconsole-color-text-primary: #faf8f5;
    --pinconsole-color-text-secondary: #d6d3d1;
    --pinconsole-color-text-muted: #a8a29e;

    --pinconsole-color-accent-default: #2dd4bf;
    --pinconsole-color-accent-hover: #5eead4;
    --pinconsole-color-accent-active: #14b8a6;
    --pinconsole-color-accent-subtle: #134e4a;
    --pinconsole-color-accent-on: #1c1917;

    --pinconsole-color-success: #4ade80;
    --pinconsole-color-warning: #fbbf24;
    --pinconsole-color-danger: #fca5a5;
    --pinconsole-color-info: #67e8f9;

    --pinconsole-shadow-sm: 0 2px 4px rgba(0, 0, 0, 0.45),
      0 1px 2px rgba(0, 0, 0, 0.35);
    --pinconsole-shadow-md: 0 4px 8px rgba(0, 0, 0, 0.5),
      0 2px 4px rgba(0, 0, 0, 0.4);
    --pinconsole-shadow-lg: 0 8px 16px rgba(0, 0, 0, 0.55),
      0 4px 8px rgba(0, 0, 0, 0.45);

    --pinconsole-focus-ring: 0 0 0 3px rgba(45, 212, 191, 0.35);

    --pinconsole-scrim: rgba(0, 0, 0, 0.6);
  }
}
`;

const STYLE_ID = '__pinconsole_tokens__';

/**
 * 注入 SDK token styles 到页面 :root。
 *
 * 必须在创建任何 SDK 元素(banner / chat / popup)之前调用一次。
 * 重复调用是幂等的(style 标签已存在则跳过)。
 *
 * 注:SDK 用 --pinconsole-* 前缀避免与宿主页面 CSS 变量冲突。
 */
export function injectTokenStyles(): void {
  if (typeof document === 'undefined') return;
  if (document.getElementById(STYLE_ID)) return;

  const style = document.createElement('style');
  style.id = STYLE_ID;
  style.textContent = TOKENS_CSS;
  document.head.appendChild(style);
}

/**
 * 移除 SDK token styles(用于 SDK 卸载 / 测试清理)。
 */
export function removeTokenStyles(): void {
  if (typeof document === 'undefined') return;
  const el = document.getElementById(STYLE_ID);
  if (el && el.parentNode) el.parentNode.removeChild(el);
}
