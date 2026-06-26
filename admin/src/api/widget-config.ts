// pe-1:Widget 配置 REST 客户端
// 对应后端 server/internal/api/widget_config.go

import { apiFetch, apiJson } from './client';

export interface WidgetConfigMap {
  popup?: PopupConfig;
  chat?: ChatConfig;
  cobrowse_banner?: CoBrowseBannerConfig;
  consent_banner?: ConsentBannerConfig;
}

export interface PopupConfig {
  title: string;
  body: string;
  action_label: string;
  action_url?: string;
  dismissible: boolean;
  primary_color?: string;
}

export interface ChatConfig {
  header: string;
  placeholder: string;
  send_label: string;
  bubble_color?: string;
  header_color?: string;
}

export interface CoBrowseBannerConfig {
  operator_label: string;
  assist_hint: string;
  exit_label: string;
}

export interface ConsentBannerConfig {
  title: string;
  body: string;
  accept_label: string;
  reject_label: string;
  privacy_link?: string;
}

export interface WidgetConfigResponse {
  tenant_id: string;
  configs: Partial<WidgetConfigMap>;
}

/** GET /api/widget-config — 拉取全部配置 */
export async function fetchWidgetConfigs(): Promise<WidgetConfigResponse> {
  const { data } = await apiJson<WidgetConfigResponse>('/api/widget-config');
  return data;
}

/** PUT /api/widget-config/:type — 更新某类配置 */
export async function upsertWidgetConfig(
  widgetType: string,
  config: Record<string, unknown>,
): Promise<void> {
  await apiFetch(`/api/widget-config/${widgetType}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ config }),
  });
}
