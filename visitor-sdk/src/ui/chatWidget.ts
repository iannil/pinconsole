// 聊天浮动气泡 widget（1g）
// 右下角浮动，初始为小圆点（含未读数），点击展开

import { t } from './i18n';

const WIDGET_ID = '__mm_chat_widget__';

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
  private lastMessageId = 0;
  private expanded = false;

  constructor(callbacks: ChatWidgetCallbacks) {
    this.callbacks = callbacks;
  }

  show(): void {
    if (this.container) return;
    this.container = document.createElement('div');
    this.container.id = WIDGET_ID;
    this.container.style.cssText = 'position: fixed;bottom: 20px;right: 20px;z-index: 999997;font-family: system-ui,sans-serif';
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

  /** 接收一条新消息（来自 admin 或回声）。 */
  receiveMessage(msg: ChatMessage): void {
    if (msg.id <= this.lastMessageId) return;
    this.lastMessageId = msg.id;
    if (!this.expanded && msg.sender === 'operator') {
      this.unreadCount++;
      this.updateBubbleBadge();
    }
    if (this.expanded && this.messageList) {
      this.appendMessageDOM(msg);
      this.messageList.scrollTop = this.messageList.scrollHeight;
    }
  }

  private renderBubble(): void {
    this.bubble = document.createElement('div');
    this.bubble.style.cssText = [
      'width: 56px', 'height: 56px', 'border-radius: 50%',
      'background: #409eff', 'color: #fff',
      'display: flex', 'align-items: center', 'justify-content: center',
      'cursor: pointer', 'font-size: 24px', 'box-shadow: 0 4px 12px rgba(64,158,255,0.4)',
      'transition: transform 0.2s',
    ].join(';');
    this.bubble.textContent = '💬';
    this.bubble.onclick = () => this.togglePanel();
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
    this.panel.style.cssText = [
      'position: absolute', 'bottom: 70px', 'right: 0',
      'width: 320px', 'height: 400px',
      'background: #fff', 'border-radius: 8px',
      'box-shadow: 0 8px 32px rgba(0,0,0,0.15)',
      'display: flex', 'flex-direction: column', 'overflow: hidden',
    ].join(';');

    // header
    const header = document.createElement('div');
    header.style.cssText = 'padding: 12px 16px;background: #409eff;color: #fff;font-size: 14px;font-weight: 600';
    header.textContent = t('chat_header');
    this.panel.appendChild(header);

    // message list
    this.messageList = document.createElement('div');
    this.messageList.style.cssText = 'flex: 1;overflow-y: auto;padding: 8px 12px';
    this.panel.appendChild(this.messageList);

    // input bar
    const inputBar = document.createElement('div');
    inputBar.style.cssText = 'display:flex;padding: 8px;gap: 4px;border-top: 1px solid #ebeef5';
    this.input = document.createElement('input');
    this.input.type = 'text';
    this.input.placeholder = t('chat_input_placeholder');
    this.input.style.cssText = 'flex:1;padding: 6px 10px;border:1px solid #dcdfe6;border-radius:4px;font-size:13px;outline:none';
    const sendBtn = document.createElement('button');
    sendBtn.textContent = t('chat_send');
    sendBtn.style.cssText = 'padding: 6px 14px;background:#409eff;color:#fff;border:none;border-radius:4px;cursor:pointer;font-size:13px';
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
      const msgs = await this.callbacks.onFetchMessages(this.lastMessageId);
      for (const m of msgs) {
        this.receiveMessage(m);
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
    // 本地回声
    this.receiveMessage({ id: Date.now(), sender: 'visitor', content, created_at: Date.now() });
    if (this.messageList) {
      this.messageList.scrollTop = this.messageList.scrollHeight;
    }
  }

  private appendMessageDOM(msg: ChatMessage): void {
    if (!this.messageList) return;
    const row = document.createElement('div');
    row.style.cssText = `margin-bottom: 8px;display:flex;${msg.sender === 'visitor' ? 'justify-content:flex-end' : 'justify-content:flex-start'}`;
    const bubble = document.createElement('div');
    bubble.style.cssText = `max-width: 75%;padding: 6px 10px;border-radius: 8px;font-size: 13px;word-break: break-word;${msg.sender === 'visitor' ? 'background:#409eff;color:#fff' : 'background:#f5f7fa;color:#303133'}`;
    bubble.textContent = msg.content;
    row.appendChild(bubble);
    this.messageList.appendChild(row);
  }

  private updateBubbleBadge(): void {
    if (!this.bubble) return;
    const existing = this.bubble.querySelector('.badge');
    if (existing) existing.remove();
    if (this.unreadCount > 0) {
      const badge = document.createElement('span');
      badge.className = 'badge';
      badge.textContent = String(this.unreadCount);
      badge.style.cssText = 'position:absolute;top:-4px;right:-4px;background:#f56c6c;color:#fff;font-size:10px;width:18px;height:18px;border-radius:50%;display:flex;align-items:center;justify-content:center';
      this.bubble.style.position = 'relative';
      this.bubble.appendChild(badge);
    }
  }
}
