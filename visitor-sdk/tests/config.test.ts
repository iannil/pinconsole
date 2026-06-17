import { describe, it, expect } from 'vitest';

// 切片 1a：SDK 占位 smoke 测试。
// rrweb 与 WebSocket 集成测试从切片 1b/1c 起加入。
describe('visitor-sdk smoke', () => {
  it('runs vitest', () => {
    expect(1 + 1).toBe(2);
  });
});
