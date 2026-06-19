// 1ad 续集测试:Dashboard.vue toggleCoBrowsing + auto-subscribe 接线源码契约
// (审计 v1-followups T1-v1f-2 + T1-v1f-3)。
//
// v1-followups fix1 toggleCoBrowsing 自动 claim/release:
//   toggleCoBrowsing 进入时 POST claim,退出时 POST release(而不是裸显 overlay)。
//   否则后端 requireClaimOwnership 会 403,overlay 形同虚设。
//
// v1-followups fix5 切换 visitor 自动 subscribe:
//   selectedSessionId 变化时,自动 subscribe 新 session。
//   否则用户需要手动点订阅,UX 退化。
//
// 源码契约:Vue 组件 mount 测试 setup 成本高,用源码 grep 验证接线。
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const dashboardSrc = readFileSync(resolve(__dirname, '../src/views/Dashboard.vue'), 'utf8');

describe('1ad: Dashboard.vue toggleCoBrowsing 接线(v1-followups fix1)', () => {
  it('T1-v1f-2: toggleCoBrowsing 函数存在', () => {
    expect(dashboardSrc).toContain('async function toggleCoBrowsing');
  });

  it('T1-v1f-2: 进入 co-browse 时调 claimSession', () => {
    // 找 toggleCoBrowsing 函数体
    const idx = dashboardSrc.indexOf('async function toggleCoBrowsing');
    expect(idx).toBeGreaterThan(-1);
    const fnEnd = dashboardSrc.indexOf('\n}', idx);
    const fnBody = dashboardSrc.slice(idx, fnEnd);
    expect(fnBody).toContain('claimSession');
  });

  it('T1-v1f-2: 退出 co-browse 时调 releaseSession', () => {
    const idx = dashboardSrc.indexOf('async function toggleCoBrowsing');
    const fnEnd = dashboardSrc.indexOf('\n}', idx);
    const fnBody = dashboardSrc.slice(idx, fnEnd);
    expect(fnBody).toContain('releaseSession');
  });

  it('T1-v1f-2: claim 失败时设 claimError(不激活 overlay)', () => {
    const idx = dashboardSrc.indexOf('async function toggleCoBrowsing');
    const fnEnd = dashboardSrc.indexOf('\n}', idx);
    const fnBody = dashboardSrc.slice(idx, fnEnd);
    expect(fnBody).toMatch(/catch.*claimError|claimError.*=.*error/i);
  });
});

describe('1ad: Dashboard.vue 切换 visitor 释放旧 claim(v1-followups fix1 副验)', () => {
  it('T1-v1f-2: selectedSessionId 变化时释放旧 claim(防 stale lock)', () => {
    // watch 模式存在 + 包含 releaseSession(oldId)
    expect(dashboardSrc).toMatch(/watch\([\s\S]+?selectedSessionId/);
    expect(dashboardSrc).toContain('releaseSession(oldId)');
  });
});

describe('1ad: Dashboard.vue useWs subscribe 接线(v1-followups fix5 部分)', () => {
  it('T1-v1f-3: useWs 解构 subscribe + unsubscribe', () => {
    expect(dashboardSrc).toMatch(/const\s*\{[\s\S]*subscribe[\s\S]*unsubscribe[\s\S]*\}\s*=\s*useWs/);
  });

  it('T1-v1f-3: useWs 启用 onPresence 回调(visitors store 同步)', () => {
    expect(dashboardSrc).toMatch(/onPresence[\s\S]*store\.applyPresence/);
  });

  it('T1-v1f-3: useWs 启用 onEvent 回调(events store 同步)', () => {
    expect(dashboardSrc).toMatch(/onEvent[\s\S]*store\.appendEvent/);
  });
});
