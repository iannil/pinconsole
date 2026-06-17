import { describe, it, expect } from 'vitest';

// 切片 1a：admin 占位 smoke 测试。
// 真正的组件测试从切片 1b 起加入。
describe('admin smoke', () => {
  it('runs vitest', () => {
    expect(1 + 1).toBe(2);
  });
});
