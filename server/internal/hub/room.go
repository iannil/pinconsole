// Package hub：session 与 tenant 房间的订阅实现。
package hub

import "github.com/google/uuid"

// subscribe 给当前 sub 加订阅，返回接收 chan。
// chan 容量 64 足以吸收瞬时突发；满了 publish 会丢弃（带日志）。
func (sc *SessionChan) subscribe(subID uuid.UUID) <-chan []byte {
	ch := make(chan []byte, 64)
	sc.mu.Lock()
	sc.subs[subID] = ch
	sc.mu.Unlock()
	return ch
}

func (sc *SessionChan) unsubscribe(subID uuid.UUID) {
	sc.mu.Lock()
	if ch, ok := sc.subs[subID]; ok {
		delete(sc.subs, subID)
		close(ch)
	}
	sc.mu.Unlock()
}

func (sc *SessionChan) unsubscribeAll(subID uuid.UUID) {
	sc.unsubscribe(subID)
}

// unsubscribeAllLocking 由 hub 在 VisitorOffline 时持有写锁调用。
func (sc *SessionChan) unsubscribeAllLocking() {
	sc.mu.Lock()
	for _, ch := range sc.subs {
		close(ch)
	}
	sc.subs = make(map[uuid.UUID]chan []byte)
	sc.mu.Unlock()
}

func (sc *SessionChan) publish(msg []byte) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	for _, ch := range sc.subs {
		select {
		case ch <- msg:
		default:
			// 满，丢弃
		}
	}
}

// ===== TenantRoom =====

func (tr *TenantRoom) subscribe(subID uuid.UUID) <-chan []byte {
	ch := make(chan []byte, 64)
	tr.mu.Lock()
	tr.subs[subID] = ch
	tr.mu.Unlock()
	return ch
}

func (tr *TenantRoom) unsubscribe(subID uuid.UUID) {
	tr.mu.Lock()
	if ch, ok := tr.subs[subID]; ok {
		delete(tr.subs, subID)
		close(ch)
	}
	tr.mu.Unlock()
}

func (tr *TenantRoom) unsubscribeAll(subID uuid.UUID) {
	tr.unsubscribe(subID)
}

func (tr *TenantRoom) publish(msg []byte) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	for _, ch := range tr.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}
