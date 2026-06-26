// @pinconsole/proto — 共享协议定义
//
// 1p-llm-friendly:消除 admin/SDK 手写同步重复,从单一源消费。
// Go 端在 server/internal/proto/ 单独维护(类型系统不同,无法共享)。
// 未来协议稳定后可引入 codegen(buf/quicktype 从单一 schema 生成 Go + TS)。
export * from './envelope';
export * from './events';
export * from './command';
export * from './widget-config';
