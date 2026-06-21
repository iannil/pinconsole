// 弹窗渲染器（1g + 1k P0-8 URL scheme 白名单）
// SDK 接收 show_popup command 后用结构化 JSON + 预设 HTML 模板渲染
//
// Phase 5:Calm Modal 升级(design-system.md §5.5)
// - scrim backdrop + blur
// - radius-xl 卡 + shadow-lg
// - accent-default action button(代替蓝按钮)
// - 内联 SVG X icon(代替文本 close)
// - 保留 __mm_popup__ ID + URL scheme 白名单 + dismissible 逻辑
import type { CommandPopup } from '@pinconsole/proto';
import { t } from './i18n';

const POPUP_ID = '__mm_popup__';

// 内联 SVG X icon
const CLOSE_ICON_SVG = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 256 256" fill="currentColor" aria-hidden="true"><path d="M205.66,194.34a8,8,0,0,1-11.32,11.32L128,139.31,61.66,205.66a8,8,0,0,1-11.32-11.32L116.69,128,50.34,61.66A8,8,0,0,1,61.66,50.34L128,116.69l66.34-66.35a8,8,0,0,1,11.32,11.32L139.31,128Z"/></svg>`;

// isURLSchemeAllowed 1k P0-8:双重防御。
// 后端 command.go 也校验,这里再加一层防止绕过(直接构造 envelope)。
// 只允许 http/https;空字符串、protocol-relative (//host)、相对路径 (/path 或 page.html) 允许。
function isURLSchemeAllowed(rawURL: string): boolean {
  if (!rawURL) return true;
  const lower = rawURL.toLowerCase();

  // 显式拒绝危险 scheme
  for (const bad of ['javascript:', 'data:', 'vbscript:', 'file:', 'about:']) {
    if (lower.startsWith(bad)) return false;
  }

  // 显式允许 http/https
  if (lower.startsWith('http://') || lower.startsWith('https://')) return true;

  // 允许 protocol-relative
  if (rawURL.startsWith('//')) return true;

  // 检测是否含 scheme:":" 出现在第一个 "/" 之前
  const firstColon = rawURL.indexOf(':');
  const firstSlash = rawURL.indexOf('/');
  if (firstColon === -1 || (firstSlash !== -1 && firstSlash < firstColon)) {
    return true; // 无 scheme,相对路径
  }

  return false; // 含非 http/https scheme
}

