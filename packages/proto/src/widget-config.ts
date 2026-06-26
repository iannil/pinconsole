// Widget 配置类型（page-editor 切片，pe-1）
// 运营可在 admin 后台编辑 4 类 visitor UI widget 的文案和样式。
// SDK init 时 GET /api/widget-config 获取，替换默认硬编码。

// ── Widget 类型枚举 ──────────────────────────────────────

export type WidgetType = 'popup' | 'chat' | 'cobrowse_banner' | 'consent_banner';

// ── 各 Widget 配置 ───────────────────────────────────────

/** 弹窗（popup）配置 */
export interface PopupConfig {
  title: string;
  body: string;
  action_label: string;
  action_url?: string;
  dismissible: boolean;
  primary_color?: string; // CSS color, e.g. "#1a73e8"
}

/** 聊天（chat）配置 */
export interface ChatConfig {
  header: string;
  placeholder: string;
  send_label: string;
  bubble_color?: string;
  header_color?: string;
}

/** 共浏览 banner（cobrowse_banner）配置 */
export interface CoBrowseBannerConfig {
  operator_label: string;
  assist_hint: string;
  exit_label: string;
}

/** 同意书 banner（consent_banner）配置 */
export interface ConsentBannerConfig {
  title: string;
  body: string;
  accept_label: string;
  reject_label: string;
  privacy_link?: string;
}

// ── 统一配置结构 ─────────────────────────────────────────

/** 4 类 widget 的完整配置映射 */
export interface WidgetConfigMap {
  popup: PopupConfig;
  chat: ChatConfig;
  cobrowse_banner: CoBrowseBannerConfig;
  consent_banner: ConsentBannerConfig;
}

/** API 响应的完整结构 */
export interface WidgetConfigResponse {
  tenant_id: string;
  configs: Partial<WidgetConfigMap>;
}

/** API 更新的请求结构 */
export interface WidgetConfigUpdateRequest {
  widget_type: WidgetType;
  /** 完整的 widget 配置（覆盖式更新） */
  config: PopupConfig | ChatConfig | CoBrowseBannerConfig | ConsentBannerConfig;
}
