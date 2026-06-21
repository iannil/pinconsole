// TS-3 切片补测:sessions API client + time formatRelative。
//
// 测试策略:mock apiJson 模块,验证各函数的 URL 拼接 + 参数序列化。
import { describe, it, expect, vi, beforeEach } from 'vitest';

// mock apiJson 模块(避免真实 fetch)
vi.mock('../src/api/client', () => ({
  apiJson: vi.fn(),
}));

import { apiJson } from '../src/api/client';
import {
  listEndedSessions,
  getSessionReplay,
  sendCommand,
  listMessages,
  sendMessage,
} from '../src/api/sessions';

describe('sessions API client', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('listEndedSessions 默认参数', async () => {
    (apiJson as any).mockResolvedValueOnce({
      data: { sessions: [], total: 0 },
    });
    const got = await listEndedSessions();
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/ended?since=24h&limit=200',
    );
    expect(got.total).toBe(0);
  });

  it('listEndedSessions 自定义 since/limit', async () => {
    (apiJson as any).mockResolvedValueOnce({
      data: { sessions: [{ session_id: 's1' }], total: 1 },
    });
    await listEndedSessions('7d', 50);
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/ended?since=7d&limit=50',
    );
  });

  it('listEndedSessions 30d range', async () => {
    (apiJson as any).mockResolvedValueOnce({ data: { sessions: [], total: 0 } });
    await listEndedSessions('30d');
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/ended?since=30d&limit=200',
    );
  });

  it('getSessionReplay 默认 offset/limit', async () => {
    (apiJson as any).mockResolvedValueOnce({
      data: { events: [], total: 0, offset: 0, limit: 10000, has_more: false, session_id: 's1' },
    });
    await getSessionReplay('s1');
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/s1/replay?offset=0&limit=10000',
    );
  });

  it('getSessionReplay 自定义 offset/limit', async () => {
    (apiJson as any).mockResolvedValueOnce({
      data: { events: [], total: 0, offset: 100, limit: 50, has_more: false, session_id: 's1' },
    });
    await getSessionReplay('s1', 100, 50);
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/s1/replay?offset=100&limit=50',
    );
  });

  it('getSessionReplay encodeURIComponent 特殊字符', async () => {
    (apiJson as any).mockResolvedValueOnce({
      data: { events: [], total: 0, offset: 0, limit: 10000, has_more: false, session_id: 's/special' },
    });
    await getSessionReplay('s/special');
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/s%2Fspecial/replay?offset=0&limit=10000',
    );
  });

  it('sendCommand POST + body', async () => {
    (apiJson as any).mockResolvedValueOnce({ data: { ok: true } });
    const got = await sendCommand('s1', 'cursor_highlight', { x: 100, y: 200 });
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/s1/command',
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type: 'cursor_highlight', payload: { x: 100, y: 200 } }),
      },
    );
    expect(got.ok).toBe(true);
  });

  it('sendCommand 所有命令类型', async () => {
    const types = [
      'cursor_highlight', 'click', 'scroll', 'fill_input',
      'navigate', 'release_control', 'show_popup', 'chat_message',
    ] as const;
    for (const type of types) {
      (apiJson as any).mockResolvedValueOnce({ data: { ok: true } });
      await sendCommand('s1', type, {});
    }
    expect(apiJson).toHaveBeenCalledTimes(types.length);
  });

  it('listMessages 默认 sinceId', async () => {
    (apiJson as any).mockResolvedValueOnce({ data: { messages: [] } });
    await listMessages('s1');
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/s1/messages?since_id=0',
    );
  });

  it('listMessages 自定义 sinceId', async () => {
    (apiJson as any).mockResolvedValueOnce({
      data: { messages: [{ id: 5 }] },
    });
    await listMessages('s1', 10);
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/s1/messages?since_id=10',
    );
  });

  it('sendMessage POST 默认 sender=operator', async () => {
    (apiJson as any).mockResolvedValueOnce({
      data: { id: 1, sender: 'operator', content: 'hi', created_at: 1 },
    });
    await sendMessage('s1', 'hi');
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/s1/messages',
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: 'hi', sender: 'operator' }),
      },
    );
  });

  it('sendMessage 显式 sender=visitor', async () => {
    (apiJson as any).mockResolvedValueOnce({
      data: { id: 1, sender: 'visitor', content: 'reply', created_at: 2 },
    });
    await sendMessage('s1', 'reply', 'visitor');
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/s1/messages',
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: 'reply', sender: 'visitor' }),
      },
    );
  });
});
