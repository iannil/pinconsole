// Privacy.vue 基础测试：渲染 / 删除流 / 结果 / 错误
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import enUS from '../src/i18n/en-US';
import Privacy from '../src/views/Privacy.vue';

const deleteMock = vi.fn();
vi.mock('../src/api/privacy', () => ({
  deleteVisitorByFingerprint: (...args: unknown[]) => deleteMock(...args),
}));

const i18n = createI18n({
  locale: 'en',
  fallbackLocale: 'en',
  messages: { en: enUS },
});

describe('Privacy.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('confirm', vi.fn(() => true));
  });

  it('渲染标题和描述', () => {
    const w = mount(Privacy, { global: { plugins: [i18n] } });
    expect(w.text()).toContain('Privacy');
    expect(w.text()).toContain('GDPR');
    expect(w.find('.fp-input').exists()).toBe(true);
    expect(w.find('.delete-btn').exists()).toBe(true);
    w.unmount();
  });

  it('空 fingerprint 时按钮禁用', () => {
    const w = mount(Privacy, { global: { plugins: [i18n] } });
    expect(w.find('.delete-btn').attributes('disabled')).toBe('');
    w.unmount();
  });

  it('输入 fingerprint 后按钮启用', async () => {
    const w = mount(Privacy, { global: { plugins: [i18n] } });
    await w.find('.fp-input').setValue('abc123');
    expect(w.find('.delete-btn').attributes('disabled')).toBeUndefined();
    w.unmount();
  });

  it('confirm 取消不调 API', async () => {
    vi.stubGlobal('confirm', vi.fn(() => false));
    const w = mount(Privacy, { global: { plugins: [i18n] } });
    await w.find('.fp-input').setValue('fp123');
    await w.find('.delete-btn').trigger('click');
    expect(deleteMock).not.toHaveBeenCalled();
    w.unmount();
  });

  it('删除成功后显示结果', async () => {
    deleteMock.mockResolvedValue({ deleted_sessions: 5, deleted_minio_objects: 3, fingerprint: 'fp' });
    const w = mount(Privacy, { global: { plugins: [i18n] } });
    await w.find('.fp-input').setValue('fp123');
    await w.find('.delete-btn').trigger('click');
    await flushPromises();
    expect(deleteMock).toHaveBeenCalledWith('fp123');
    expect(w.text()).toContain('Deleted 5 sessions');
    expect(w.text()).toContain('Deleted 3 MinIO objects');
    w.unmount();
  });

  it('删除失败显示错误', async () => {
    deleteMock.mockRejectedValue(new Error('not found'));
    const w = mount(Privacy, { global: { plugins: [i18n] } });
    await w.find('.fp-input').setValue('bad-fp');
    await w.find('.delete-btn').trigger('click');
    await flushPromises();
    expect(w.text()).toContain('not found');
    w.unmount();
  });
});
