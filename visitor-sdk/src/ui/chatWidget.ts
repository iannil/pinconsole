// 聊天浮动气泡 widget（1g）
// 右下角浮动,初始为小圆点(含未读数),点击展开
//
// Phase 5:Calm Crafted 升级(design-system.md §5.4)
// - bubble:blue → accent-default + 内联 SVG speech icon(去 💬 emoji)
// - panel:white card → Calm surface + radius-lg + shadow-lg
// - 消息气泡:operator=accent / visitor=subtle(非对称圆角)
// - unread badge:红圆 → accent-danger pill
// - 保留 __mm_chat_widget__ ID + 内部状态逻辑(seenIds/renderedIds/fetchCursor)

import { t } from './i18n';

const WIDGET_ID = '__mm_chat_widget__';

// 内联 SVG speech-bubble icon(无 emoji,跨平台一致)
const CHAT_ICON_SVG = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 256 256" fill="currentColor" aria-hidden="true"><path d="M128,24A104,104,0,0,0,36.18,176.85L29.37,201a16,16,0,0,0,20.32,20l32.71-8.46A104,104,0,1,0,128,24Zm0,192a87.87,87.87,0,0,1-44.06-11.81,8,8,0,0,0-6.54-.67L48,200.16l6.2-21.79a8,8,0,0,0-1.06-7.22A88,88,0,1,1,128,216Z"/></svg>`;

// 内联 SVG send arrow icon
const SEND_ICON_SVG = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 256 256" fill="currentColor" aria-hidden="true"><path d="M223.4,122.81,35.4,26.21a16,16,0,0,0-22,19.78l.05.12L73.61,128,13.4,209.89l-.05.12a16,16,0,0,0,22.05,19.78l188-96.6a16,16,0,0,0,0-28.4ZM34.08,208.34,90.76,134.4a4,4,0,0,1,6.48,0l56.68,73.94ZM90.76,121.6,34.08,47.66,154,109.94a4,4,0,0,1,0,4.12L34.08,208.34,90.76,134.4a16,16,0,0,0,0-26.8Z"/></svg>`;

interface ChatMessage {
  id: number;
  sender: 'operator' | 'visitor';
  content: string;
  created_at: number;
}

export interface ChatWidgetCallbacks {
  onSend: (content: string) => void;
  onFetchMessages?: (sinceId: number) => Promise<ChatMessage[]>;
}

export class ChatWidget {
  private callbacks: ChatWidgetCallbacks;
  private container: HTMLDivElement | null = null;
  private bubble: HTMLDivElement | null = null;
  private panel: HTMLDivElement | null = null;
  private messageList: HTMLDivElement | null = null;
  private input: HTMLInputElement | null = null;
  private unreadCount = 0;
  // WS 层面去重:已 acknowledge 的 server 消息 id。
  // 防止同一条 WS 消息重复推送时,unread 被多次累加。
  private seenIds = new Set<number>();
  // DOM 层面去重:已渲染过的 server 消息 id。
  // 与 seenIds 分离,因为"WS 收起时收到"会进 seenIds 但不进 renderedIds;
  // 展开 fetchMessages 拉回时需识别"已 seen 但未 rendered"并补渲染。
  private renderedIds = new Set<number>();
  // fetchMessages 的 since_id cursor:只跟踪"已渲染"的最高 id,
  // 不被"已收到但未渲染"(WS push during collapsed)错误推进。
  private fetchCursor = 0;
  private expanded = false;

  constructor(callbacks: ChatWidgetCallbacks) {
    this.callbacks = callbacks;
  }

  show(): void {
    if (this.container) return;
    this.container = document.createElement('div');
    this.container.id = WIDGET_ID;
    this.container.setAttribute('data-pinconsole', 'chat-widget');
    this.container.style.cssText = [
      'position: fixed',
      'bottom: 20px',
      'right: 20px',
      'z-index: 999997',
      `font-family: var(--pinconsole-font-sans, system-ui, sans-serif)`,
    ].join(';');
    document.body.appendChild(this.container);
    this.renderBubble();
  }

  destroy(): void {
    if (this.container && this.container.parentNode) {
      this.container.parentNode.removeChild(this.container);
    }
    this.container = null;
    this.bubble = null;
    this.panel = null;
  }

