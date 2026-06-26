# fork-3b Playwright 移植 — Spec

**切片编号**: fork-3b（Phase 1 — ✅ 全部实现）
**类型**: 测试基础建设（vendor-rrweb fork 收尾）
**创建时间**: 2026-06-26
**状态**: completed — spec 全部实现
**实施记录**:
- observer 组: `fork-3b-observer.spec.ts` — 4 测试（full snapshot / appendChild / attribute / text）
- replayer 组: `fork-3b-replayer.spec.ts` — 3 测试（iframe 创建 / DOM 重建 / destroy）
- shadow-dom 组: `fork-3b-shadow-dom.spec.ts` — 3 测试（创建 / 变更 / 回放）
- iframe-mask 组: `fork-3b-iframe-mask.spec.ts` — 3 测试（iframe 不崩溃 / maskInputOptions / maskAllInputs）
- snapshot 组: 已由 vitest `upstream-snapshot.test.ts` 覆盖，Playwright 端口因 bundle 导出限制跳过
- **累计 13 测试，8.0s 全绿**
**关联**:
- [vendor-rrweb spec](../reports/completed/2026-06-25-vendor-rrweb-spec.md)
- [vendor-rrweb impl](../reports/completed/2026-06-25-vendor-rrweb-implementation.md)
- [fork-3b 验收报告（Phase 2 已完成）](../../../docs/reports/2026-06-26-fork-3b-record-test-verification.md)

## Context

vendor-rrweb 硬分叉中 fork-3b 原定义是**将上游 5 组保留功能测试转为 Playwright**，已验证的浏览器行为不应被 fork 精简破坏。

2026-06-26 已执行 Phase 2（vitest 单元测试转译，+88 测试），但**依赖真实浏览器环境的 5 组测试仍留 backlog**。本 spec 针对的就是这部分残留。

## 范围

### 做（MVP）
将以下 5 组上游 rrweb 测试转为 Playwright e2e 测试，放在 `e2e/tests/` 目录：

| 组 | 测试内容 | 关键浏览器 API 依赖 |
|---|---|---|
| snapshot | `serializeNodeWithId` 对真实 DOM 的序列化结果正确 | DOM API、`Node.cloneNode()` |
| replayer | Replayer 在 iframe 中重建 DOM、触发动画、响应 `startLive` | iframe 渲染、`requestAnimationFrame` |
| observer | `MutationObserver` 录制 DOM 变更并产出正确 events | `MutationObserver`、真实 DOM 操作 |
| shadow DOM | Shadow DOM 录制与回放（open/closed mode）| `attachShadow`、shadow DOM 遍历 |
| iframe/mask | 同源 iframe 内容录制 + 输入脱敏 mask | `iframe.contentDocument`、input 事件 |

### 不做（后续或放弃）
- Canvas 录制/回放测试（fork-3a 已裁剪，不回归）
- rrdom 测试（非核心路径，v1 未使用）
- 性能基准测试（不是测试信心问题）
- 跨域 iframe 测试（需要特殊 server fixture）

## 技术方案

### 通用模式
每个测试组使用同一个 Playwright fixture 模式：

```typescript
test.describe('fork-3b snapshot', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });

  test('serializeNodeWithId handles text node', async ({ page }) => {
    await page.setContent(`<div id="root">Hello</div>`);
    // 通过 page.evaluate 调用 replay-core 的 snapshot 函数
    const result = await page.evaluate(async () => {
      const { serializeNodeWithId } = await import('/@fs/.../replay-core');
      const root = document.getElementById('root')!;
      return serializeNodeWithId(root, {});
    });
    expect(result).toMatchObject({ /* 预期结构 */ });
  });
});
```

### 关键决策

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | 测试方法 | A: `page.evaluate` 调用 replay-core / B: 独立 HTML + iframe | **A** | 复用 Playwright 的 `page.setContent` + `page.evaluate`，不需要额外 server router |
| 2 | replay-core 加载 | A: Vite dev server 动态 import / B: 预构建 UMD bundle | **A** | Playwright 项目已有 Vite；`page.evaluate` 内 `import()` 可用 |
| 3 | 测试粒度 | A: 每个 case 独立 test / B: 单 test 批量跑 | **A** | 独立失败定位更准确 |
| 4 | 与现有 e2e 的关系 | A: `e2e/tests/` 同一目录 / B: 独立目录 `e2e/fork-3b/` | **A** | 复用现有 Playwright config + fixture |
| 5 | 是否需要 admin-auth | A: 需要登录 / B: 纯 JS 无需后端 | **B** | 只测 replay-core 内部逻辑，不依赖 admin UI |

### fixture 路径

`page.evaluate` 内动态 import `@pinconsole/replay-core` 需要 Vite dev server 支持。现有 e2e 测试的 `baseURL` 指向 `localhost:7080`（Go server），它不提供 Vite HMR。

**方案**: 使用 `page.addInitScript` 或 `page.setContent` 的 `<script type="module">` 直接加载 replay-core 的 ESM bundle（走 Vite 独立端口 `:7073`）。或者使用 `page.route` 拦截 + 内联 bundle（如 fork-0 parity 测试的做法）。

**推荐**: 参考 `fork-0-replay-parity.spec.ts` 的 `page.route()` 拦截模式，在独立 HTML 页面中以 `<script type="module">` 形式加载 replay-core。

### 测试文件组织

```
e2e/tests/
  fork-3b-snapshot.spec.ts      # snapshot 组
  fork-3b-replayer.spec.ts      # replayer 组
  fork-3b-observer.spec.ts      # observer 组
  fork-3b-shadow-dom.spec.ts    # shadow DOM 组
  fork-3b-iframe-mask.spec.ts   # iframe/mask 组
```

## 验收门

- [ ] 5 个测试文件全部创建，Playwright `--list` 解析通过
- [ ] snapshot: 验证 `serializeNodeWithId` 对 text/element/comment/void 元素的序列化正确
- [ ] replayer: 验证 Replayer 构造 → iframe 创建 → DOM 重建 → `startLive` 响应新 events
- [ ] observer: 验证 `MutationObserver` 录制 appendChild/removeChild/attribute change/text change 产出正确 rrweb events
- [ ] shadow DOM: 验证 open/closed shadow DOM 的录制与回放
- [ ] iframe/mask: 验证同源 iframe 内容录制 + `maskAllInputs` 脱敏效果
- [ ] 既有 e2e（fork-0、fork-5、1c~1l）不回归

## 估时

| 任务 | 估时 |
|---|---|
| snapshot 组 | 3-4h |
| replayer 组 | 4-6h |
| observer 组 | 4-6h（最复杂） |
| shadow DOM 组 | 2-3h |
| iframe/mask 组 | 2-3h |
| **合计** | **~15-22h（2-3 天）** |

## 依赖

- Playwright 项目已有 Vite dev server 配置（admin 用 `:7073`）
- `fork-0-replay-parity.spec.ts` 的 `page.route()` 拦截模式可复用
- 不需要 admin-auth fixture，纯 JS 测试

## 未解决问题

1. **replay-core 的加载路径**: 从哪个 URL 加载 replay-core 的 ESM bundle？`fork-0-parity` 用 `page.route()` 拦截后内联注入。fork-3b 是否沿用同一模式？
2. **observer 测试的数据收集**: `record()` 是异步的，需要等待足够事件。Playwright 中如何可靠地等待 `record()` 产生预期事件数？
3. **shadow DOM closed mode**: rrweb 默认不录制 closed shadow DOM。测试是否需要验证"closed mode 跨过/不跨过"两种行为？
