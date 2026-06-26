// 命令处理器：接收 envelope.command → 执行 → 临时锁定 → ESC 监听
// 详见 docs/progress/2026-06-17-slice-1e-spec.md + 1f-spec.md（Toast）

import type { Envelope } from '@pinconsole/proto';
import type { CommandPayload } from '@pinconsole/proto';
import { OperatorCursor } from './cursor';
import { OperatorToast } from './toast';
import { sdkLogger } from '../logging';
import { t } from '../ui/i18n';

const ESC_TIMEOUT_MS = 1000; // 三连 ESC 在 1s 内
const FILL_LOCK_MS = 5000; // 临时锁定 5s

export interface ChatMessage {
  id: number;
  sender: 'operator' | 'visitor';
  content: string;
  created_at: number;
}

export interface CommandHandlerOptions {
  debug?: boolean;
  onReleased?: () => void;
  onChatMessage?: (msg: ChatMessage) => void;
  /** 1l:co-browsing 接管开始(首个 cursor/click/fill 命令) → 显示横幅 */
  onControlStart?: () => void;
}

/**
 * CommandHandler 接收服务端下行的 command envelope 并执行。
 *
 * 5 类命令：
 *   - cursor_highlight: OperatorCursor.moveTo
 *   - click: NodeMap.get(nodeID).click()
 *   - scroll: window.scrollTo
 *   - fill_input: NodeMap.get(nodeID).value = value + 临时锁定
 *   - navigate: 同源校验 + window.location
 *   - release_control: 清理 + 通知
 *
 * ESC 三连或 Ctrl+Shift+X 触发 release_control 回调（紧急退出）。
 */
export class CommandHandler {
  private opts: Required<CommandHandlerOptions>;
  private cursor: OperatorCursor;
  private toast: OperatorToast;
  private active = false;
  private lockedInputs: Set<HTMLElement> = new Set();
  private escTimestamps: number[] = [];
  private keyHandler: ((e: KeyboardEvent) => void) | null = null;

  constructor(opts: CommandHandlerOptions = {}) {
    this.opts = {
      debug: opts.debug ?? false,
      onReleased: opts.onReleased ?? (() => {}),
      onChatMessage: opts.onChatMessage ?? (() => {}),
      onControlStart: opts.onControlStart ?? (() => {}),
    };
    this.cursor = new OperatorCursor();
    this.toast = new OperatorToast();
  }

  /** 启动：监听键盘 + 初始化 NodeMap。 */
  start(): void {
    if (this.active) return;
    this.active = true;
    this.attachKeyboard();
    if (this.opts.debug) {
      // eslint-disable-next-line no-console
      sdkLogger.debug('command_handler_started');
    }
  }

  stop(): void {
    this.active = false;
    this.cursor.destroy();
    this.toast.destroy();
    this.detachKeyboard();
    this.clearLocks();
  }

  /** 处理收到的 command envelope。 */
  handle(env: Envelope): void {
    if (env.type !== 'command') return;
    const cp = env.payload as CommandPayload | undefined;
    if (!cp) return;
    this.execute(cp);
  }

  private execute(cp: CommandPayload): void {
    switch (cp.type) {
      case 'cursor_highlight':
        if (cp.cursor) {
          this.opts.onControlStart();
          this.cursor.moveTo(cp.cursor.x, cp.cursor.y, cp.cursor.name);
        }
        break;
      case 'click':
        if (cp.click) {
          this.opts.onControlStart();
          this.doClick(cp.click.node_id);
        }
        break;
      case 'scroll':
        if (cp.scroll) {
          window.scrollTo(cp.scroll.x, cp.scroll.y);
        }
        break;
      case 'fill_input':
        if (cp.fill_input) {
          this.opts.onControlStart();
          this.doFill(cp.fill_input.node_id, cp.fill_input.value);
        }
        break;
      case 'navigate':
        if (cp.navigate) {
          this.opts.onControlStart();
          this.doNavigate(cp.navigate.url);
        }
        break;
      case 'release_control':
        this.cursor.hide();
        this.clearLocks();
        this.opts.onReleased();
        break;
      case 'show_popup':
        if (cp.popup) {
          import('../ui/popup').then(({ showPopup }) => showPopup(cp.popup!));
        }
        break;
      case 'chat_message':
        if (cp.chat && this.opts.onChatMessage) {
          this.opts.onChatMessage({
            id: cp.chat.message_id,
            sender: 'operator',
            content: cp.chat.content,
            created_at: cp.ts,
          });
        }
        break;
    }
  }

