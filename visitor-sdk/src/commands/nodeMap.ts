// nodeID ↔ DOM 节点映射
// rrweb-snapshot 用 buildNodeID 为每个节点分配 ID（数字）。
// 本模块遍历 document，为每个 element 计算同算法 ID，维护 map。
//
// 详见 docs/progress/2026-06-17-slice-1e-spec.md §访客端节点定位
//
// 简化实现：1e 用 data-mm-node-id attribute 标记（避免与 rrweb 内部算法同步）。
// SDK 启动时调 rrweb-snapshot 后，DOM 节点含 data-rr-node-id。
// 我们读 attribute 维护 map。

const RR_NODE_ATTR = 'data-rr-node-id';
const CUSTOM_ATTR = 'data-mm-node-id';

/**
 * NodeMap 维护 nodeID → DOM 节点的映射。
 * 通过遍历 document + 读 rrweb-snapshot 写入的 data-rr-node-id attribute 建立。
 */
export class NodeMap {
  private map: Map<number, Element> = new Map();
  private observer: MutationObserver | null = null;

  /** 初始化 map + 启动 MutationObserver 维护。 */
  start(): void {
    this.rebuild();
    this.observer = new MutationObserver(() => this.rebuild());
    this.observer.observe(document.body, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: [RR_NODE_ATTR, CUSTOM_ATTR, 'id', 'class'],
    });
  }

  stop(): void {
    if (this.observer) {
      this.observer.disconnect();
      this.observer = null;
    }
    this.map.clear();
  }

  /** 重新扫描整个 document，重建 map。 */
  rebuild(): void {
    this.map.clear();
    const all = document.querySelectorAll(`[${RR_NODE_ATTR}], [${CUSTOM_ATTR}]`);
    for (const el of all) {
      const idStr = el.getAttribute(RR_NODE_ATTR) ?? el.getAttribute(CUSTOM_ATTR);
      if (idStr === null) continue;
      const id = parseInt(idStr, 10);
      if (!Number.isNaN(id)) {
        this.map.set(id, el);
      }
    }
  }

  /** 按 ID 取节点。 */
  get(nodeID: number): Element | null {
    // 先看缓存
    const cached = this.map.get(nodeID);
    if (cached && document.body.contains(cached)) {
      return cached;
    }
    // fallback: 实时查
    const el = document.querySelector(
      `[${RR_NODE_ATTR}="${nodeID}"], [${CUSTOM_ATTR}="${nodeID}"]`,
    );
    if (el) {
      this.map.set(nodeID, el);
    }
    return el;
  }
}
