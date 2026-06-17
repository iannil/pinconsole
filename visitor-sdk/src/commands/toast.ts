// 运营代填 Toast（右上角浮动提示）
// 详见 docs/progress/2026-06-17-slice-1f-spec.md §访客端代填提示

const TOAST_ID = '__mm_toast__';
const TOAST_TIMEOUT_MS = 5000; // 与临时锁定时长一致

/**
 * OperatorToast 在访客页面右上角显示浮动 toast。
 * 当 fill_input 触发时显示 "运营员 正在代为填写"。
 */
export class OperatorToast {
  private container: HTMLDivElement | null = null;
  private timer: ReturnType<typeof setTimeout> | null = null;

  /** 显示一条 toast。message 为空则隐藏。 */
  show(operatorName: string, message: string): void {
    if (!this.container) this.ensureContainer();
    if (!this.container) return;
    this.container.textContent = `${operatorName} ${message}`;
    this.container.style.transform = 'translateX(0)';
    this.container.style.opacity = '1';

    if (this.timer) clearTimeout(this.timer);
    this.timer = setTimeout(() => this.hide(), TOAST_TIMEOUT_MS);
  }

  hide(): void {
    if (!this.container) return;
    this.container.style.transform = 'translateX(120%)';
    this.container.style.opacity = '0';
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
  }

  destroy(): void {
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    if (this.container && this.container.parentNode) {
      this.container.parentNode.removeChild(this.container);
    }
    this.container = null;
  }

  private ensureContainer(): void {
    if (this.container) return;
    if (!document.body) return;
    this.container = document.createElement('div');
    this.container.id = TOAST_ID;
    this.container.style.cssText = [
      'position: fixed',
      'top: 16px',
      'right: 16px',
      'padding: 10px 16px',
      'background: rgba(64, 158, 255, 0.95)',
      'color: #fff',
      'font-family: system-ui, sans-serif',
      'font-size: 13px',
      'border-radius: 4px',
      'box-shadow: 0 4px 12px rgba(0,0,0,0.15)',
      'z-index: 999999',
      'pointer-events: none',
      'transform: translateX(120%)',
      'opacity: 0',
      'transition: transform 0.3s, opacity 0.3s',
      'max-width: 280px',
    ].join(';');
    document.body.appendChild(this.container);
  }
}
