// SDK i18n 模块测试：detectLocale / t() / sdkMessages key parity
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { detectLocale, t, sdkMessages } from '../src/ui/i18n';
import type { SdkMessageKey } from '../src/ui/i18n';

describe('detectLocale', () => {
  const origLanguage = navigator.language;

  afterEach(() => {
    Object.defineProperty(navigator, 'language', {
      value: origLanguage,
      configurable: true,
    });
  });

  it('zh 语言返回 zh', () => {
    Object.defineProperty(navigator, 'language', { value: 'zh-CN', configurable: true });
    expect(detectLocale()).toBe('zh');
  });

  it('zh-Hans 返回 zh', () => {
    Object.defineProperty(navigator, 'language', { value: 'zh-Hans', configurable: true });
    expect(detectLocale()).toBe('zh');
  });

  it('en-US 返回 en', () => {
    Object.defineProperty(navigator, 'language', { value: 'en-US', configurable: true });
    expect(detectLocale()).toBe('en');
  });

  it('ja-JP 返回 en（非 zh 均 fallback en）', () => {
    Object.defineProperty(navigator, 'language', { value: 'ja-JP', configurable: true });
    expect(detectLocale()).toBe('en');
  });

  it('navigator undefined 时返回 en', () => {
    const origNav = globalThis.navigator;
    (globalThis as any).navigator = undefined;
    expect(detectLocale()).toBe('en');
    (globalThis as any).navigator = origNav;
  });
});

describe('sdkMessages key parity', () => {
  const zhKeys = Object.keys(sdkMessages.zh).sort();
  const enKeys = Object.keys(sdkMessages.en).sort();

  it('zh ↔ en key 完全一致', () => {
    expect(zhKeys).toEqual(enKeys);
  });

  it('所有值非空', () => {
    for (const [locale, msgs] of Object.entries(sdkMessages)) {
      for (const [key, val] of Object.entries(msgs)) {
        expect(val, `${locale}.${key} 为空`).toBeTruthy();
      }
    }
  });
});

describe('t()', () => {
  it('zh 返回中文文案', () => {
    expect(t('popup_dismiss', 'zh')).toBe('关闭');
  });

  it('en 返回英文文案', () => {
    expect(t('popup_dismiss', 'en')).toBe('Close');
  });

  it('带参数替换 {field}', () => {
    expect(t('cobrowse_fill_toast', 'zh', { field: '邮箱' })).toBe('正在代为填写 邮箱');
    expect(t('cobrowse_fill_toast', 'en', { field: 'email' })).toBe('Filling email on your behalf');
  });

  it('带参数替换 {name}', () => {
    expect(t('cobrowse_assist_hint', 'zh', { name: 'Alice' })).toBe('Alice 正在协助你 · 可见页面操作');
    expect(t('cobrowse_assist_hint', 'en', { name: 'Bob' })).toBe('Bob is assisting you · can see page actions');
  });

  it('不存在 locale fallback 到 en', () => {
    // @ts-expect-error 测试 invalid locale
    expect(t('chat_header', 'fr')).toBe('Support');
  });

  it('不存在 key fallback 到 key 本身', () => {
    expect(t('nonexistent_key' as SdkMessageKey, 'en')).toBe('nonexistent_key');
  });
});
