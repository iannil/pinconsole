// pe-3:Widget 配置获取器
// SDK start() 时 GET /api/widget-config,替换各 UI widget 硬编码。

import type { WidgetConfigMap } from '@pinconsole/proto';

export type { WidgetConfigMap };

const WIDGET_CONFIG_CACHE_KEY = '__pc_widget_config';

/** 从服务端拉取 widget 配置,优先走缓存。 */
export async function fetchWidgetConfig(apiBase: string): Promise<Partial<WidgetConfigMap>> {
  try {
    const res = await fetch(`${apiBase}/api/widget-config`, { cache: 'no-cache' });
    if (!res.ok) return {};
    const data: { configs: Partial<WidgetConfigMap> } = await res.json();
    const configs = data.configs || {};

    // 缓存到 sessionStorage 供后续使用
    try {
      sessionStorage.setItem(WIDGET_CONFIG_CACHE_KEY, JSON.stringify(configs));
    } catch { /* ignore */ }

    return configs;
  } catch {
    // 网络不可用时尝试缓存
    try {
      const cached = sessionStorage.getItem(WIDGET_CONFIG_CACHE_KEY);
      if (cached) return JSON.parse(cached);
    } catch { /* ignore */ }
    return {};
  }
}

/** 从缓存读取 widget 配置（免网络请求）。 */
export function getCachedWidgetConfig(): Partial<WidgetConfigMap> {
  try {
    const cached = sessionStorage.getItem(WIDGET_CONFIG_CACHE_KEY);
    if (cached) return JSON.parse(cached);
  } catch { /* ignore */ }
  return {};
}
