// WebSocket composable：连接 /ws/operator，订阅 session，接收事件
// 详见 docs/progress/2026-06-17-slice-1b-spec.md §admin WS 客户端架构

import { ref, onUnmounted } from 'vue';
import { encode, decode } from '@msgpack/msgpack';
import type { Envelope, PresencePayload, SubscribePayload } from '../proto/envelope';
import { PROTOCOL_VERSION } from '../proto/envelope';

export type WsStatus = 'idle' | 'connecting' | 'connected' | 'reconnecting' | 'closed';

export interface VisitorPresence {
  event: 'online' | 'offline' | 'navigated';
  sessionId: string;
  visitorId: string;
  fingerprint: string;
  startedAt: number;
  // 1f：navigated 事件
  oldSessionId?: string;
  newSessionId?: string;
}

export interface IncomingEvent {
  sessionId: string;
  envelope: Envelope;
}

export interface UseWsOptions {
  endpoint: string;
  onPresence?: (p: PresencePayload) => void;
  onEvent?: (e: IncomingEvent) => void;
  onStatusChange?: (s: WsStatus) => void;
}

export function useWs(opts: UseWsOptions) {
  const status = ref<WsStatus>('idle');
  const error = ref<string | null>(null);

  let ws: WebSocket | null = null;
  let reconnectAttempt = 0;
  let closedByUser = false;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  // 当前订阅的 sessionID 集合（重连后重新订阅）
  const subscribedSessions = new Set<string>();

  function connect() {
    closedByUser = false;
    setStatus(reconnectAttempt === 0 ? 'connecting' : 'reconnecting');
    let socket: WebSocket;
    try {
      socket = new WebSocket(opts.endpoint);
      socket.binaryType = 'arraybuffer';
    } catch (e) {
      error.value = String(e);
      scheduleReconnect();
      return;
    }
    ws = socket;

    socket.onopen = () => {
      reconnectAttempt = 0;
      error.value = null;
      setStatus('connected');
      // 重连后重新订阅
      for (const sid of subscribedSessions) {
        sendSubscribe(sid);
      }
    };

    socket.onmessage = (ev) => {
      if (!(ev.data instanceof ArrayBuffer)) return;
      try {
        const env = decode(new Uint8Array(ev.data)) as Envelope;
        handle(env);
      } catch (e) {
        console.warn('[admin ws] decode failed', e);
      }
    };

    socket.onerror = () => {
      error.value = 'websocket error';
    };

    socket.onclose = () => {
      ws = null;
      if (closedByUser) {
        setStatus('closed');
        return;
      }
      scheduleReconnect();
    };
  }

  function handle(env: Envelope) {
    switch (env.type) {
      case 'presence':
        opts.onPresence?.(env.payload as PresencePayload);
        break;
      case 'event':
        if (env.session_id) {
          opts.onEvent?.({ sessionId: env.session_id, envelope: env });
        }
        break;
      default:
        // ack / error / others：暂不处理
        break;
    }
  }

  function subscribe(sessionId: string) {
    subscribedSessions.add(sessionId);
    sendSubscribe(sessionId);
  }

  function unsubscribe(sessionId: string) {
    subscribedSessions.delete(sessionId);
    if (ws?.readyState === WebSocket.OPEN) {
      sendRaw({
        v: PROTOCOL_VERSION,
        type: 'unsubscribe',
        ts: Date.now(),
        payload: { session_id: sessionId } satisfies SubscribePayload,
      });
    }
  }

  function sendSubscribe(sessionId: string) {
    if (ws?.readyState !== WebSocket.OPEN) return;
    sendRaw({
      v: PROTOCOL_VERSION,
      type: 'subscribe',
      ts: Date.now(),
      payload: { session_id: sessionId } satisfies SubscribePayload,
    });
  }

  function sendRaw(env: Envelope) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    try {
      ws.send(encode(env));
    } catch (e) {
      console.warn('[admin ws] send failed', e);
    }
  }

  function scheduleReconnect() {
    if (closedByUser) return;
    if (reconnectTimer) clearTimeout(reconnectTimer);
    const initial = 1000;
    const max = 30_000;
    const delay = Math.min(initial * 2 ** reconnectAttempt, max);
    reconnectAttempt++;
    setStatus('reconnecting');
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null;
      connect();
    }, delay);
  }

  function setStatus(s: WsStatus) {
    status.value = s;
    opts.onStatusChange?.(s);
  }

  function close() {
    closedByUser = true;
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    if (ws) {
      try {
        ws.close(1000, 'client_close');
      } catch {
        // ignore
      }
      ws = null;
    }
    setStatus('closed');
  }

  onUnmounted(() => close());

  return {
    status,
    error,
    connect,
    close,
    subscribe,
    unsubscribe,
  };
}
