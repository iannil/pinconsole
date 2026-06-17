# 验证深度判定标准

> 本文件落地 [`CLAUDE.md`](../../CLAUDE.md) 中"工作提示 → 测试深度判定"小节。
> 每个切片(以及未来每个交付单元)按以下三级标定验证深度。深度决定 LLM
> 与人类对该切片实现的可信度假设。

## 1. 三级定义

### 🟢 verified-deep(深度验证)

必须满足以下全部:

1. **正向 e2e 覆盖**:Playwright 或同级测试覆盖**用户可见的真实行为**
   - 不只是 `request.get/post` 调 API endpoint
   - admin / visitor 都通过真实浏览器交互
   - 测试时序模拟真实使用场景(打开 admin → 访客访问 → 订阅 → 触发交互)

2. **断言强度**:测试断言必须验证实际行为,而不是只 `expect(resp.ok()).toBeTruthy()`
   - 例子:鼠标坐标到了、rrweb-player 真渲染了、replay 控制器真响应了、chat 面板真出现消息了

3. **切片特性要求**:
   - **展示/采集类切片**(如 1a 骨架、1b 单向最小、1c rrweb、1d 录像):1 + 2 即可
   - **安全/边界类切片**(如 1e 紧急退出、1f navigate 跨域拒绝、1g 离线消息、1h 认证、1i 反爬):**额外要求至少 1 个负向测试**(401/403/429/拒绝/失败模式)

4. **不允许静默跳过**:`if (!x.length) return;` 类让测试在空状态下静默 pass 的模式自动降级为 🟡

### 🟡 verified-shallow(浅验证)

至少有 e2e/集成测试触达该切片,但存在以下任一:

- 测试只验证 API endpoint(`request.post(...).ok()`),没有真实 UI 流
- 断言强度不足(只 `resp.ok()` 或只检查 console 文本)
- 含静默跳过模式(`if (!x.length) return;`)
- 切片属安全/边界类但缺负向测试

### 🔴 implemented-unverified(已实现未验证)

代码已交付、能编译、可能跑过 smoke,但**无切片级 e2e/集成测试**。

## 2. 升级路径

| 当前 → 目标 | 需补 |
|---|---|
| 🔴 → 🟡 | 加任意触达该切片的 e2e(即使是 API 级 smoke) |
| 🟡 → 🟢 | (a) 替换浅断言为真实行为断言;(b) 若是安全/边界类,补负向测试;(c) 移除静默跳过模式 |
| 🟢 维持 | 任何重构后必须重新确认上述四项仍满足 |

## 3. 当前分布(2026-06-18 A 阶段升级后)

详见 [`docs/project-status.md`](../project-status.md) §5。snapshot:

| 切片 | 深度 | 备注 |
|---|---|---|
| 1a 骨架 | 🟢 | – |
| 1b 单向最小 | 🟢 | – |
| 1c rrweb | 🟢 | – |
| 1d 录像归档 | 🟢 | – |
| 1e 双向通道 | 🟢 | – |
| 1f 表单 + 跳转 | 🟢 | – |
| 1g 弹窗 + 聊天 | 🟢 | – |
| 1h 认证 + 多运营(后端) | 🔴 | spec partial:决策 #5 login UI 未实施;已拆为 1h-ui 新切片 |
| 1i 反爬虫 | 🟢 | A 阶段升级:BehaviorTracker 接线 + Go 单测覆盖深度 + e2e 真查 PG fingerprint |
| 1j i18n + 部署 + CI | 🟢 | A 阶段升级:硬编码抽 key + 语言切换按钮 + 4 个真 e2e + 修 Dockerfile go 版本 bug |

> **分布会变化**:本节 snapshot 可能滞后。永远以 `docs/project-status.md` §5 为准。

> **新警示(2026-06-18)**:reality check 验证测试存在,但 **spec vs 实施对照**才发现 1h/1i/1j 三处重大 gap。深度判定**必须**包含 spec 决策点逐项验证,不只是测试通过。

## 4. 与完成报告的关系

每个 `docs/reports/completed/slice-X-implementation.md` 报告顶部都有:

```markdown
> **Verification Depth**: 🟢/🟡/🔴(以 YYYY-MM-DD reality check 为准)
> **报告叙述免责**:[详见模板](../templates/report.md)
```

报告内容是"实施者自述",**不是**"已验证事实"。报告叙述的准确性在 A 阶段补深测时一并 audit。

## 5. 触发重新评级的场景

- 加新 e2e 测试 → 评估是否升级
- 重构某切片代码 → 评估是否需要重新跑全 e2e 守底
- 发现 bug → 评估是否需要降级(说明测试不够)
- reality check → 每次完整审计后更新分布表
