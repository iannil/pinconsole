# 切片 1aa-ts-test-deepening — Spec + Implementation

**切片编号**:1aa
**类型**:测试
**创建时间**:2026-06-19
**完成时间**:2026-06-19
**状态**:completed
**深度**:🟢 verified-deep(112 测试全绿,断言真实行为,无 vacuous truth)
**关联**:[deep-audit P2-12](../../audits/2026-06-18-deep-audit.md)、[1t 测试覆盖补全](./2026-06-18-slice-1t-implementation.md)、[IMPLEMENTATION_PLAN](../../../IMPLEMENTATION_PLAN.md)

## Context

deep-audit P2-12 指出 admin 和 visitor-sdk 各只有 2 个 vitest 测试,其中各 1 个是 `1+1=2` 占位 smoke。v1 e2e 65 测试全绿但单测层覆盖薄——e2e 慢、CI 资源消耗大、调试不友好。补单测能在重构时提供毫秒级反馈、锁定纯逻辑契约、降低 e2e 维护成本。

## 决策表

| # | 决策点 | 选定 | 理由 |
|---|---|---|---|
| 1 | 范围 | B 完整覆盖 | A 漏掉 SDK transport 关键逻辑;C 引入 coverage 工具链增加 CI 复杂度 |
| 2 | admin 重点 | stores + router + API + composables | useWs 是 SPA 状态机高风险区(v1-followups 已验证) |
| 3 | SDK 重点 | config + batch + session + transport | transport 重连逻辑 + 协议契约是核心 |
| 4 | Mock 策略 | vitest 内置 vi.stubGlobal | 项目已有模式,不引入新依赖 |
| 5 | DOM 环境 | admin 用 jsdom / SDK 不用 | SDK 测纯逻辑;admin stores/router 需要 DOM |
| 6 | 测试位置 | 各包 tests/ | 跟随现有约定 |
| 7 | commands/handler.ts | 不测 | DOM 重度依赖(cursor/NodeMap/Toast),投入产出比低,留作 v1 后续 |

## Changes Delivered

### admin(64 测试,新增 54)

- ✅ **`tests/auth.store.test.ts`**(13 测试)— login/logout/fetchMe + 401 handler 注册 + computed 状态(displayName `??` 行为锁定)
- ✅ **`tests/visitors.store.test.ts`**(20 测试)— setInitialList / applyPresence 三态(online/offline/navigated)/ select / appendEvent 单+批+500 截断 / clear / 排序
- ✅ **`tests/api.auth.test.ts`**(7 测试)— postLogin / postLogout / getMe + 401 特殊处理 + X-Trace-Id 自动注入
- ✅ **`tests/router.test.ts`**(6 测试)— beforeEach 守卫 4 分支(已认证→dashboard、未认证→login with query、public 直通、requiresAuth 阻断)+ ensureAuthInit 缓存 + fetchMe 拒绝容错
- ✅ **`tests/useWs.test.ts`**(7 测试)— connect/open 状态机、subscribe 记忆、close 抑制重连、onPresence 回调、decode 失败容错

### visitor-sdk(48 测试,新增 41)

- ✅ **`tests/config.test.ts`**(14 测试,原 smoke 替换)— dropUndefined 行为锁定(v1-followups 修复的回归保护)+ readScriptData + readWindowConfig + parseBool + consentMode enum 校验
- ✅ **`tests/batch.test.ts`**(9 测试)— push/flush/destroy + 阈值触发(50 events 或 100ms)+ timer 去重 + 默认参数
- ✅ **`tests/session.test.ts`**(8 测试)— getOrCreateVisitorId 持久化 + initSession 网络与持久化 + getCachedSessionId / clearCachedSessionId
- ✅ **`tests/transport.ws.test.ts`**(11 测试)— 缓冲入队/flush、缓冲满丢最旧、指数退避(无中间成功时增长)、close 抑制重连、hello/ack 握手、ack.ok=false 触发 close

### 不变

- 现有 `tests/api-client.test.ts`(9 测试)+ `tests/ws-trace-inherit.test.ts`(6 测试)+ 两个 smoke.test.ts 占位均保留

## Verification

```bash
# 单测
pnpm --filter @marketing-monitor/admin test         # 64/64 passed
pnpm --filter @marketing-monitor/visitor-sdk test   # 48/48 passed

# 累计
pnpm test:js                                       # 112/112 passed (admin 64 + SDK 48)

# 类型检查(无错误输出 = 通过)
pnpm --filter @marketing-monitor/admin exec vue-tsc --noEmit
pnpm --filter @marketing-monitor/visitor-sdk exec tsc --noEmit
```

**预期结果**(2026-06-19 实测):
- admin:64 passed / 0 failed,~500ms
- SDK:48 passed / 0 failed,~190ms
- 累计 112 测试,~700ms

**已知 pre-existing issue**(不属本切片):
- `pnpm lint` 失败:`eslint-config-prettier` 未安装,`.eslintrc.cjs` 引用 `prettier` config 找不到。已 git stash 验证非本切片引入。

## 深度判定

🟢 verified-deep 依据:

- 每个 store 测试覆盖状态变化的多个分支(成功/失败/边界/loading 状态)
- 每个 API 测试覆盖正常响应 + 错误映射 + trace_id 注入
- transport 测试覆盖缓冲满策略 + 重连退避增长 + close 抑制
- 重要行为锁定:
  - `dropUndefined` 行为(v1-followups 关键修复)
  - visitors store SDK reload 自动切 session(v1-followups 关键修复)
  - selectedSessionId offline 不清空(v1-followups 关键修复)
  - displayName `??` 空字符串行为(锁定当前实现,非 bug)
  - 500 events 截断策略
  - fetchMe 内部 catch 不抛错(router 依赖此契约)

无 vacuous assertion(每个 expect 都验证具体值/状态而非 trivial truthy)。

## Follow-ups

- **commands/handler.ts 测试**:DOM 重度依赖,需 jsdom + 复杂 mock。投入产出比低,留作 v1 后续
- **Vue 组件测试**:需 @vue/test-utils mount + DOM 交互,投入大收益低
- **coverage 配置**:本切片未引入 vitest coverage,留作单独切片
- **lint 修复**:`eslint-config-prettier` 缺失,非本切片引入,但应单独修

## Notes

- **测试隔离**:`vi.resetModules` + 动态 import 用于 router 测试,因 router 是单例且 ensureAuthInit 缓存 promise
- **Mock WebSocket 生命周期**:MockWebSocket 初始 readyState=CONNECTING(0),fireOpen 后变 OPEN(1),模拟真实 WS 生命周期。原 ws-trace-inherit.test.ts 直接 OPEN 是因为只测 ack 后行为
- **Partial\<T\> 回归保护**:config.test.ts 中 'regression: Partial<T> with explicit undefined must not override DEFAULTS' 测试明确锁定 v1-followups 修复行为
- **选中状态锚点**:visitors store selectedFingerprint 是内部状态(未导出),通过 selectedSessionId 行为间接验证