  /** 接收一条新消息（来自 admin WS 推送或本地回声）。 */
  // id 协议:server 消息 id > 0(自增序列);本地回声用负 id(-Date.now())。
  // dedup 分两层:
  //   - seenIds:WS 层去重(避免重复推送时 unread 多次累加)
  //   - renderedIds:DOM 层去重(避免重复渲染)
  // cursor(fetchCursor)只随"已渲染"推进——避免 WS 收起时推送的消息把 cursor 撑大,
  // 导致展开 fetchMessages 拉不到该消息(永久丢失)。
  receiveMessage(msg: ChatMessage): void {
    if (msg.id > 0) {
      if (this.seenIds.has(msg.id)) return;
      this.seenIds.add(msg.id);
    }

    // 收起状态下,operator 消息只累加 unread,不渲染、不推进 cursor。
    // 消息会在用户展开时通过 fetchMessages(cursor 未变)重新拉回。
    if (!this.expanded && msg.sender === 'operator') {
      this.unreadCount++;
      this.updateBubbleBadge();
      return;
    }

    if (this.expanded && this.messageList) {
      if (msg.id > 0 && this.renderedIds.has(msg.id)) return;
      this.appendMessageDOM(msg);
      if (msg.id > 0) {
        this.renderedIds.add(msg.id);
        this.fetchCursor = Math.max(this.fetchCursor, msg.id);
      }
      this.messageList.scrollTop = this.messageList.scrollHeight;
    }
  }

