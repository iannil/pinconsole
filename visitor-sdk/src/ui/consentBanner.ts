// 1l-privacy-gdpr:consent banner
// SDK 加载后在 opt-in 模式下显示,访客点同意/拒绝后回调。
// 同意 → 启动 rrweb/screenshot;拒绝 → 不采集。
//
// GDPR Art.7 要求"明确、自由、可撤回"的同意;banner 提供清晰选项 + 撤回入口(mm.setConsent(false))。
//
// Phase 5:全宽底栏 → Centered Card on Cream(design-system.md §5.2)
// - 卡片居中(~30% from top),不占全宽
// - 背景 scrim + blur
// - 卡内 logo + tagline + 文本 + Privacy link + Accept/Reject 按钮
// - 保留 __mm_consent_banner__ ID + 默认文案(test 依赖)+ text override API
import { t } from './i18n';

const BANNER_ID = '__mm_consent_banner__';
const SCRIM_ID = '__mm_consent_banner_scrim__';

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

  // === Scrim backdrop ===
  const scrim = document.createElement('div');
  scrim.id = SCRIM_ID;
  scrim.setAttribute('data-pinconsole', 'consent-scrim');
  scrim.style.cssText = [
    'position: fixed',
    'top: 0',
    'left: 0',
    'width: 100%',
    'height: 100%',
    `background: var(--pinconsole-scrim, rgba(28, 25, 23, 0.4))`,
    'backdrop-filter: blur(4px)',
    '-webkit-backdrop-filter: blur(4px)',
    'z-index: 999999',
    'display: flex',
    'align-items: flex-start',
    'justify-content: center',
    'padding-top: 20vh',
    `font-family: var(--pinconsole-font-sans, system-ui, sans-serif)`,
    'animation: pinconsole-fade-in 0.18s ease-out',
  ].join(';');

  // === Card ===
  const card = document.createElement('div');
  card.id = BANNER_ID;
  card.setAttribute('data-pinconsole', 'consent-card');
  card.setAttribute('role', 'dialog');
  card.setAttribute('aria-modal', 'true');
  card.style.cssText = [
    'background: var(--pinconsole-color-bg-surface, #fff)',
    'color: var(--pinconsole-color-text-primary, #1c1917)',
    `border: 1px solid var(--pinconsole-color-border-default, #e7e5e4)`,
    'border-radius: 16px',
    'padding: 32px',
    'width: 100%',
    'max-width: 440px',
    'margin: 0 16px',
    'box-shadow: 0 8px 16px rgba(28, 25, 23, 0.08), 0 4px 8px rgba(28, 25, 23, 0.05)',
    'display: flex',
    'flex-direction: column',
    'gap: 16px',
  ].join(';');

  // === Brand(logo + tagline) ===
  const brand = document.createElement('div');
  brand.style.cssText = 'display: flex; flex-direction: column; gap: 4px; align-items: flex-start';
  const wordmark = document.createElement('div');
  wordmark.textContent = t('consent_wordmark');
  wordmark.style.cssText =
    'font-size: 17px; font-weight: 600; letter-spacing: -0.01em; color: var(--pinconsole-color-accent-default, #0f766e)';
  const tagline = document.createElement('div');
  tagline.textContent = t('consent_tagline');
  tagline.style.cssText = 'font-size: 13px; color: var(--pinconsole-color-text-muted, #78716c)';
  brand.appendChild(wordmark);
  brand.appendChild(tagline);
  card.appendChild(brand);

  // === Text content ===
  const content = document.createElement('div');
  content.style.cssText = 'display: flex; flex-direction: column; gap: 8px';
  const title = document.createElement('strong');
  title.textContent = text.title;
  title.style.cssText =
    'font-size: 15px; font-weight: 600; color: var(--pinconsole-color-text-primary, #1c1917)';
  const body = document.createElement('span');
  body.textContent = text.body;
  body.style.cssText =
    'font-size: 13px; line-height: 1.55; color: var(--pinconsole-color-text-secondary, #57534e)';
  content.appendChild(title);
  content.appendChild(body);
  card.appendChild(content);

  // === Privacy link ===
  const privacyRow = document.createElement('div');
  const privacyLink = document.createElement('span');
  privacyLink.textContent = t('consent_privacy_link');
  privacyLink.style.cssText =
    'font-size: 12px; color: var(--pinconsole-color-accent-default, #0f766e); text-decoration: underline; cursor: pointer';
  privacyRow.appendChild(privacyLink);
  card.appendChild(privacyRow);

  // === Buttons ===
  const buttonRow = document.createElement('div');
  buttonRow.style.cssText = 'display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px';

  const rejectBtn = document.createElement('button');
  rejectBtn.textContent = text.reject;
  rejectBtn.style.cssText = [
    'padding: 8px 18px',
    'font-size: 14px',
    'font-weight: 500',
    'color: var(--pinconsole-color-text-secondary, #57534e)',
    'background: transparent',
    `border: 1px solid var(--pinconsole-color-border-default, #e7e5e4)`,
    'border-radius: 8px',
    'cursor: pointer',
    'transition: background 0.12s ease-out',
    `font-family: var(--pinconsole-font-sans, system-ui, sans-serif)`,
  ].join(';');
  rejectBtn.onmouseenter = () => {
    rejectBtn.style.background = 'var(--pinconsole-color-bg-subtle, #f5f1ec)';
  };
  rejectBtn.onmouseleave = () => {
    rejectBtn.style.background = 'transparent';
  };
  rejectBtn.onclick = () => {
    removeConsentBanner();
    opts.onReject();
  };

  const acceptBtn = document.createElement('button');
  acceptBtn.textContent = text.accept;
  acceptBtn.style.cssText = [
    'padding: 8px 20px',
    'font-size: 14px',
    'font-weight: 600',
    'color: var(--pinconsole-color-accent-on, #fff)',
    'background: var(--pinconsole-color-accent-default, #0f766e)',
    'border: 1px solid var(--pinconsole-color-accent-default, #0f766e)',
    'border-radius: 8px',
    'cursor: pointer',
    'transition: background 0.12s ease-out',
    `font-family: var(--pinconsole-font-sans, system-ui, sans-serif)`,
  ].join(';');
  acceptBtn.onmouseenter = () => {
    acceptBtn.style.background = 'var(--pinconsole-color-accent-hover, #115e59)';
  };
  acceptBtn.onmouseleave = () => {
    acceptBtn.style.background = 'var(--pinconsole-color-accent-default, #0f766e)';
  };
  acceptBtn.onclick = () => {
    removeConsentBanner();
    opts.onAccept();
  };

  buttonRow.appendChild(rejectBtn);
  buttonRow.appendChild(acceptBtn);
  card.appendChild(buttonRow);

  scrim.appendChild(card);
  document.body.appendChild(scrim);

  // 自动聚焦 accept 按钮(键盘可访问)
  setTimeout(() => acceptBtn.focus(), 100);

  // 注入 keyframe(zoom 快,无外部 CSS 文件依赖)
  ensureFadeInKeyframe();
}

export function removeConsentBanner(): void {
  const scrim = document.getElementById(SCRIM_ID);
  if (scrim && scrim.parentNode) scrim.parentNode.removeChild(scrim);
  // 兼容旧代码直接通过 banner ID 查找的场景
  const el = document.getElementById(BANNER_ID);
  if (el && el.parentNode) el.parentNode.removeChild(el);
}

// fade-in keyframe 一次注入(sdk 全局只一份)
let fadeInInjected = false;
function ensureFadeInKeyframe(): void {
  if (fadeInInjected) return;
  if (typeof document === 'undefined') return;
  const style = document.createElement('style');
  style.setAttribute('data-pinconsole', 'keyframes');
  style.textContent = `
    @keyframes pinconsole-fade-in {
      from { opacity: 0; transform: translateY(-8px); }
      to { opacity: 1; transform: translateY(0); }
    }
    @media (prefers-reduced-motion: reduce) {
      @keyframes pinconsole-fade-in {
        from { opacity: 0; }
        to { opacity: 1; }
      }
    }
  `;
  document.head.appendChild(style);
  fadeInInjected = true;
}
