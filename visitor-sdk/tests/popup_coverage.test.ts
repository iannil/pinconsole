// TS-2 切片补测:ui/popup 全函数覆盖(isURLSchemeAllowed 内部逻辑通过 showPopup 间接验证)。
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { showPopup, removePopup } from '../src/ui/popup';
import type { CommandPopup } from '@pinconsole/proto';

describe('ui/popup', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
  });

  afterEach(() => {
    removePopup();
  });

  it('showPopup 创建 overlay + card', () => {
    const p: CommandPopup = {
      title: 'Hello',
      body: 'World',
      dismissible: true,
    };
    showPopup(p);

    const overlay = document.getElementById('__mm_popup__');
    expect(overlay).not.toBeNull();
    expect(overlay?.tagName).toBe('DIV');
  });

  it('showPopup dismissible=true 含关闭按钮', () => {
    showPopup({
      title: 'T',
      body: 'B',
      dismissible: true,
    });

    const overlay = document.getElementById('__mm_popup__');
    // 应有按钮(dismissible=true)
    const buttons = overlay?.querySelectorAll('button');
    expect(buttons?.length).toBeTruthy();
  });

  it('showPopup 含 action_url 时渲染 action 按钮', () => {
    showPopup({
      title: 'T',
      body: 'B',
      dismissible: false,
      action_label: 'Click',
      action_url: 'https://example.com',
    });

    const overlay = document.getElementById('__mm_popup__');
    // 应有 action 链接
    const actionLink = overlay?.querySelector('a[href]');
    expect(actionLink).not.toBeNull();
    // href 包含 example.com
    expect(actionLink?.getAttribute('href')).toContain('example.com');
  });

  it('showPopup 含 javascript: action_url 被 isURLSchemeAllowed 拒绝(不渲染 a 标签)', () => {
    showPopup({
      title: 'T',
      body: 'B',
      dismissible: false,
      action_label: 'XSS',
      action_url: 'javascript:alert(1)',
    });

    const overlay = document.getElementById('__mm_popup__');
    // javascript: 应被拒绝,无 a 标签或 href 不含恶意 URL
    const actionLink = overlay?.querySelector('a[href]');
    if (actionLink) {
      const href = actionLink.getAttribute('href') || '';
      expect(href).not.toContain('javascript:');
    }
  });

  it('removePopup 清除 DOM', () => {
    showPopup({ title: 'T', body: 'B', dismissible: true });
    expect(document.getElementById('__mm_popup__')).not.toBeNull();

    removePopup();
    expect(document.getElementById('__mm_popup__')).toBeNull();
  });

  it('removePopup 幂等(无 popup 时不抛)', () => {
    expect(() => removePopup()).not.toThrow();
    expect(() => removePopup()).not.toThrow();
  });
});
