// 时间格式化工具

export function formatRelative(ts: number): string {
  const diff = Date.now() - ts;
  if (diff < 1000) return '刚刚';
  if (diff < 60_000) return `${Math.round(diff / 1000)} 秒前`;
  if (diff < 3_600_000) return `${Math.round(diff / 60_000)} 分前`;
  if (diff < 86_400_000) return `${Math.round(diff / 3_600_000)} 小时前`;
  const d = new Date(ts);
  return `${d.getMonth() + 1}/${d.getDate()}`;
}
