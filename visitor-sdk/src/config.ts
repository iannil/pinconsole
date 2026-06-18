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
  /** 1l:consent 模式(默认 opt-in = 用户必须同意才采集) */
  consentMode?: 'opt-in' | 'opt-out' | 'always-on' | 'always-off';
  /** 1l:co-browsing 接管横幅是否显示(默认 true,GDPR Art.22 透明度) */
  showCoBrowseBanner?: boolean;
  /** 1l:consent banner 自定义文案(可覆盖默认中英文) */
  consentBannerText?: { title?: string; body?: string; accept?: string; reject?: string };
}

const DEFAULTS: VisitorConfig = {
  enableRecording: false,
  debug: false,
  consentMode: 'opt-in',
  showCoBrowseBanner: true,
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

  const consentMode = get('data-consent-mode');
  return {
    tenantId: get('data-tenant-id'),
    pageId: get('data-page-id'),
    endpoint: get('data-endpoint'),
    enableRecording: parseBool(get('data-enable-recording')),
    debug: parseBool(get('data-debug')),
    consentMode: consentMode === 'opt-in' || consentMode === 'opt-out' || consentMode === 'always-on' || consentMode === 'always-off'
      ? consentMode
      : undefined,
    showCoBrowseBanner: parseBool(get('data-show-co-browse-banner')),
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
