// 1ad 续集测试:visitor-sdk rrweb 重试 + visibility + iframe + 截图 接线(审计 T1-1c 4 项 + T1-1b-4)。
//
// 源码契约:验证 SDK 关键韧性逻辑接线(重构回归保护)。
//
// 不测:实际 rrweb 调用语义(需 Playwright e2e)
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const rrwebSrc = readFileSync(resolve(__dirname, '../src/collectors/rrweb.ts'), 'utf8');
const screenshotSrc = readFileSync(resolve(__dirname, '../src/collectors/screenshot.ts'), 'utf8');

describe('1ad: visitor-sdk rrweb 韧性接线', () => {
  it('T1-1c-5: rrweb collector 有 maxRetries 配置 + 默认 3', () => {
    expect(rrwebSrc).toContain('maxRetries');
    expect(rrwebSrc).toMatch(/maxRetries.*\?\?.*3/);
  });

  it('T1-1c-5: rrweb 有 retries 计数器 + 错误重试逻辑', () => {
    expect(rrwebSrc).toContain('private retries');
    // 错误处理路径必须存在(try/catch + retry)
    expect(rrwebSrc.toLowerCase()).toContain('catch');
  });

  it('T1-1c-5: rrweb 监听 visibilitychange + 60s 阈值', () => {
    expect(rrwebSrc).toContain('visibilitychange');
    expect(rrwebSrc).toContain('visibilityHandler');
    // 60s 默认阈值
    expect(rrwebSrc).toMatch(/60[_\s]*000|60\s*\*\s*1000/);
  });

  it('T1-1c-5: rrweb visibility hidden → visible 触发 takeFullSnapshot', () => {
    expect(rrwebSrc).toContain('takeFullSnapshot');
    expect(rrwebSrc).toMatch(/hidden.*takeFullSnapshot|visibility.*takeFullSnapshot/);
  });

  it('T1-1c-3: screenshot collector 存在(选择性截图触发)', () => {
    expect(screenshotSrc).toBeDefined();
    expect(screenshotSrc.length).toBeGreaterThan(0);
    // 应包含 canvas/WebGL/cross-origin iframe 触发判断
    expect(screenshotSrc.toLowerCase()).toMatch(/canvas|webgl|iframe/);
  });

  it('T1-1b-4: rrweb maskAllInputs 默认 true(敏感字段过滤)', () => {
    // 默认配置应启用 maskAllInputs
    expect(rrwebSrc).toMatch(/maskAllInputs.*\?\?.*true/);
  });
});