export function showPopup(p: CommandPopup): void {
  removePopup();
  const overlay = document.createElement('div');
  overlay.id = POPUP_ID;
  overlay.setAttribute('data-pinconsole', 'popup-overlay');
  overlay.setAttribute('role', 'dialog');
  overlay.setAttribute('aria-modal', 'true');
  overlay.style.cssText = [
    'position: fixed', 'top: 0', 'left: 0', 'width: 100%', 'height: 100%',
    `background: var(--pinconsole-scrim, rgba(28, 25, 23, 0.4))`,
    'backdrop-filter: blur(4px)',
    '-webkit-backdrop-filter: blur(4px)',
    'z-index: 999998',
    'display: flex', 'align-items: center', 'justify-content: center',
    'padding: 16px',
    `font-family: var(--pinconsole-font-sans, system-ui, sans-serif)`,
  ].join(';');

  const card = document.createElement('div');
  card.setAttribute('data-pinconsole', 'popup-card');
  card.style.cssText = [
    `background: var(--pinconsole-color-bg-surface, #fff)`,
    `color: var(--pinconsole-color-text-primary, #1c1917)`,
    `border: 1px solid var(--pinconsole-color-border-default, #e7e5e4)`,
    'border-radius: 16px',
    'padding: 24px 28px',
    'max-width: 440px',
    'width: 100%',
    'box-shadow: 0 8px 16px rgba(28, 25, 23, 0.08), 0 4px 8px rgba(28, 25, 23, 0.05)',
    'position: relative',
    'display: flex',
    'flex-direction: column',
    'gap: 12px',
  ].join(';');

  // close button(右上角 X) —— 仅 dismissible 时显示
  if (p.dismissible) {
    const closeBtn = document.createElement('button');
    closeBtn.setAttribute('data-pinconsole', 'popup-close');
    closeBtn.setAttribute('aria-label', t('popup_dismiss'));
    closeBtn.style.cssText = [
      'position: absolute',
      'top: 10px',
      'right: 10px',
      'display: inline-flex',
      'align-items: center',
      'justify-content: center',
      'width: 28px',
      'height: 28px',
      'padding: 0',
      'border: none',
      'border-radius: 6px',
      'background: transparent',
      `color: var(--pinconsole-color-text-muted, #78716c)`,
      'cursor: pointer',
      'transition: background 0.12s ease-out, color 0.12s ease-out',
    ].join(';');
    closeBtn.innerHTML = CLOSE_ICON_SVG;
    closeBtn.onmouseenter = () => {
      closeBtn.style.background = 'var(--pinconsole-color-bg-subtle, #f5f1ec)';
      closeBtn.style.color = 'var(--pinconsole-color-text-primary, #1c1917)';
    };
    closeBtn.onmouseleave = () => {
      closeBtn.style.background = 'transparent';
      closeBtn.style.color = 'var(--pinconsole-color-text-muted, #78716c)';
    };
    closeBtn.onclick = removePopup;
    card.appendChild(closeBtn);
  }

  // title(用 textContent 防 XSS)
  if (p.title) {
    const h = document.createElement('h3');
    h.textContent = p.title;
    h.style.cssText = 'margin: 0;padding-right: 32px;font-size: 17px;font-weight: 600;color: var(--pinconsole-color-text-primary, #1c1917)';
    card.appendChild(h);
  }
  // body
  if (p.body) {
    const body = document.createElement('p');
    body.textContent = p.body;
    body.style.cssText = 'margin: 0;font-size: 14px;line-height: 1.55;color: var(--pinconsole-color-text-secondary, #57534e)';
    card.appendChild(body);
  }
  // action button(1k P0-8:URL scheme 白名单)
  const actionRow = document.createElement('div');
  actionRow.style.cssText = 'display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px';
  if (p.action_label && p.action_url && isURLSchemeAllowed(p.action_url)) {
    const btn = document.createElement('a');
    btn.textContent = p.action_label;
    btn.href = p.action_url;
    btn.setAttribute('data-pinconsole', 'popup-action');
    btn.style.cssText = [
      'display: inline-flex',
      'align-items: center',
      'padding: 8px 20px',
      'font-size: 14px',
      'font-weight: 600',
      `color: var(--pinconsole-color-accent-on, #fff)`,
      `background: var(--pinconsole-color-accent-default, #0f766e)`,
      `border: 1px solid var(--pinconsole-color-accent-default, #0f766e)`,
      'border-radius: 8px',
      'text-decoration: none',
      'cursor: pointer',
      'transition: background 0.12s ease-out',
      `font-family: var(--pinconsole-font-sans, system-ui, sans-serif)`,
    ].join(';');
    btn.onmouseenter = () => {
      btn.style.background = 'var(--pinconsole-color-accent-hover, #115e59)';
    };
    btn.onmouseleave = () => {
      btn.style.background = 'var(--pinconsole-color-accent-default, #0f766e)';
    };
    actionRow.appendChild(btn);
  }
  // dismiss button(text-link 风,与 X 关闭互为补充)
  if (p.dismissible) {
    const dismiss = document.createElement('button');
    dismiss.textContent = t('popup_dismiss');
    dismiss.style.cssText = [
      'padding: 8px 16px',
      'font-size: 14px',
      'font-weight: 500',
      `color: var(--pinconsole-color-text-secondary, #57534e)`,
      'background: transparent',
      `border: 1px solid var(--pinconsole-color-border-default, #e7e5e4)`,
      'border-radius: 8px',
      'cursor: pointer',
      'transition: background 0.12s ease-out',
      `font-family: var(--pinconsole-font-sans, system-ui, sans-serif)`,
    ].join(';');
    dismiss.onmouseenter = () => {
      dismiss.style.background = 'var(--pinconsole-color-bg-subtle, #f5f1ec)';
    };
    dismiss.onmouseleave = () => {
      dismiss.style.background = 'transparent';
    };
    dismiss.onclick = removePopup;
    actionRow.appendChild(dismiss);
  }

  if (actionRow.children.length > 0) {
    card.appendChild(actionRow);
  }

  overlay.appendChild(card);
  // 点击遮罩关闭(仅 dismissible 时)
  if (p.dismissible) {
    overlay.onclick = (e) => { if (e.target === overlay) removePopup(); };
  }
  document.body.appendChild(overlay);
}

export function removePopup(): void {
  const el = document.getElementById(POPUP_ID);
  if (el && el.parentNode) el.parentNode.removeChild(el);
}
