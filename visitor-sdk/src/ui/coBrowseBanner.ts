// 1l-privacy-gdpr:co-browsing 接管横幅
// 当运营接管访客时显示,提供透明度(GDPR Art.22 自动化决策)。
// 访客可点退出按钮立即释放控制权。
//
// Phase 5:警告黄 + 红 → Calm cream strip + teal accent(design-system.md §5.3)
// - bg-subtle(cream)+ border-default 下边框
// - 内联 SVG eye icon(去 🔔 emoji)
// - 退出按钮用 text-link 风(代替红色,降威胁感)
// - 保留 __mm_cobrowse_banner__ ID + onExit callback
import { t } from './i18n';
import { getCachedWidgetConfig } from '../widget-config';

const BANNER_ID = '__mm_cobrowse_banner__';

export interface CoBrowseBannerOptions {
  operatorName?: string; // 运营名(可选,服务端目前不传)
  onExit: () => void;    // 访客点退出按钮触发
}

// 内联 SVG eye icon(无 emoji 跨平台一致)
const EYE_SVG = `<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 256 256" fill="currentColor" aria-hidden="true"><path d="M247.31,124.76c-.35-.79-8.82-19.58-27.65-38.41C194.57,61.26,162.88,48,128,48S61.43,61.26,36.34,86.35C17.51,105.18,9,124,8.69,124.76a8,8,0,0,0,0,6.5c.35.79,8.82,19.57,27.65,38.4C61.43,194.74,93.12,208,128,208s66.57-13.26,91.66-38.34c18.83-18.83,27.3-37.61,27.65-38.4A8,8,0,0,0,247.31,124.76ZM128,192c-30.78,0-57.67-11.19-79.93-33.25A132.45,132.45,0,0,1,25,128a132.27,132.27,0,0,1,23.07-30.75C70.33,75.19,97.22,64,128,64s57.67,11.19,79.93,33.25A132.45,132.45,0,0,1,231,128C223.94,141.83,192.43,192,128,192Zm0-112a48,48,0,1,0,48,48A48.05,48.05,0,0,0,128,80Zm0,80a32,32,0,1,1,32-32A32,32,0,0,1,128,160Z"/></svg>`;

export function showCoBrowseBanner(opts: CoBrowseBannerOptions): void {
  removeCoBrowseBanner();

  // 读取 widgetConfig 文案,fallback 到 i18n 硬编码
  const cached = getCachedWidgetConfig();
  const cfg = cached.cobrowse_banner;
  const operatorLabel = opts.operatorName ?? cfg?.operator_label ?? t('cobrowse_operator_label');
  const hintText = cfg?.assist_hint ?? t('cobrowse_assist_hint');
  const hintWithName = hintText.replace('{name}', operatorLabel);
  const exitLabel = cfg?.exit_label ?? t('cobrowse_exit');

  const banner = document.createElement('div');
  banner.id = BANNER_ID;
  banner.setAttribute('data-pinconsole', 'cobrowse-banner');
  banner.setAttribute('role', 'status');
  banner.style.cssText = [
    'position: fixed',
    'top: 0',
    'left: 0',
    'width: 100%',
    `background: var(--pinconsole-color-bg-subtle, #f5f1ec)`,
    `color: var(--pinconsole-color-text-primary, #1c1917)`,
    'padding: 10px 24px',
    'z-index: 999999',
    'display: flex',
    'align-items: center',
    'justify-content: space-between',
    'gap: 12px',
    `font-family: var(--pinconsole-font-sans, system-ui, sans-serif)`,
    'font-size: 14px',
    `border-bottom: 1px solid var(--pinconsole-color-border-default, #e7e5e4)`,
    'box-shadow: 0 2px 4px rgba(28, 25, 23, 0.05)',
  ].join(';');

  // === 左侧 hint(support icon + text) ===
  const label = document.createElement('span');
  label.style.cssText = 'display: inline-flex; align-items: center; gap: 8px';

  // icon wrapper(accent 色,内嵌 SVG)
  const iconWrap = document.createElement('span');
  iconWrap.setAttribute('aria-hidden', 'true');
  iconWrap.style.cssText =
    'display: inline-flex; align-items: center; color: var(--pinconsole-color-accent-default, #0f766e); flex-shrink: 0';
  iconWrap.innerHTML = EYE_SVG;

  const text = document.createElement('span');
  text.textContent = hintWithName;

  label.appendChild(iconWrap);
  label.appendChild(text);
  banner.appendChild(label);

  // === 右侧退出按钮(text-link 风,非红色) ===
  const exitBtn = document.createElement('button');
  exitBtn.textContent = exitLabel;
  exitBtn.setAttribute('data-pinconsole', 'cobrowse-exit');
  exitBtn.style.cssText = [
    'padding: 4px 12px',
    'font-size: 13px',
    'font-weight: 500',
    `color: var(--pinconsole-color-text-secondary, #57534e)`,
    'background: transparent',
    'border: none',
    'border-radius: 6px',
    'cursor: pointer',
    'transition: background 0.12s ease-out, color 0.12s ease-out',
    `font-family: var(--pinconsole-font-sans, system-ui, sans-serif)`,
    'flex-shrink: 0',
  ].join(';');
  exitBtn.onmouseenter = () => {
    exitBtn.style.background = 'var(--pinconsole-color-bg-surface, #fff)';
    exitBtn.style.color = 'var(--pinconsole-color-text-primary, #1c1917)';
  };
  exitBtn.onmouseleave = () => {
    exitBtn.style.background = 'transparent';
    exitBtn.style.color = 'var(--pinconsole-color-text-secondary, #57534e)';
  };
  exitBtn.onclick = () => {
    opts.onExit();
  };
  banner.appendChild(exitBtn);

  document.body.prepend(banner);

  // 推下页面顶部内容,避免 banner 遮挡
  document.body.style.paddingTop = '48px';
}

export function removeCoBrowseBanner(): void {
  const el = document.getElementById(BANNER_ID);
  if (el && el.parentNode) el.parentNode.removeChild(el);
  document.body.style.paddingTop = '';
}
