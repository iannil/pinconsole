// 1ad 续集测试:visitor-sdk rrweb 重试 + visibility + iframe + 截图 接线(审计 T1-1c 4 项 + T1-1b-4)。
// 1af G3 扩展:除源码契约外,加行为级测试 — 真实例化 RRWebCollector + 验证默认配置。
//
// 源码契约保留(重构回归保护);行为级测试加在底部,验证语义正确。
//
// 不测:实际 rrweb 调用语义(需 Playwright e2e)
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { RRWebCollector } from '../src/collectors/rrweb';

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

// 1af G3: 行为级测试 — 真实例化 RRWebCollector + 断言默认配置值。
//
// 源码契约测试只 grep 字符串,不能捕获:
// - 配置变量名改了但默认值错(如 maxRetries ?? 0 仍 grep "maxRetries" PASS)
// - 配置未真应用到 this.opts(构造函数漏赋值)
//
// 行为级测试通过真实例化,直接读 collector 实例的 opts 字段。
describe('1af G3: rrweb collector 行为级默认配置', () => {
  // Helper:通过类型断言读私有 opts 字段
  type RRWebCollectorInternals = {
    opts: {
      maskAllInputs: boolean;
      maxRetries: number;
      visibilityHiddenMs: number;
      blockClass: string;
      ignoreClass: string;
    };
  };

  it('T1-1c-5 behavioral: 默认 maxRetries=3 + visibilityHiddenMs=60000', () => {
    const collector = new RRWebCollector(() => {}) as unknown as RRWebCollectorInternals;
    expect(collector.opts.maxRetries).toBe(3);
    expect(collector.opts.visibilityHiddenMs).toBe(60_000);
  });

  it('T1-1b-4 behavioral: 默认 maskAllInputs=true(敏感字段过滤)', () => {
    const collector = new RRWebCollector(() => {}) as unknown as RRWebCollectorInternals;
    expect(collector.opts.maskAllInputs).toBe(true);
  });

  it('T1-1c-5 behavioral: 用户传入 maxRetries 覆盖默认值', () => {
    const collector = new RRWebCollector(() => {}, { maxRetries: 5 }) as unknown as RRWebCollectorInternals;
    expect(collector.opts.maxRetries).toBe(5);
    expect(collector.opts.maxRetries).not.toBe(3); // 不能 silent 回退到默认
  });

  it('T1-1c-5 behavioral: 用户传入 maskAllInputs=false 覆盖默认', () => {
    const collector = new RRWebCollector(() => {}, { maskAllInputs: false }) as unknown as RRWebCollectorInternals;
    expect(collector.opts.maskAllInputs).toBe(false);
  });

  it('T1-1c behavioral: blockClass + ignoreClass 默认值', () => {
    const collector = new RRWebCollector(() => {}) as unknown as RRWebCollectorInternals;
    expect(collector.opts.blockClass).toBe('mm-block');
    expect(collector.opts.ignoreClass).toBe('mm-ignore');
  });
});
