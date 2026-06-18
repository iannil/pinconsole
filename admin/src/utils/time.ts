// 时间格式化工具(1r i18n 化)
// 用法:formatRelative(ts, t) 其中 t 是 vue-i18n 的 t 函数
// i18n key:time.just_now / time.seconds_ago / time.minutes_ago / time.hours_ago / time.fallback_date

import type { ComposerTranslation } from 'vue-i18n';

export function formatRelative(ts: number, t: ComposerTranslation): string {
  const diff = Date.now() - ts;
  if (diff < 1000) return t('time.just_now');
  if (diff < 60_000) return t('time.seconds_ago', { n: Math.round(diff / 1000) });
  if (diff < 3_600_000) return t('time.minutes_ago', { n: Math.round(diff / 60_000) });
  if (diff < 86_400_000) return t('time.hours_ago', { n: Math.round(diff / 3_600_000) });
  const d = new Date(ts);
  return t('time.fallback_date', { month: d.getMonth() + 1, day: d.getDate() });
}
