// i18n key parity test: 验证 en-US ↔ zh-CN key 结构完全一致
import { describe, it, expect } from 'vitest';
import enUS from '../src/i18n/en-US';
import zhCN from '../src/i18n/zh-CN';

type I18nRecord = Record<string, unknown>;

/** 递归提取所有叶子 key 路径（如 'app.title'） */
function flattenKeys(obj: I18nRecord, prefix = ''): string[] {
  const keys: string[] = [];
  for (const [k, v] of Object.entries(obj)) {
    const path = prefix ? `${prefix}.${k}` : k;
    if (v !== null && typeof v === 'object') {
      keys.push(...flattenKeys(v as I18nRecord, path));
    } else {
      keys.push(path);
    }
  }
  return keys.sort();
}

/** 递归检查所有值非空字符串 */
function checkNonEmpty(obj: I18nRecord, prefix = ''): string[] {
  const issues: string[] = [];
  for (const [k, v] of Object.entries(obj)) {
    const path = prefix ? `${prefix}.${k}` : k;
    if (v !== null && typeof v === 'object') {
      issues.push(...checkNonEmpty(v as I18nRecord, path));
    } else if (typeof v !== 'string' || v.trim() === '') {
      issues.push(path);
    }
  }
  return issues;
}

describe('i18n key parity', () => {
  it('en-US 所有 key 在 zh-CN 中存在', () => {
    const enKeys = flattenKeys(enUS as unknown as I18nRecord);
    const zhKeys = new Set(flattenKeys(zhCN as unknown as I18nRecord));
    const missing = enKeys.filter((k) => !zhKeys.has(k));
    expect(missing, `zh-CN missing keys: ${missing.join(', ')}`).toEqual([]);
  });

  it('zh-CN 所有 key 在 en-US 中存在（无多余 key）', () => {
    const zhKeys = flattenKeys(zhCN as unknown as I18nRecord);
    const enKeys = new Set(flattenKeys(enUS as unknown as I18nRecord));
    const extra = zhKeys.filter((k) => !enKeys.has(k));
    expect(extra, `zh-CN has extra keys not in en-US: ${extra.join(', ')}`).toEqual([]);
  });

  it('en-US 所有值非空', () => {
    const issues = checkNonEmpty(enUS as unknown as I18nRecord);
    expect(issues, `en-US empty values at: ${issues.join(', ')}`).toEqual([]);
  });

  it('zh-CN 所有值非空', () => {
    const issues = checkNonEmpty(zhCN as unknown as I18nRecord);
    expect(issues, `zh-CN empty values at: ${issues.join(', ')}`).toEqual([]);
  });

  it('en-US ↔ zh-CN 顶级 section 数一致', () => {
    const enSections = Object.keys(enUS).sort();
    const zhSections = Object.keys(zhCN).sort();
    expect(enSections).toEqual(zhSections);
  });

  it('静态 key 计数稳定（防意外删减）', () => {
    const enKeys = flattenKeys(enUS as unknown as I18nRecord);
    // 当前已知 key 数: en-US = zh-CN
    expect(enKeys.length).toBeGreaterThanOrEqual(90);
    expect(enKeys.length).toBeLessThanOrEqual(200);
  });
});
