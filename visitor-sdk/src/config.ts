// 访客 SDK 配置类型与解析逻辑。
//
// 配置来源（按优先级）：
//   1. window.MM_CONFIG 全局对象（页面方手动设置）
//   2. <script src="/sdk.js" data-tenant-id="..." data-page-id="..."> data-* 属性
//   3. 默认值（仅 endpoint 推断，其余必填）

export interface VisitorConfig {
  /** 后端 WebSocket 端点，默认推断为 location.origin + '/ws' */
  endpoint?: string;
  /** 租户 ID（schema 预留，1a 不使用） */
  tenantId?: string;
  /** 落地页 ID */
  pageId?: string;
  /** 是否启用 rrweb 采集（1a 仅占位，1b 接入） */
  enableRecording?: boolean;
  /** 是否启用调试日志 */
  debug?: boolean;
}

const DEFAULTS: VisitorConfig = {
  enableRecording: false,
  debug: false,
};

/** 从 <script data-*> 与 window.MM_CONFIG 合并配置。 */
export function resolveConfig(): VisitorConfig {
  const fromScript = readScriptData();
  const fromWindow = readWindowConfig();
  return { ...DEFAULTS, ...fromScript, ...fromWindow };
}

function readScriptData(): Partial<VisitorConfig> {
  // 找到当前 script 标签（最后一个 sdk.js）
  const scripts = document.querySelectorAll('script[data-tenant-id], script[data-page-id]');
  const current = scripts[scripts.length - 1];
  if (!current) return {};

  const get = (k: string): string | undefined => current.getAttribute(k) ?? undefined;

  return {
    tenantId: get('data-tenant-id'),
    pageId: get('data-page-id'),
    endpoint: get('data-endpoint'),
    enableRecording: parseBool(get('data-enable-recording')),
    debug: parseBool(get('data-debug')),
  };
}

function readWindowConfig(): Partial<VisitorConfig> {
  const w = window as unknown as { MM_CONFIG?: Partial<VisitorConfig> };
  return w.MM_CONFIG ?? {};
}

function parseBool(v: string | undefined): boolean | undefined {
  if (v === undefined) return undefined;
  return v === 'true' || v === '1';
}
