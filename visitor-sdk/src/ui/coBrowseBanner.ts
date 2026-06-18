// 1l-privacy-gdpr:co-browsing 接管横幅
// 当运营接管访客时显示,提供透明度(GDPR Art.22 自动化决策)。
// 访客可点退出按钮立即释放控制权。
const BANNER_ID = '__mm_cobrowse_banner__';

export interface CoBrowseBannerOptions {
  operatorName?: string; // 运营名(可选,服务端目前不传)
  onExit: () => void;    // 访客点退出按钮触发
}

export function showCoBrowseBanner(opts: CoBrowseBannerOptions): void {
  removeCoBrowseBanner();

  const isZh = /^zh\b/i.test(navigator.language);
  const operatorLabel = opts.operatorName ?? (isZh ? '运营' : 'Operator');

  const banner = document.createElement('div');
  banner.id = BANNER_ID;
  banner.style.cssText = [
    'position: fixed', 'top: 0', 'left: 0', 'width: 100%',
    'background: #fff3cd', 'color: #856404',
    'padding: 8px 24px', 'z-index: 999999',
    'display: flex', 'align-items: center', 'justify-content: space-between',
    'font-family: system-ui, sans-serif', 'font-size: 14px',
    'border-bottom: 1px solid #ffe082',
  ].join(';');

  const label = document.createElement('span');
  label.textContent = isZh
    ? `🔔 ${operatorLabel} 正在协助您,可看到您的页面操作。`
    : `🔔 ${operatorLabel} is assisting you and can see your page actions.`;
  banner.appendChild(label);

  const exitBtn = document.createElement('button');
  exitBtn.textContent = isZh ? '退出协助' : 'End Assistance';
  exitBtn.style.cssText = 'background: #dc3545; color: #fff; border: none; padding: 4px 16px; border-radius: 4px; cursor: pointer; font-size: 13px';
  exitBtn.onclick = () => {
    opts.onExit();
  };
  banner.appendChild(exitBtn);

  document.body.prepend(banner);

  // 推下页面顶部内容,避免 banner 遮挡
  document.body.style.paddingTop = '40px';
}

export function removeCoBrowseBanner(): void {
  const el = document.getElementById(BANNER_ID);
  if (el && el.parentNode) el.parentNode.removeChild(el);
  document.body.style.paddingTop = '';
}
