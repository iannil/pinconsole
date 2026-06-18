// WebSocket transport：连接、重连、缓冲、事件发送
// 详见 docs/progress/2026-06-17-slice-1b-spec.md §SDK 重连策略

import { encode, decode } from '@msgpack/msgpack';
import type {
  Envelope,
  HelloPayload,
  AckPayload,
  ErrorPayload,
  PresencePayload,
} from '../proto/envelope';
import { PROTOCOL_VERSION } from '../proto/envelope';
import { sdkLogger, generateTraceId } from '../logging';

export interface TransportOptions {
  /** WS 端点 URL，如 ws://localhost:8080/ws/visitor */
  endpoint: string;
  /** Hello payload（连接成功后第一条消息） */
  hello: HelloPayload;
  /** 消息接收回调 */
  onMessage?: (env: Envelope) => void;
  /** 错误回调 */
  onError?: (err: Error) => void;
  /** 连接状态变化回调 */
  onStatusChange?: (status: TransportStatus) => void;
  /** 本地缓冲上限（默认 1000） */
  bufferMaxEvents?: number;
  /** 最大重连退避（默认 30s） */
  reconnectMaxBackoffMs?: number;
  /** 初始重连退避（默认 1s） */
  reconnectInitialMs?: number;
}

export type TransportStatus = 'connecting' | 'connected' | 'reconnecting' | 'closed';

export class WSTransport {
  private opts: TransportOptions;
  private ws: WebSocket | null = null;
  private status: TransportStatus = 'closed';
  private buffer: Uint8Array[] = [];
  private bufferBytes = 0;
  private reconnectAttempt = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private closed = false;
  private helloAcked = false;

  constructor(opts: TransportOptions) {
    this.opts = opts;
  }

  /** 启动 transport（建立首次连接） */
  start(): void {
    this.closed = false;
    this.connect();
  }

  /** 主动关闭（不再重连） */
  close(): void {
    this.closed = true;
    this.clearReconnectTimer();
    if (this.ws) {
      try {
        this.ws.close(1000, 'client_close');
      } catch {
        // ignore
      }
      this.ws = null;
    }
    this.setStatus('closed');
  }

