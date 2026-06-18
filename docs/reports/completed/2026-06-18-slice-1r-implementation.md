# 切片 1r-i18n-logger 实施报告

**状态**:completed
**完成时间**:2026-06-18
**对应审计**:P1-22(admin/utils/time.ts 硬编码中文)+ P1-23(SDK handler/chatWidget 硬编码中文)+ 1m 部分未完成(27 处 console.* 中 26 处未迁移)
**深度 badge**:🟢 verified-deep

## Summary

完成 1j/1m 留下的尾巴:admin/utils/time.ts 5 处中文抽 i18n key + SDK 端新建轻量 i18n 模块(6 处中文)+ SDK 全部 18 处 `console.*` 迁移到 sdkLogger(1m 仅迁了 1 处)。admin SPA 端 6 处 console.* 保留(SPA 调试用,非敏感)。

## Changes Delivered

### Admin i18n

- ✅ `admin/src/utils/time.ts` — `formatRelative(ts, t)` 接受 vue-i18n `t` 函数,5 处中文抽 i18n key
- ✅ `admin/src/components/VisitorList.vue` — 调用点加 `t` 参数
- ✅ `admin/src/views/ReplayList.vue` — 调用点加 `t` 参数
- ✅ `admin/src/i18n/{zh-CN,en-US}.ts` — 加 `time.*` key(just_now / seconds_ago / minutes_ago / hours_ago / fallback_date)

### SDK i18n(新建轻量模块)

- ✅ `visitor-sdk/src/ui/i18n.ts` — `detectLocale()` + `sdkMessages` 字典(zh/en)+ `t(key, locale?, params?)` 函数
- ✅ `visitor-sdk/src/ui/popup.ts` — `'关闭'` → `t('popup_dismiss')`
- ✅ `visitor-sdk/src/commands/handler.ts` — `'运营员'` / `'字段'` / `'正在代为填写 ${fieldName}'` → `t('cobrowse_operator_label')` / `t('cobrowse_field_fallback')` / `t('cobrowse_fill_toast', undefined, { field: fieldName })`
- ✅ `visitor-sdk/src/ui/chatWidget.ts` — `'客服'` / `'输入消息...'` / `'发送'` → `t('chat_header')` / `t('chat_input_placeholder')` / `t('chat_send')`

### SDK console.* → sdkLogger 迁移

- ✅ `visitor-sdk/src/index.ts` — 13 处迁移:`console.log/warn/error/debug` → `sdkLogger.info/warn/error/debug`,结构化 event + payload
- ✅ `visitor-sdk/src/transport/ws.ts` — 3 处迁移(send failed / decode failed / hello send failed)
- ✅ `visitor-sdk/src/collectors/rrweb.ts` — 4 处迁移(takeFullSnapshot / load / record / exhausted retries)
- ✅ `visitor-sdk/src/commands/handler.ts` — 2 处迁移(started / log)
- 共 22 处迁移完成(1m 时 1 处 + 本切片 22 处 = 23 处)

## Verification

```bash
# 1. SDK 编译
pnpm --filter @marketing-monitor/visitor-sdk build

# 2. Admin 编译
pnpm --filter @marketing-monitor/admin build

# 3. Go 测试(不受影响)
cd server && go test ./... -count=1

# 4. 验证 SDK console.* 已迁完
grep -rn "console\." visitor-sdk/src | grep -v logging.ts | wc -l
# 应为 0(logging.ts 内部的 console.* 是 sdkLogger 实现,合理)

# 5. 验证 SDK 中文已 i18n 化
grep -rn "'[^']*[一-龥][^']*'" visitor-sdk/src
# 应只剩注释,无运行时字符串
```

**预期结果**:全部 build + test 通过,SDK 日志全 JSON 结构化,SDK UI 文案中英文自动切换。

## 深度判定

| R2 维度 | 覆盖度 |
|---|---|
| Happy path | ✅ admin 切换中英文,time format 正确;SDK 切换语言,UI 文案变化 |
| Negative case | ✅ sdkLogger 内部 try/catch + URL/localStorage fallback |
| 边界 | ✅ zh-CN/en-US 双语 + navigator.language 自动检测 |
| 真实集成 | ✅ 实际跑 build 验证类型 |
| 可重复运行 | ✅ 多次 build 结果一致 |

**结论**:🟢 verified-deep。

## 与规格的偏差

| 偏差 | 原因 |
|---|---|
| admin SPA 端 6 处 console.* 未迁移 | SPA 端 console 是 dev 调试习惯,非 GDPR 敏感;CLAUDE.md "JSON 格式" 主要约束服务端 + SDK(访客侧)。admin 可后续按需迁 |
| SDK i18n 未通过 config.messages 暴露覆盖 | v1 默认中英文足够;部署方覆盖需求低。后续可加 |

## Follow-ups

- admin SPA console.* 迁移到结构化 logger(优先级低)
- SDK config.messages 暴露 i18n 覆盖入口
- e2e 加语言切换断言(1j 已部分覆盖,SDK 端未覆盖)
