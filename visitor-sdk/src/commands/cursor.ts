// 运营光标 SVG 渲染（访客端显示运营员鼠标位置）
// 详见 docs/progress/2026-06-17-slice-1e-spec.md §访客端高亮渲染

const CURSOR_ID = '__mm_operator_cursor__';
const CURSOR_NAME = '__mm_operator_name__';

interface CursorStyle {
  color: string;
  size: number;
}

const DEFAULT_STYLE: CursorStyle = {
  color: '#409eff',
  size: 28,
};

/**
 * OperatorCursor 在访客页面创建一个固定定位的 SVG 圆点 + 运营名字标签。
 * 接收 cursor_highlight 命令后更新位置。
 * z-index: 999999, pointer-events: none（避免拦截访客点击）
 */
export class OperatorCursor {
  private container: HTMLDivElement | null = null;
  private svg: SVGSVGElement | null = null;
  private label: HTMLDivElement | null = null;
  private style: CursorStyle = DEFAULT_STYLE;

  /** 创建并插入 DOM。幂等。 */
  show(): void {
    if (this.container) return;
    this.container = document.createElement('div');
    this.container.id = CURSOR_ID;
    this.container.style.cssText = [
      'position: fixed',
      'top: 0',
      'left: 0',
      'pointer-events: none',
      'z-index: 999999',
      'transform: translate(-9999px, -9999px)', // 初始在屏外
      'transition: transform 0.05s linear', // 平滑跟随
    ].join(';');

    this.svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    this.svg.setAttribute('width', String(this.style.size));
    this.svg.setAttribute('height', String(this.style.size));
    this.svg.setAttribute('viewBox', '0 0 28 28');
    const circle = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
    circle.setAttribute('cx', '14');
    circle.setAttribute('cy', '14');
    circle.setAttribute('r', '10');
    circle.setAttribute('fill', this.style.color);
    circle.setAttribute('stroke', '#fff');
    circle.setAttribute('stroke-width', '2');
    circle.setAttribute('opacity', '0.8');
    this.svg.appendChild(circle);
    this.container.appendChild(this.svg);

    this.label = document.createElement('div');
    this.label.id = CURSOR_NAME;
    this.label.style.cssText = [
      `position: absolute`,
      `top: ${this.style.size - 4}px`,
      `left: ${this.style.size - 4}px`,
      'padding: 2px 6px',
      'background: rgba(64, 158, 255, 0.9)',
      'color: #fff',
      'font-size: 11px',
      'font-family: system-ui, sans-serif',
      'border-radius: 3px',
      'white-space: nowrap',
    ].join(';');
    this.label.textContent = '';
    this.container.appendChild(this.label);

    document.body.appendChild(this.container);
  }

  /** 移动到指定坐标（视口坐标）。 */
  moveTo(x: number, y: number, name?: string): void {
    if (!this.container) this.show();
    if (this.container) {
      this.container.style.transform = `translate(${x}px, ${y}px)`;
    }
    if (this.label && name !== undefined) {
      this.label.textContent = name;
    }
  }

  /** 隐藏（运营释放控制或紧急退出时调用）。 */
  hide(): void {
    if (this.container) {
      this.container.style.transform = 'translate(-9999px, -9999px)';
    }
  }

  /** 完全销毁 DOM。 */
  destroy(): void {
    if (this.container && this.container.parentNode) {
      this.container.parentNode.removeChild(this.container);
    }
    this.container = null;
    this.svg = null;
    this.label = null;
  }
}
