# Slice TS-1 + TS-2 implementation:vitest coverage + visitor-sdk 核心

> **状态**:TS-1 completed / TS-2 partial
> **深度 badge**:🟡 touched
> **实际工时**:~2h

## 实施清单

### TS-1 vitest coverage 配置

#### `admin/vite.config.ts` + `visitor-sdk/vite.config.ts`

加 `/// <reference types="vitest/config" />` + `test` 段:

```typescript
test: {
  environment: 'jsdom',
  coverage: {
    provider: 'v8',
    reporter: ['text', 'json', 'html', 'lcov'],
    exclude: ['node_modules/', 'dist/', '**/*.d.ts', '**/*.config.ts', 'tests/**', ...],
    thresholds: {
      lines: 60,      // admin 起步;visitor-sdk 15%(因 src/index.ts 拉低)
      functions: 50,
      branches: 50,
      statements: 60,
    },
  },
},
```

#### root `package.json` 加 `test:js:cov` script

```json
"test:js:cov": "pnpm --filter '@pinconsole/admin' --filter '@pinconsole/visitor-sdk' --if-present exec vitest run --coverage"
```

#### rename 遗留修复

`visitor-sdk/vite.config.ts` 的 `lib.name: 'MarketingMonitorSDK'` → `'PinconsoleSDK'`。

### TS-2 visitor-sdk 核心(partial)

#### `tests/handler_coverage.test.ts`(新建,~220 行,16 测试)

mock OperatorCursor/NodeMap/OperatorToast/ui/popup,覆盖 CommandHandler.handle 全 case:
- non-command envelope / undefined payload 忽略
- cursor_highlight / click / scroll / fill_input / navigate / release_control / show_popup / chat_message 全分发
- 无 cursor/click/navigate 等字段时忽略
- chat_message 无 onChatMessage 回调不抛
- 生命周期:start/stop 幂等

#### `tests/popup_coverage.test.ts`(新建,~80 行,6 测试)

- showPopup 创建 overlay + card
- dismissible=true 含关闭按钮
- action_url 渲染 action 链接(querySelector 检查)
- javascript: action_url 被 isURLSchemeAllowed 拒绝
- removePopup 清除 + 幂等

## 实测覆盖率(2026-06-21)

### admin

```bash
$ cd admin && pnpm exec vitest run --coverage
All files | 82.17 | 78.52 | 64.81 | 82.17
```

### visitor-sdk

```bash
$ cd visitor-sdk && pnpm exec vitest run --coverage
All files | 36.05 | 75.46 | 77.52 | 36.05
```

## 关键修正

### 修正 1:版本兼容

`@vitest/coverage-v8` 最新 4.1.9 不兼容 vitest 2.1.9。改装 `^2.1.9`。

### 修正 2:lib.name rename 遗留

`MarketingMonitorSDK` → `PinconsoleSDK`(rename 重构遗漏)。

### 修正 3:innerHTML XSS 警告

测试中检查 popup 内容初版用 `innerHTML.includes(...)`,触发安全工具警告。改用 `querySelector('a[href]')` + `getAttribute('href')` 精确检查,避免 innerHTML 关键字触发 XSS 警告扫描。

### 修正 4:jsdom 导航限制

`navigate` case 测试时 `doNavigate` 调 `location.href = url`,jsdom 不允许导航会输出 stderr。这是 jsdom 限制(不是测试错误),不影响测试 PASS。

## 未达 90% 的说明

visitor-sdk 36.05%,差 53.95pp。主因:
- `src/index.ts`(400 行 SDK 主入口,占 visitor-sdk LOC ~40%)未测
- `collectors/rrweb.ts/screenshot.ts` 需 mock rrweb 库
- `ui/consentBanner.ts/chatWidget.ts/coBrowseBanner.ts` DOM + i18n 集成

**决策**:按 plan 风险点 #5,接受 36.05% 作为 TS-2 当前结果,留 backlog。

## 副作用与回归

- ✅ admin / visitor-sdk 现有测试不破坏
- ✅ 新测试全 PASS(handler 16 + popup 6 = 22 测试)
- ✅ coverage threshold 起步低(15%/60%),不阻塞 CI

## 同步文档

- ✅ project-status §2.1 TS 行更新(admin / visitor-sdk 首次有覆盖率)
- ✅ daily 追加 TS-1+TS-2 段

## 后续切片

- TS-3:admin 纯逻辑层(fetchJson.ts/api/*.ts/stores/*.ts)~8h
- TS-4:3 个 .vue 组件 ~6h
- TS-5:门槛拉到 90% ~4h
- visitor-sdk src/index.ts backlog:补 main entry 测试(预估 6-8h)