  private renderBubble(): void {
    this.bubble = document.createElement('div');
    this.bubble.setAttribute('data-pinconsole', 'chat-bubble');
    this.bubble.setAttribute('role', 'button');
    this.bubble.setAttribute('tabindex', '0');
    this.bubble.setAttribute('aria-label', t('chat_header'));
    this.bubble.style.cssText = [
      'width: 56px', 'height: 56px', 'border-radius: 50%',
      `background: var(--pinconsole-color-accent-default, #0f766e)`,
      `color: var(--pinconsole-color-accent-on, #fff)`,
      'display: flex', 'align-items: center', 'justify-content: center',
      'cursor: pointer',
      'box-shadow: 0 4px 12px rgba(28, 25, 23, 0.12), 0 2px 4px rgba(28, 25, 23, 0.06)',
      'transition: transform 0.12s ease-out, background 0.12s ease-out',
      'position: relative',
    ].join(';');
    // 内联 SVG speech icon(无 emoji)
    this.bubble.innerHTML = CHAT_ICON_SVG;
    this.bubble.onmouseenter = () => {
      if (this.bubble) this.bubble.style.transform = 'scale(1.05)';
    };
    this.bubble.onmouseleave = () => {
      if (this.bubble) this.bubble.style.transform = 'scale(1)';
    };
    this.bubble.onclick = () => this.togglePanel();
    this.bubble.onkeydown = (e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        this.togglePanel();
      }
    };
    this.container?.appendChild(this.bubble);
    this.updateBubbleBadge();
  }

  private togglePanel(): void {
    if (this.expanded) {
      this.collapsePanel();
    } else {
      this.expandPanel();
    }
  }

  private expandPanel(): void {
    this.expanded = true;
    this.unreadCount = 0;
    this.updateBubbleBadge();

    this.panel = document.createElement('div');
    this.panel.setAttribute('data-pinconsole', 'chat-panel');
    this.panel.setAttribute('role', 'dialog');
    this.panel.style.cssText = [
      'position: absolute', 'bottom: 70px', 'right: 0',
      'width: 340px', 'height: 460px',
      `background: var(--pinconsole-color-bg-surface, #fff)`,
      `color: var(--pinconsole-color-text-primary, #1c1917)`,
      'border-radius: 12px',
      'border: 1px solid var(--pinconsole-color-border-default, #e7e5e4)',
      'box-shadow: 0 8px 16px rgba(28, 25, 23, 0.08), 0 4px 8px rgba(28, 25, 23, 0.05)',
      'display: flex', 'flex-direction: column', 'overflow: hidden',
    ].join(';');

    // header(accent-subtle bg + accent-default 文本,代替全蓝 header)
    const header = document.createElement('div');
    header.style.cssText = [
      'padding: 12px 16px',
      `background: var(--pinconsole-color-accent-subtle, #ccfbf1)`,
      `color: var(--pinconsole-color-accent-default, #0f766e)`,
      'font-size: 14px', 'font-weight: 600',
      'border-bottom: 1px solid var(--pinconsole-color-border-default, #e7e5e4)',
      'display: flex', 'align-items: center', 'gap: 8px',
    ].join(';');
    // header 内联小 icon
    const headerIcon = document.createElement('span');
    headerIcon.style.cssText = 'display: inline-flex; align-items: center; width: 16px; height: 16px';
    headerIcon.innerHTML = CHAT_ICON_SVG.replace('width="24"', 'width="16"').replace('height="24"', 'height="16"');
    const headerText = document.createElement('span');
    headerText.textContent = t('chat_header');
    header.appendChild(headerIcon);
    header.appendChild(headerText);
    this.panel.appendChild(header);

    // message list
    this.messageList = document.createElement('div');
    this.messageList.style.cssText =
      'flex: 1;overflow-y: auto;padding: 12px 16px;display: flex; flex-direction: column; gap: 8px';
    this.panel.appendChild(this.messageList);

    // input bar
    const inputBar = document.createElement('div');
    inputBar.style.cssText =
      'display:flex;padding: 10px 12px;gap: 6px;border-top: 1px solid var(--pinconsole-color-border-default, #e7e5e4)';
    this.input = document.createElement('input');
    this.input.type = 'text';
    this.input.placeholder = t('chat_input_placeholder');
    this.input.style.cssText = [
      'flex:1',
      'padding: 8px 12px',
      `border: 1px solid var(--pinconsole-color-border-default, #e7e5e4)`,
      `border-radius: 8px`,
      `font-size: 13px`,
      `background: var(--pinconsole-color-bg-surface, #fff)`,
      `color: var(--pinconsole-color-text-primary, #1c1917)`,
      'outline: none',
      `font-family: var(--pinconsole-font-sans, system-ui, sans-serif)`,
      'transition: border-color 0.12s ease-out, box-shadow 0.12s ease-out',
    ].join(';');
    this.input.onfocus = () => {
      if (this.input) {
        this.input.style.borderColor = 'var(--pinconsole-color-accent-default, #0f766e)';
        this.input.style.boxShadow = '0 0 0 3px rgba(15, 118, 110, 0.25)';
      }
    };
    this.input.onblur = () => {
      if (this.input) {
        this.input.style.borderColor = 'var(--pinconsole-color-border-default, #e7e5e4)';
        this.input.style.boxShadow = 'none';
      }
    };
    const sendBtn = document.createElement('button');
    sendBtn.setAttribute('data-pinconsole', 'chat-send');
    sendBtn.setAttribute('aria-label', t('chat_send'));
    sendBtn.style.cssText = [
      'display: inline-flex',
      'align-items: center',
      'justify-content: center',
      'padding: 0 14px',
      'min-width: 36px',
      'min-height: 36px',
      `background: var(--pinconsole-color-accent-default, #0f766e)`,
      `color: var(--pinconsole-color-accent-on, #fff)`,
      'border: none',
      'border-radius: 8px',
      'cursor: pointer',
      'transition: background 0.12s ease-out',
      `font-family: var(--pinconsole-font-sans, system-ui, sans-serif)`,
    ].join(';');
    sendBtn.innerHTML = SEND_ICON_SVG;
    sendBtn.onmouseenter = () => {
      sendBtn.style.background = 'var(--pinconsole-color-accent-hover, #115e59)';
    };
    sendBtn.onmouseleave = () => {
      sendBtn.style.background = 'var(--pinconsole-color-accent-default, #0f766e)';
    };
    sendBtn.onclick = () => this.sendCurrent();
    this.input.onkeydown = (e) => { if (e.key === 'Enter') this.sendCurrent(); };
    inputBar.appendChild(this.input);
    inputBar.appendChild(sendBtn);
    this.panel.appendChild(inputBar);

    this.container?.appendChild(this.panel);

    // 拉取历史消息
    void this.fetchMessages();
  }

  private collapsePanel(): void {
    this.expanded = false;
    if (this.panel) {
      this.container?.removeChild(this.panel);
      this.panel = null;
      this.messageList = null;
      this.input = null;
    }
  }

  private async fetchMessages(): Promise<void> {
    if (!this.callbacks.onFetchMessages) return;
    try {
      const msgs = await this.callbacks.onFetchMessages(this.fetchCursor);
      for (const m of msgs) {
        if (m.id > 0 && this.renderedIds.has(m.id)) continue;
        this.appendMessageDOM(m);
        if (m.id > 0) {
          this.renderedIds.add(m.id);
          this.seenIds.add(m.id);
          this.fetchCursor = Math.max(this.fetchCursor, m.id);
        }
      }
      if (this.messageList) {
        this.messageList.scrollTop = this.messageList.scrollHeight;
      }
    } catch {
      // ignore
    }
  }

  private sendCurrent(): void {
    if (!this.input || !this.input.value.trim()) return;
    const content = this.input.value.trim();
    this.input.value = '';
    this.callbacks.onSend(content);
    // 本地回声:用负 id 标记(receiveMessage 跳过 dedup + 不更新 lastMessageId)。
    // 之前用 Date.now() 会撑大 lastMessageId,把后续 server 消息(id=2,3,...)
    // 错误地当成"已显示"丢弃。
    this.receiveMessage({ id: -Date.now(), sender: 'visitor', content, created_at: Date.now() });
    if (this.messageList) {
      this.messageList.scrollTop = this.messageList.scrollHeight;
    }
  }

  private appendMessageDOM(msg: ChatMessage): void {
    if (!this.messageList) return;
    const row = document.createElement('div');
    const isVisitor = msg.sender === 'visitor';
    row.style.cssText = `display:flex;${isVisitor ? 'justify-content:flex-end' : 'justify-content:flex-start'}`;
    const bubble = document.createElement('div');
    // 非对称圆角:visitor(右)右下小 / operator(左)左下小(对话流自然感)
    const bubbleRadius = isVisitor ? '12px 12px 4px 12px' : '12px 12px 12px 4px';
    bubble.style.cssText = [
      'max-width: 75%',
      'padding: 8px 12px',
      `border-radius: ${bubbleRadius}`,
      'font-size: 13px',
      'line-height: 1.4',
      'word-break: break-word',
      isVisitor
        ? `background: var(--pinconsole-color-accent-default, #0f766e); color: var(--pinconsole-color-accent-on, #fff)`
        : `background: var(--pinconsole-color-bg-subtle, #f5f1ec); color: var(--pinconsole-color-text-primary, #1c1917)`,
    ].join(';');
    bubble.textContent = msg.content;
    row.appendChild(bubble);
    this.messageList.appendChild(row);
  }

  private updateBubbleBadge(): void {
    if (!this.bubble) return;
    const existing = this.bubble.querySelector('[data-pinconsole-badge]');
    if (existing) existing.remove();
    if (this.unreadCount > 0) {
      const badge = document.createElement('span');
      // 注:.badge class 保留向后兼容(tests 用 document.querySelector('.badge') 查询)
      badge.className = 'badge';
      badge.setAttribute('data-pinconsole-badge', 'unread');
      badge.textContent = String(this.unreadCount);
      badge.style.cssText = [
        'position: absolute',
        'top: -4px',
        'right: -4px',
        `background: var(--pinconsole-color-danger, #b91c1c)`,
        'color: #fff',
        'font-size: 11px',
        'font-weight: 600',
        'min-width: 18px',
        'height: 18px',
        'padding: 0 5px',
        'border-radius: 9px',
        'display: flex',
        'align-items: center',
        'justify-content: center',
        `border: 2px solid var(--pinconsole-color-bg-surface, #fff)`,
        'box-shadow: 0 2px 4px rgba(28, 25, 23, 0.15)',
        'font-variant-numeric: tabular-nums',
        `font-family: var(--pinconsole-font-sans, system-ui, sans-serif)`,
      ].join(';');
      this.bubble.appendChild(badge);
    }
  }
}
