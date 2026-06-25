/**
 * replay-parity: 验证 replay-core Replayer 与原始 rrweb Replayer 对相同
 * events 产生相同渲染输出。
 *
 * 测试策略：
 *   1. 用原始 rrweb 录制一段 events ⇒ 作为公共输入 fixtures
 *   2. 分别传给旧 rrweb.Replayer 和新 replay-core Replayer
 *   3. 等待双方 Finish ⇒ 比较 iframe 内容
 *
 * 注意：本测试使用 jsdom 环境 (iframe srcdoc 渲染)，
 * 如 jsdom 限制过严，后续升级到 Playwright。
 */
import { describe, it, expect } from 'vitest';
import { Replayer as OldReplayer } from 'rrweb';
import { Replayer as NewReplayer } from '../src/replay/index';
import type { eventWithTime, EventType } from '../src/types';
import { IncrementalSource, ReplayerEvents } from '../src/types';

// ============================================================
// 最小 events 夹具（由下游 Playwright 录制生成并替换）
// ============================================================
const FIXTURE_EVENTS: eventWithTime[] = [
  // Event 1: Meta — 窗口信息
  {
    type: 4 as EventType, // EventType.Meta
    timestamp: 0,
    data: {
      href: 'about:blank',
      width: 1024,
      height: 768,
    },
  },
  // Event 2: FullSnapshot — 最小 DOM 树
  {
    type: 2 as EventType, // EventType.FullSnapshot
    timestamp: 0,
    data: {
      node: {
        type: 0, // NodeType.Document
        childNodes: [
          {
            type: 1, // NodeType.DocumentType
            name: 'html',
            publicId: '',
            systemId: '',
            id: 1,
          },
          {
            type: 2, // NodeType.Element
            tagName: 'html',
            attributes: { lang: 'en' },
            childNodes: [
              {
                type: 2,
                tagName: 'head',
                attributes: {},
                childNodes: [],
                id: 3,
              },
              {
                type: 2,
                tagName: 'body',
                attributes: {},
                childNodes: [
                  {
                    type: 2,
                    tagName: 'div',
                    attributes: { id: 'root', 'class': 'container' },
                    childNodes: [
                      { type: 3, textContent: 'Hello Parity!', id: 6 },
                    ],
                    id: 4,
                  },
                  {
                    type: 2,
                    tagName: 'span',
                    attributes: {},
                    childNodes: [
                      { type: 3, textContent: 'world', id: 8 },
                    ],
                    id: 7,
                  },
                ],
                id: 5,
              },
            ],
            id: 2,
          },
        ],
        id: 0,
      },
      initialOffset: { top: 0, left: 0 },
    },
  },
];

// ============================================================
// 工具函数
// ============================================================

/** 等待 Replayer 触发 Finish 事件（最大 5s） */
function waitForFinish(replayer: OldReplayer | NewReplayer): Promise<void> {
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => reject(new Error('Replayer finish timeout')), 5000);

    // rrweb v2 使用 emitter (mitt) 而非 on
    // Replayer.on(event, handler) 方法
    replayer.on('finish', () => {
      clearTimeout(timeout);
      resolve();
    });
  });
}

/** 获取 Replayer 创建的 iframe 内容（标准化空白以容忍微小差异） */
function getIframeContent(replayer: OldReplayer | NewReplayer): string {
  const wrapper = replayer.wrapper as HTMLElement;
  if (!wrapper) return '';

  // 找到 renderer iframe（第二个 iframe，第一个可能是 mirror）
  const iframes = wrapper.querySelectorAll('iframe');
  if (iframes.length === 0) return '';

  // 取最后一个 iframe（渲染用）
  const iframe = iframes[iframes.length - 1];
  const doc = iframe.contentDocument || iframe.contentWindow?.document;
  if (!doc) return iframe.outerHTML;

  return doc.documentElement?.outerHTML || iframe.outerHTML;
}

// ============================================================
// 测试
// ============================================================
describe('replay-parity: Replayer 渲染等价', () => {
  it('相同 events 产生相同 iframe 内容', async () => {
    // 速度拉满，让 replay 瞬间完成
    const config = { speed: 999999 };

    const oldReplayer = new OldReplayer(FIXTURE_EVENTS, config);
    const newReplayer = new NewReplayer(FIXTURE_EVENTS, config);

    // 等待双方结束
    await Promise.all([waitForFinish(oldReplayer), waitForFinish(newReplayer)]);

    const oldHTML = getIframeContent(oldReplayer);
    const newHTML = getIframeContent(newReplayer);

    // 比较
    if (oldHTML !== newHTML) {
      console.error('REPLAY PARITY MISMATCH');
      console.error('=== OLD (rrweb) ===');
      console.error(oldHTML.substring(0, 500));
      console.error('=== NEW (replay-core) ===');
      console.error(newHTML.substring(0, 500));
    }

    expect(oldHTML).toBeTruthy();
    expect(newHTML).toBeTruthy();
    expect(oldHTML).toBe(newHTML);
  });
});