  private doClick(nodeID: number): void {
    const el = this.findElement(nodeID);
    if (!el) {
      this.log('click: node not found', nodeID);
      return;
    }
    try {
      (el as HTMLElement).click();
    } catch (e) {
      this.log('click failed', e);
    }
  }

  private doFill(nodeID: number, value: string): void {
    const el = this.findElement(nodeID);
    if (!el) {
      this.log('fill: node not found', nodeID);
      return;
    }
    if (!(el instanceof HTMLInputElement) && !(el instanceof HTMLTextAreaElement)) {
      this.log('fill: not input/textarea', el.tagName);
      return;
    }
    // 临时锁定：5s 内访客输入被覆盖
    this.lockInput(el);
    // 1f：Toast 提示
    const fieldName = (el as HTMLInputElement).name || (el as HTMLInputElement).placeholder || t('cobrowse_field_fallback');
    this.toast.show(t('cobrowse_operator_label'), t('cobrowse_fill_toast', undefined, { field: fieldName }));
    try {
      // React 等框架需用 native setter 才能触发 onChange
      const proto = el instanceof HTMLInputElement
        ? HTMLInputElement.prototype
        : HTMLTextAreaElement.prototype;
      const setter = Object.getOwnPropertyDescriptor(proto, 'value')?.set;
      if (setter) {
        setter.call(el, value);
      } else {
        el.value = value;
      }
      el.dispatchEvent(new Event('input', { bubbles: true }));
      el.dispatchEvent(new Event('change', { bubbles: true }));
    } catch (e) {
      this.log('fill failed', e);
    }
  }

  private doNavigate(url: string): void {
    // 服务端已校验同源/白名单。SDK 这里只执行
    try {
      window.location.href = url;
    } catch (e) {
      this.log('navigate failed', e);
    }
  }

  private lockInput(el: HTMLElement): void {
    this.lockedInputs.add(el);
    // 视觉提示：边框变蓝
    const origBorder = el.style.border;
    el.dataset.mmOrigBorder = origBorder;
    el.style.border = '2px solid #409eff';
    el.style.transition = 'border 0.2s';

    setTimeout(() => {
      el.style.border = el.dataset.mmOrigBorder ?? '';
      delete el.dataset.mmOrigBorder;
      this.lockedInputs.delete(el);
    }, FILL_LOCK_MS);
  }

  private clearLocks(): void {
    for (const el of this.lockedInputs) {
      el.style.border = el.dataset.mmOrigBorder ?? '';
      delete el.dataset.mmOrigBorder;
    }
    this.lockedInputs.clear();
  }

  private attachKeyboard(): void {
    this.keyHandler = (e: KeyboardEvent) => {
      // Ctrl+Shift+X：立即触发
      if (e.ctrlKey && e.shiftKey && (e.key === 'X' || e.key === 'x')) {
        e.preventDefault();
        this.opts.onReleased();
        return;
      }
      // ESC 三连（1s 内）
      if (e.key === 'Escape') {
        const now = Date.now();
        this.escTimestamps.push(now);
        // 只保留 1s 内的
        this.escTimestamps = this.escTimestamps.filter((t) => now - t < ESC_TIMEOUT_MS);
        if (this.escTimestamps.length >= 3) {
          this.escTimestamps = [];
          this.opts.onReleased();
        }
      }
    };
    window.addEventListener('keydown', this.keyHandler);
  }

  private detachKeyboard(): void {
    if (this.keyHandler) {
      window.removeEventListener('keydown', this.keyHandler);
      this.keyHandler = null;
    }
  }

  private findElement(nodeID: number): Element | null {
    return document.querySelector(`[data-rr-node-id="${nodeID}"]`);
  }

  private log(...args: unknown[]): void {
    if (this.opts.debug) {
      // eslint-disable-next-line no-console
      sdkLogger.debug('command_handler_log', { args: String(args) });
    }
  }
}
