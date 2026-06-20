// 切片 1aa:Batch 单元测试
// 覆盖 push / flush / destroy + 阈值触发(50 events 或 100ms)+ timer 清理。

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { Batch } from '../src/batch';
import type { EventPayload } from '@pinconsole/proto';

const sampleEvent = (i: number): EventPayload => ({
  type: 2,
  timestamp: i,
  data: { i },
} as EventPayload);

describe('Batch', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('flushes when reaching maxEvents threshold', () => {
    const onFlush = vi.fn();
    const batch = new Batch(onFlush, { maxEvents: 3, maxMs: 1000 });

    batch.push(sampleEvent(1));
    batch.push(sampleEvent(2));
    expect(onFlush).not.toHaveBeenCalled();

    batch.push(sampleEvent(3));
    expect(onFlush).toHaveBeenCalledTimes(1);
    const flushed = onFlush.mock.calls[0]![0] as EventPayload[];
    expect(flushed).toHaveLength(3);
    expect(flushed[0]!.data).toEqual({ i: 1 });
  });

  it('flushes after maxMs timeout when below event threshold', () => {
    const onFlush = vi.fn();
    const batch = new Batch(onFlush, { maxEvents: 100, maxMs: 100 });

    batch.push(sampleEvent(1));
    batch.push(sampleEvent(2));
    expect(onFlush).not.toHaveBeenCalled();

    vi.advanceTimersByTime(99);
    expect(onFlush).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1);
    expect(onFlush).toHaveBeenCalledTimes(1);
    expect(onFlush.mock.calls[0]![0]).toHaveLength(2);
  });

  it('does not schedule duplicate timer on subsequent pushes', () => {
    const onFlush = vi.fn();
    const batch = new Batch(onFlush, { maxEvents: 100, maxMs: 100 });

    batch.push(sampleEvent(1));
    batch.push(sampleEvent(2));
    batch.push(sampleEvent(3));

    // 推进 100ms 应只触发一次 flush
    vi.advanceTimersByTime(100);
    expect(onFlush).toHaveBeenCalledTimes(1);
  });

  it('manual flush triggers callback immediately with buffered events', () => {
    const onFlush = vi.fn();
    const batch = new Batch(onFlush, { maxEvents: 100, maxMs: 1000 });

    batch.push(sampleEvent(1));
    batch.push(sampleEvent(2));
    batch.flush();

    expect(onFlush).toHaveBeenCalledTimes(1);
    expect(onFlush.mock.calls[0]![0]).toHaveLength(2);

    // 后续 timer 不应再触发 flush
    vi.advanceTimersByTime(1000);
    expect(onFlush).toHaveBeenCalledTimes(1);
  });

  it('flush on empty buffer does not call callback', () => {
    const onFlush = vi.fn();
    const batch = new Batch(onFlush);

    batch.flush();
    expect(onFlush).not.toHaveBeenCalled();
  });

  it('flush clears timer (no double-flush after threshold + timer)', () => {
    const onFlush = vi.fn();
    const batch = new Batch(onFlush, { maxEvents: 2, maxMs: 50 });

    batch.push(sampleEvent(1));
    batch.push(sampleEvent(2)); // triggers flush (maxEvents)
    expect(onFlush).toHaveBeenCalledTimes(1);

    // 即使 timer 仍在 schedule,推进后不应再次 flush(因 flush 已清 timer)
    vi.advanceTimersByTime(100);
    expect(onFlush).toHaveBeenCalledTimes(1);
  });

  it('destroy stops timer and clears buffer', () => {
    const onFlush = vi.fn();
    const batch = new Batch(onFlush, { maxEvents: 100, maxMs: 100 });

    batch.push(sampleEvent(1));
    batch.destroy();

    vi.advanceTimersByTime(200);
    expect(onFlush).not.toHaveBeenCalled();

    // destroy 后再 flush 不应触发(已清空)
    batch.flush();
    expect(onFlush).not.toHaveBeenCalled();
  });

  it('resumes batching after flush', () => {
    const onFlush = vi.fn();
    const batch = new Batch(onFlush, { maxEvents: 2, maxMs: 1000 });

    batch.push(sampleEvent(1));
    batch.push(sampleEvent(2));
    expect(onFlush).toHaveBeenCalledTimes(1);

    // 第二轮 push
    batch.push(sampleEvent(3));
    batch.push(sampleEvent(4));
    expect(onFlush).toHaveBeenCalledTimes(2);
    expect(onFlush.mock.calls[1]![0]).toHaveLength(2);
  });

  it('uses default 50 events and 100ms when no options given', () => {
    const onFlush = vi.fn();
    const batch = new Batch(onFlush);

    // 推 49 个,不应触发
    for (let i = 0; i < 49; i++) {
      batch.push(sampleEvent(i));
    }
    expect(onFlush).not.toHaveBeenCalled();

    // 第 50 个触发
    batch.push(sampleEvent(49));
    expect(onFlush).toHaveBeenCalledTimes(1);
    expect(onFlush.mock.calls[0]![0]).toHaveLength(50);
  });
});
