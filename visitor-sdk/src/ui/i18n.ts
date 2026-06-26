// 1r: SDK 端轻量 i18n(无 vue-i18n 依赖,按 navigator.language 切换)
// 默认中英文,未来可扩展 SDK config.messages 覆盖

export type Locale = 'zh' | 'en';

export function detectLocale(): Locale {
  if (typeof navigator === 'undefined') return 'en';
  return /^zh\b/i.test(navigator.language) ? 'zh' : 'en';
}

// SDK 端文案字典(1r:替换原硬编码)
// Phase 5:加 consent / cobrowse / chat Calm UI 新 key
export const sdkMessages = {
  zh: {
    popup_dismiss: '关闭',
    cobrowse_operator_label: '运营员',
    cobrowse_fill_toast: '正在代为填写 {field}',
    cobrowse_field_fallback: '字段',
    chat_header: '客服',
    chat_input_placeholder: '输入消息...',
    chat_send: '发送',
    // Phase 5: Calm co-browse banner 文案
    cobrowse_assist_hint: '{name} 正在协助你 · 可见页面操作',
    cobrowse_exit: '退出协助',
    // Phase 5: Calm consent card 文案
    consent_privacy_link: '隐私',
    consent_wordmark: 'pinconsole',
    consent_tagline: '开源访客监控控制台',
  },
  en: {
    popup_dismiss: 'Close',
    cobrowse_operator_label: 'Operator',
    cobrowse_fill_toast: 'Filling {field} on your behalf',
    cobrowse_field_fallback: 'field',
    chat_header: 'Support',
    chat_input_placeholder: 'Type a message...',
    chat_send: 'Send',
    // Phase 5: Calm co-browse banner text
    cobrowse_assist_hint: '{name} is assisting you · can see page actions',
    cobrowse_exit: 'End assistance',
    // Phase 5: Calm consent card text
    consent_privacy_link: 'Privacy',
    consent_wordmark: 'pinconsole',
    consent_tagline: 'open-source visitor console',
  },
} as const;

export type SdkMessageKey = keyof typeof sdkMessages['zh'];

export function t(key: SdkMessageKey, locale?: Locale, params?: Record<string, string>): string {
  const loc = locale ?? detectLocale();
  const msgs = sdkMessages[loc] ?? sdkMessages.en;
  let s: string = msgs[key] ?? sdkMessages.en[key] ?? key;
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      s = s.replace(`{${k}}`, v);
    }
  }
  return s;
}