  /**
   * 1f：发送 presence.navigated 通知（访客即将跳转到新页面）。
   * 服务端收到后广播给 admin，admin 自动重订阅新 session。
   */
  sendNavigated(): void {
    const env: Envelope = {
      v: PROTOCOL_VERSION,
      type: 'presence',
      ts: Date.now(),
      session_id: this.opts.hello.session_id,
      trace_id: generateTraceId(),
      payload: {
        event: 'navigated',
        session_id: this.opts.hello.session_id,
        visitor_id: this.opts.hello.visitor_id,
        fingerprint: this.opts.hello.visitor_id,
        started_at: Date.now(),
      } as PresencePayload,
    };
    const bytes = encode(env);
    // 尽力发送；连接已断时丢弃
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      try {
        this.ws.send(bytes);
      } catch {
        // ignore
      }
    }
  }

  /** 发送一条事件 envelope(自动缓冲+异步重试,1m:trace_id 生成) */
  sendEvent(payload: unknown): void {
    const env: Envelope = {
      v: PROTOCOL_VERSION,
      type: 'event',
      ts: Date.now(),
      session_id: this.opts.hello.session_id,
      trace_id: generateTraceId(),
      payload,
    };
    const bytes = encode(env);
    this.enqueueOrSend(bytes);
  }

  /** 批量发送事件:把一组 EventPayload 打包成 array,单 envelope 上行(1m:trace_id)。 */
  sendBatch(events: unknown[]): void {
    if (events.length === 0) return;
    const env: Envelope = {
      v: PROTOCOL_VERSION,
      type: 'event',
      ts: Date.now(),
      session_id: this.opts.hello.session_id,
      trace_id: generateTraceId(),
      payload: events,
    };
    const bytes = encode(env);
    this.enqueueOrSend(bytes);
  }

  private enqueueOrSend(bytes: Uint8Array): void {
    const max = this.opts.bufferMaxEvents ?? 1000;
    if (this.buffer.length >= max) {
      // 缓冲满,丢弃最旧
      const oldest = this.buffer.shift();
      if (oldest) this.bufferBytes -= oldest.length;
      sdkLogger.warn('buffer_full_drop_oldest', { buffer_size: this.buffer.length, max });
    }
    if (this.status === 'connected' && this.helloAcked) {
      // 直发
      this.rawSend(bytes);
    } else {
      // 入缓冲
      this.buffer.push(bytes);
      this.bufferBytes += bytes.length;
    }
  }

  private flushBuffer(): void {
    if (this.buffer.length === 0) return;
    const items = this.buffer;
    this.buffer = [];
    this.bufferBytes = 0;
    for (const b of items) {
      this.rawSend(b);
    }
  }

  private rawSend(bytes: Uint8Array): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      // 重新入缓冲
      this.buffer.push(bytes);
      this.bufferBytes += bytes.length;
      return;
    }
    try {
      this.ws.send(bytes);
    } catch (e) {
      console.warn('[marketing-monitor] send failed, re-buffering', e);
      this.buffer.push(bytes);
      this.bufferBytes += bytes.length;
    }
  }

  private connect(): void {
    this.setStatus(this.reconnectAttempt === 0 ? 'connecting' : 'reconnecting');
    let ws: WebSocket;
    try {
      ws = new WebSocket(this.opts.endpoint, []);
      // 关键：以 ArrayBuffer 接收，避免 Blob 解码开销
      ws.binaryType = 'arraybuffer';
    } catch (e) {
      this.opts.onError?.(e as Error);
      this.scheduleReconnect();
      return;
    }
    this.ws = ws;

    ws.onopen = () => {
      this.reconnectAttempt = 0;
      this.sendHello();
    };

    ws.onmessage = (ev: MessageEvent) => {
      if (!(ev.data instanceof ArrayBuffer)) return;
      try {
        const env = decode(new Uint8Array(ev.data)) as Envelope;
        this.handleIncoming(env);
      } catch (e) {
        console.warn('[marketing-monitor] decode failed', e);
      }
    };

    ws.onerror = () => {
      // onclose 会处理重连；这里只通知
      this.opts.onError?.(new Error('websocket error'));
    };

    ws.onclose = () => {
      this.helloAcked = false;
      this.ws = null;
      if (this.closed) {
        this.setStatus('closed');
        return;
      }
      this.scheduleReconnect();
    };
  }

  private sendHello(): void {
    const env: Envelope<HelloPayload> = {
      v: PROTOCOL_VERSION,
      type: 'hello',
      ts: Date.now(),
      payload: this.opts.hello,
    };
    const bytes = encode(env);
    try {
      this.ws?.send(bytes);
    } catch (e) {
      console.warn('[marketing-monitor] hello send failed', e);
    }
  }

  private handleIncoming(env: Envelope): void {
    switch (env.type) {
      case 'ack': {
        const payload = env.payload as AckPayload;
        if (payload?.ok) {
          this.helloAcked = true;
          this.setStatus('connected');
          this.flushBuffer();
        } else {
          this.opts.onError?.(new Error('server rejected hello'));
          this.close();
        }
        break;
      }
      case 'error': {
        const payload = env.payload as ErrorPayload;
        this.opts.onError?.(new Error(`${payload?.code}: ${payload?.message ?? ''}`));
        break;
      }
      default:
        this.opts.onMessage?.(env);
    }
  }

  private scheduleReconnect(): void {
    if (this.closed) return;
    this.clearReconnectTimer();
    const initial = this.opts.reconnectInitialMs ?? 1000;
    const max = this.opts.reconnectMaxBackoffMs ?? 30_000;
    const delay = Math.min(initial * 2 ** this.reconnectAttempt, max);
    this.reconnectAttempt++;
    this.setStatus('reconnecting');
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, delay);
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private setStatus(s: TransportStatus): void {
    if (this.status === s) return;
    this.status = s;
    this.opts.onStatusChange?.(s);
  }

  get currentStatus(): TransportStatus {
    return this.status;
  }
  get bufferLength(): number {
    return this.buffer.length;
  }
  get bufferSizeBytes(): number {
    return this.bufferBytes;
  }
}
