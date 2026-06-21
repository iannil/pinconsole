# Slice TS-1 + TS-2 spec:vitest coverage 配置 + visitor-sdk 核心补测(部分)

> **状态**:TS-1 completed / TS-2 partial(核心部分完成)
> **实际工时**:TS-1 ~1h / TS-2 ~1h
> **深度 badge**:🟡 touched

## 范围

### TS-1:vitest coverage 配置(completed)

| 包 | 配置前 | 配置后(thresholds) |
|---|---|---|
| admin | 默认无配置 | provider=v8 / reporter 4 种 / exclude 标准 / lines 60% 起步 |
| visitor-sdk | 默认无配置 | 同上 / lines 15% 起步(src/index.ts 400 行未测拉低) |

**主要工作**:
- 装 `@vitest/coverage-v8@^2.1.9`(兼容 vitest 2.1.9)
- 修改 `admin/vite.config.ts` + `visitor-sdk/vite.config.ts` 加 `test.coverage` 段
- 修复 rename 遗留:`visitor-sdk` 的 `lib.name` `MarketingMonitorSDK` → `PinconsoleSDK`
- root `package.json` 加 `test:js:cov` script

**实测**(2026-06-21):
- admin: 82.17% lines / 78.52% branches / 64.81% functions
- visitor-sdk: 23.28% lines / 77.46% branches / 73.23% functions

### TS-2:visitor-sdk 核心补测(partial)

| 文件 | 覆盖前 | 覆盖后 |
|---|---|---|
| src/commands/handler.ts | 0% | ~95%(16 测试覆盖全 case) |
| src/ui/popup.ts | 0% | ~90%(6 测试 showPopup/removePopup + URL scheme 白名单) |
| src/index.ts(SDK 主入口) | 0% | 0%(留 backlog) |
| src/collectors/rrweb.ts/screenshot.ts | 0% | 0%(留 backlog,需 rrweb mock) |
| src/ui/consentBanner.ts/chatWidget.ts/coBrowseBanner.ts | 0% | 0%(留 backlog) |
| **整包** | 23.28% | **36.05%** |

**未完成的部分**(留 backlog):
- src/index.ts(400 行 SDK 主入口):需 mock rrweb/fetch/WebSocket/localStorage 完整 setup
- collectors/rrweb.ts + screenshot.ts:需 mock rrweb 库 + Canvas API
- ui/consentBanner.ts/chatWidget.ts/coBrowseBanner.ts:DOM 操作 + i18n 集成

## 验证

```bash
cd /Users/rong.zhu/Code/pinconsole/visitor-sdk
pnpm exec vitest run --coverage
# All files: 36.05% lines / 75.46% branches / 77.52% functions
```

## 未达 90% 目标说明

visitor-sdk 36.05%,差 53.95pp 到 90%。主要因为 src/index.ts(400 行 SDK 主入口)未测,该文件占 visitor-sdk 总 LOC ~40%。

**决策**:按 plan 风险点 #5,接受 36.05% 作为 TS-2 当前结果,留 backlog 补 src/index.ts + collectors + 剩余 ui 文件。

## 关键修正

### 修正 1:@vitest/coverage-v8 版本

初版装最新版 4.1.9,与 vitest 2.1.9 不兼容(`unmet peer vitest@4.1.9`)。改装 `^2.1.9` 兼容版本。

### 修正 2:lib.name rename 遗留

`visitor-sdk/vite.config.ts` 的 `lib.name: 'MarketingMonitorSDK'`(rename 重构遗漏),改为 `'PinconsoleSDK'`。

### 修正 3:innerHTML XSS 警告

初版测试用 `innerHTML` 检查 popup 内容,触发安全警告。改用 `querySelector('a[href]')` + `getAttribute('href')` 精确检查,避免 innerHTML 关键字。
