// 1l-privacy-gdpr:consent banner
// SDK 加载后在 opt-in 模式下显示,访客点同意/拒绝后回调。
// 同意 → 启动 rrweb/screenshot;拒绝 → 不采集。
//
// GDPR Art.7 要求"明确、自由、可撤回"的同意;banner 提供清晰选项 + 撤回入口(mm.setConsent(false))。
const BANNER_ID = '__mm_consent_banner__';

export interface ConsentBannerText {
  title?: string;
  body?: string;
  accept?: string;
  reject?: string;
}

export interface ConsentBannerOptions {
  text?: ConsentBannerText;
  onAccept: () => void;
  onReject: () => void;
}

// 默认中英文文案(根据 navigator.language 自动切换;部署方可通过 config.consentBannerText 覆盖)
function defaultText(): Required<ConsentBannerText> {
  const isZh = /^zh\b/i.test(navigator.language);
  if (isZh) {
    return {
      title: '访客监控同意',
      body: '本站使用访客监控记录您的操作(鼠标、键盘、页面变化),用于客户服务与产品改进。点击"同意"即表示您授权我们采集;可随时撤回。',
      accept: '同意',
      reject: '拒绝',
    };
  }
  return {
    title: 'Visitor Monitoring Consent',
    body: 'This site records your interactions (mouse, keyboard, page changes) for customer service and product improvement. By clicking "Accept" you grant us permission to collect; you may revoke at any time.',
    accept: 'Accept',
    reject: 'Reject',
  };
}

export function showConsentBanner(opts: ConsentBannerOptions): void {
  removeConsentBanner();

  const text = { ...defaultText(), ...(opts.text ?? {}) };

  const banner = document.createElement('div');
  banner.id = BANNER_ID;
  banner.style.cssText = [
    'position: fixed', 'bottom: 0', 'left: 0', 'width: 100%',
    'background: rgba(40, 40, 40, 0.95)', 'color: #fff',
    'padding: 12px 24px', 'z-index: 999999',
    'display: flex', 'align-items: center', 'gap: 16px',
    'font-family: system-ui, sans-serif', 'font-size: 14px',
    'box-shadow: 0 -2px 8px rgba(0,0,0,0.2)',
  ].join(';');

  const content = document.createElement('div');
  content.style.cssText = 'flex: 1';
  const title = document.createElement('strong');
  title.textContent = text.title;
  title.style.cssText = 'display: block; margin-bottom: 4px; font-size: 15px';
  const body = document.createElement('span');
  body.textContent = text.body;
  body.style.cssText = 'line-height: 1.5; opacity: 0.9';
  content.appendChild(title);
  content.appendChild(body);
  banner.appendChild(content);

  const acceptBtn = document.createElement('button');
  acceptBtn.textContent = text.accept;
  acceptBtn.style.cssText = 'background: #409eff; color: #fff; border: none; padding: 8px 20px; border-radius: 4px; cursor: pointer; font-size: 14px';
  acceptBtn.onclick = () => {
    removeConsentBanner();
    opts.onAccept();
  };
  banner.appendChild(acceptBtn);

  const rejectBtn = document.createElement('button');
  rejectBtn.textContent = text.reject;
  rejectBtn.style.cssText = 'background: transparent; color: #fff; border: 1px solid #666; padding: 8px 20px; border-radius: 4px; cursor: pointer; font-size: 14px';
  rejectBtn.onclick = () => {
    removeConsentBanner();
    opts.onReject();
  };
  banner.appendChild(rejectBtn);

  document.body.appendChild(banner);
}

export function removeConsentBanner(): void {
  const el = document.getElementById(BANNER_ID);
  if (el && el.parentNode) el.parentNode.removeChild(el);
}
