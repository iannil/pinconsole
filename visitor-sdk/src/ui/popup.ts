// 弹窗渲染器（1g）
// SDK 接收 show_popup command 后用结构化 JSON + 预设 HTML 模板渲染
import type { CommandPopup } from '../proto/command';

const POPUP_ID = '__mm_popup__';

export function showPopup(p: CommandPopup): void {
  removePopup();
  const overlay = document.createElement('div');
  overlay.id = POPUP_ID;
  overlay.style.cssText = [
    'position: fixed', 'top: 0', 'left: 0', 'width: 100%', 'height: 100%',
    'background: rgba(0,0,0,0.4)', 'z-index: 999998',
    'display: flex', 'align-items: center', 'justify-content: center',
  ].join(';');

  const card = document.createElement('div');
  card.style.cssText = [
    'background: #fff', 'border-radius: 8px', 'padding: 24px 32px',
    'max-width: 400px', 'box-shadow: 0 8px 32px rgba(0,0,0,0.2)',
    'font-family: system-ui, sans-serif',
  ].join(';');

  // title（用 textContent 防 XSS）
  if (p.title) {
    const h = document.createElement('h3');
    h.textContent = p.title;
    h.style.cssText = 'margin: 0 0 12px;font-size: 18px;color: #303133';
    card.appendChild(h);
  }
  // body
  if (p.body) {
    const body = document.createElement('p');
    body.textContent = p.body;
    body.style.cssText = 'margin: 0 0 16px;font-size: 14px;color: #606266;line-height: 1.6';
    card.appendChild(body);
  }
  // action button
  if (p.action_label && p.action_url) {
    const btn = document.createElement('a');
    btn.textContent = p.action_label;
    btn.href = p.action_url;
    btn.style.cssText = 'display: inline-block;padding: 8px 20px;background: #409eff;color: #fff;border-radius: 4px;text-decoration: none;font-size: 14px;margin-right: 8px';
    card.appendChild(btn);
  }
  // dismiss button
  if (p.dismissible) {
    const dismiss = document.createElement('button');
    dismiss.textContent = '关闭';
    dismiss.style.cssText = 'padding: 8px 20px;background: #f5f7fa;color: #606266;border: 1px solid #dcdfe6;border-radius: 4px;font-size: 14px;cursor: pointer';
    dismiss.onclick = removePopup;
    card.appendChild(dismiss);
  }

  overlay.appendChild(card);
  // 点击遮罩关闭（仅 dismissible 时）
  if (p.dismissible) {
    overlay.onclick = (e) => { if (e.target === overlay) removePopup(); };
  }
  document.body.appendChild(overlay);
}

export function removePopup(): void {
  const el = document.getElementById(POPUP_ID);
  if (el && el.parentNode) el.parentNode.removeChild(el);
}
