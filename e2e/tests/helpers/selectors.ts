// 共享 e2e selectors
// 抽自原 realtime.spec.ts 各测试中的字符串 selector

export const Sel = {
  visitorList: '.visitor-list',
  visitorListItem: '.visitor-list li',
  visitorListItemNonEmpty: '.visitor-list li:not(.empty)',
  eventsArea: '.events-area',
  subscribeBtnText: '订阅实时',
  unsubscribeBtnText: '取消订阅',
} as const;
