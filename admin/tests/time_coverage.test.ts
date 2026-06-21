// TS-3 切片补测:formatRelative 时间格式化全 case。
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { formatRelative } from '../src/utils/time';

// mock vue-i18n 的 ComposerTranslation 简单版
const mockT = vi.fn((key: string, params?: any) => {
  if (key === 'time.just_now') return 'just now';
  if (key === 'time.seconds_ago') return `${params.n}s ago`;
  if (key === 'time.minutes_ago') return `${params.n}m ago`;
  if (key === 'time.hours_ago') return `${params.n}h ago`;
  if (key === 'time.fallback_date') return `${params.month}/${params.day}`;
  return key;
});

describe('formatRelative', () => {
  const realNow = Date.now;

  beforeEach(() => {
    mockT.mockClear();
    // 固定 now = 2026-06-21T12:00:00Z
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-06-21T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
    Date.now = realNow;
  });

  it('just_now(< 1s)', () => {
    const ts = Date.now() - 500;
    const got = formatRelative(ts, mockT as any);
    expect(mockT).toHaveBeenCalledWith('time.just_now');
    expect(got).toBe('just now');
  });

  it('seconds_ago(< 60s)', () => {
    const ts = Date.now() - 30_000; // 30s
    const got = formatRelative(ts, mockT as any);
    expect(mockT).toHaveBeenCalledWith('time.seconds_ago', { n: 30 });
    expect(got).toBe('30s ago');
  });

  it('minutes_ago(< 60min)', () => {
    const ts = Date.now() - 5 * 60_000; // 5min
    const got = formatRelative(ts, mockT as any);
    expect(mockT).toHaveBeenCalledWith('time.minutes_ago', { n: 5 });
    expect(got).toBe('5m ago');
  });

  it('hours_ago(< 24h)', () => {
    const ts = Date.now() - 3 * 3_600_000; // 3h
    const got = formatRelative(ts, mockT as any);
    expect(mockT).toHaveBeenCalledWith('time.hours_ago', { n: 3 });
    expect(got).toBe('3h ago');
  });

  it('fallback_date(>= 24h)', () => {
    const ts = Date.now() - 48 * 3_600_000; // 48h
    const got = formatRelative(ts, mockT as any);
    // mockT 应被 fallback_date 调用,含 month/day
    expect(mockT).toHaveBeenCalledWith(
      'time.fallback_date',
      expect.objectContaining({ month: expect.any(Number), day: expect.any(Number) }),
    );
  });

  it('seconds_ago 边界(59s)', () => {
    const ts = Date.now() - 59_000;
    const got = formatRelative(ts, mockT as any);
    expect(got).toBe('59s ago');
  });

  it('minutes_ago 边界(59min)', () => {
    const ts = Date.now() - 59 * 60_000;
    const got = formatRelative(ts, mockT as any);
    expect(got).toBe('59m ago');
  });

  it('hours_ago 边界(23h)', () => {
    const ts = Date.now() - 23 * 3_600_000;
    const got = formatRelative(ts, mockT as any);
    expect(got).toBe('23h ago');
  });
});
