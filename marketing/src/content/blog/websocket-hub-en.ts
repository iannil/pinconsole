import type { BlogContent } from './types';

export const websocketEn: BlogContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'Building a Production WebSocket Hub for 500 Concurrent Session Replay Connections',
    description: 'A technical deep-dive into PinConsole\'s WebSocket hub architecture — goroutine-per-connection, non-blocking fan-out, Redis Stream batching, and MinIO flush patterns for handling 500 concurrent visitors.',
    ogTitle: 'Building a Production WebSocket Hub for 500 Concurrent Connections',
    ogDescription: 'Goroutine-per-connection, non-blocking fan-out, Redis Stream batching, and MinIO flush patterns — the architecture behind PinConsole\'s real-time hub.',
  },
  blog: {
    author: 'Rong Zhu',
    publishedDate: '2026-06-28',
    readingTime: '11 min read',
    tags: ['Engineering', 'WebSocket', 'Architecture', 'Go'],
  },
  hero: {
    h1: 'Building a Production WebSocket Hub for 500 Concurrent Session Replay Connections',
    subtitle: 'Goroutine-per-connection, non-blocking fan-out, Redis Stream batching, and MinIO flush patterns — the real-time architecture behind PinConsole.',
  },
  content: {
    sections: [
      {
        heading: 'The Real-Time Challenge',
        body: `Session replay is fundamentally a real-time system. While replaying recorded sessions is the primary feature, the real value for operators is watching visitors live — seeing their cursor move, their form fill in, their page scroll — and being able to jump into co-browsing the moment they spot a problem.\n\nThis means every visitor event must be captured, transmitted, stored, and broadcast to the admin panel with minimal latency. For a single visitor, that's easy. For 500 concurrent visitors all generating rrweb events at 60fps, the math gets serious: 500 × 60 = 30,000 events per second flowing through the hub.\n\nWe set 500 concurrent WebSocket connections per room as our baseline. That's one operator serving a mid-size e-commerce site's traffic in real time. The architecture needs to handle this without dropping events, without memory blow-up, and without blocking the event pipeline.\n\nThis is how we built it.`,
      },
      {
        heading: 'The Hub: An In-Process Registry, Not a Message Broker',
        body: `The first architectural decision was: do we use an external message broker (Redis Pub/Sub, RabbitMQ, NATS) or an in-process hub?\n\nAn external broker would make PinConsole horizontally scalable — spin up more server instances to handle more traffic. But it adds latency, operational complexity, and deployment friction. For a self-hosted tool that targets "docker compose up -d" simplicity, every external dependency is a cost.\n\nWe chose an in-process hub: a single Go struct with four maps protected by mutexes:\n\n1. clients — maps connection ID to Client (the WebSocket wrapper)\n2. sessions — maps session ID to SessionChan (the fan-out channel)\n3. tenantRooms — maps tenant ID to TenantRoom (presence broadcasting)\n4. visitorClients — maps session ID to visitor client ID (reverse command path)\n\nThe hub is a singleton within the process. All goroutines — visitor WebSocket readers, operator WebSocket readers, Redis Stream flushers — share the same hub instance. There's no network hop, no serialization overhead, no message broker to configure.\n\nThe trade-off: PinConsole is not horizontally scalable at the hub level. All WebSocket connections must land on the same server instance. But for 500 concurrent connections per room, a single Go process handles this comfortably. We optimize for the 95th percentile deployment, not the 99.9th.`,
        code: `// Hub: in-process registry, no external broker
type Hub struct {
  mu             sync.RWMutex
  clients        map[uuid.UUID]*Client
  sessions       map[uuid.UUID]*SessionChan
  tenants        map[uuid.UUID]*TenantRoom
  visitorClients map[uuid.UUID]uuid.UUID
}`,
        codeLanguage: 'go',
      },
      {
        heading: 'Client: Goroutine-Per-Connection with Bounded Write Buffers',
        body: `Each WebSocket connection is wrapped in a Client struct. The key design element is the write loop — a dedicated goroutine that serializes writes to the WebSocket connection.\n\nWhen the hub needs to send a message to a client (e.g., an event forwarded from a visitor's session), it pushes the message onto the client's writeCh channel. The writeLoop goroutine picks it up and calls conn.Write(). This ensures writes to the same WebSocket connection are serialized — concurrent writes to a WebSocket connection are unsafe.\n\nThe writeCh has a capacity of 256 messages. If the buffer is full, the Send() method uses a non-blocking select with a default case: instead of blocking the publisher, it logs a warning and drops the message. This is a deliberate choice — a slow reader (network congestion, a paused tab) should never block the entire hub.\n\nEach Client runs exactly one writeLoop goroutine. For 500 concurrent connections, that's 500 goroutines for writing. The Go scheduler handles this gracefully — each goroutine blocks on channel receive and wakes only when there's work to do. Memory overhead is minimal (approximately 8KB per goroutine stack).\n\nWhen a client disconnects (or its connection drops), UnregisterClient() cleans up all subscriptions and joined tenant rooms atomically. The closeOnce sync.Once prevents double-close on the channels.`,
        code: `// Client write loop: serializes writes, never blocks the caller
func (c *Client) writeLoop(ctx context.Context) {
  for {
    select {
    case msg := <-c.writeCh:
      ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
      err := c.Conn.Write(ctx, websocket.MessageBinary, msg)
      cancel()
      if err != nil {
        return  // connection closed
      }
    case <-c.closeCh:
      return
    }
  }
}

// Send: non-blocking push with backpressure signal
func (c *Client) Send(msg []byte) bool {
  select {
  case c.writeCh <- msg:
    return true
  case <-c.closeCh:
    return false
  default:
    slog.Warn("client write queue full, dropping message",
      "client_id", c.ID)
    return false
  }
}`,
        codeLanguage: 'go',
      },
      {
        heading: 'SessionChan: Non-Blocking Fan-Out to Multiple Operators',
        body: `A single visitor session can be watched by multiple operators simultaneously. When the visitor generates an event, it needs to be broadcast to every operator subscribed to that session.\n\nThe SessionChan is the fan-out mechanism. It maintains a map of subscriber channels — one per operator — each with a buffer of 64 messages. When publish() is called, it iterates over all subscribers and pushes the message into each channel's buffer:\n\nfor each subscriber:\n  select { case ch <- msg: default: drop }\n\nThe non-blocking default means one slow operator (e.g., on a congested network) doesn't degrade the experience for other operators watching the same session. The slow operator's channel fills up and starts dropping, but everyone else continues to receive every event.\n\nThe same pattern is used for TenantRoom, which broadcasts presence events (visitor online/offline) to all operators in a tenant. Presence events are low-frequency, so drops are rare — but the non-blocking pattern is consistent across the codebase.\n\nWhen an operator unsubscribes from a session, the subscriber channel is removed from the map and closed. The close signal propagates to the operator's client write loop, which stops forwarding events from that session.`,
      },
      {
        heading: 'The Event Pipeline: From Browser to MinIO in Three Stages',
        body: `Events don't just need to be broadcast — they need to be stored for replay. The pipeline has three stages:\n\nStage 1: WebSocket Receive. The visitor SDK sends events in batches (100ms or 50 events, whichever comes first). Each batch is a msgpack-encoded array of rrweb events. The server's visitorWS goroutine reads the message and:\n\n1. Appends it to Redis Stream (stream:session:{uuid}) for durability\n2. Extracts full snapshots and meta events for caching\n3. Publishes to the hub for real-time delivery to operators\n4. Updates the session's last_event_at timestamp in PostgreSQL\n5. Passes to the behavioral analysis tracker\n\nStage 2: Redis Stream Flush. A background goroutine (one per session) runs every 30 seconds or every 1000 events, whichever comes first. It:\n\n1. Reads unflushed events from Redis Stream via XRANGE\n2. Aggregates them into a single msgpack blob\n3. Compresses with zstd\n4. Writes to MinIO PutObject\n5. Inserts a record in PostgreSQL event_blobs (session_id, minio_object_key, offset, size)\n6. Trims the Redis Stream to keep only the last 200 entries (XTRIM)\n\nStage 3: Garbage Collection. An hourly GC worker deletes event blobs and metadata for sessions that have ended and exceeded the retention period (default 30 days).\n\nThe Redis Stream serves as a buffer between the hot path (real-time WebSocket receive) and the cold path (object storage flush). If MinIO is temporarily unavailable, events accumulate in Redis (up to 200 unflushed entries per session). When MinIO recovers, the flusher catches up.`,
        code: `// Flusher: Redis Stream → MinIO with compensating transaction
func flushSession(ctx context.Context, s *Session) error {
  // 1. Read unflushed entries from Redis Stream
  entries, err := s.stream.XRange(ctx, "-", "+")
  if err != nil || len(entries) == 0 {
    return err
  }

  // 2. Aggregate and compress
  blob := encodeBlob(entries)
  compressed := zstdCompress(blob)

  // 3. Write to MinIO first
  objectKey := fmt.Sprintf("sessions/%s/events-%d.zst",
    s.sessionID, s.nextOffset)
  err = s.minio.PutObject(ctx, bucket, objectKey, compressed)
  if err != nil {
    return err  // MinIO write failed, retry next tick
  }

  // 4. Record in PostgreSQL (compensating: delete MinIO object if PG fails)
  err = s.db.Exec(ctx,
    "INSERT INTO event_blobs (session_id, minio_object_key, ...) VALUES ($1, $2, ...)",
    s.sessionID, objectKey)
  if err != nil {
    s.minio.RemoveObject(ctx, bucket, objectKey)  // cleanup orphan
    return err
  }

  // 5. Trim Redis Stream
  s.stream.XTrim(ctx, 200)
  return nil
}`,
        codeLanguage: 'go',
      },
      {
        heading: 'Snapshot Caching: Make the Admin Panel Feel Instant',
        body: `When an operator opens a session for live monitoring, they shouldn't see a blank canvas while waiting for the next rrweb incremental event. They should see the page immediately — as it was when the visitor loaded it.\n\nTo achieve this, we cache two critical events in Redis:\n\n• Meta event (rrweb type 4): Contains viewport dimensions. Without this, the rrweb player doesn't know how big the iframe should be.\n• Full snapshot (rrweb type 2): A complete serialization of the DOM tree. This is what the player renders to show the initial page state.\n\nBoth are cached with a 30-minute TTL. The meta event is set once per session (on the first meta event). The full snapshot is updated each time rrweb takes a new full snapshot (every 10 seconds or on page visibility change).\n\nWhen an operator subscribes to a session, the server sends meta first, then full snapshot, then streams incremental events. The rrweb player receives these in order and renders the initial page before any operator sees it. The experience is indistinguishable from watching a live page — no loading spinner, no "buffering" indicator.\n\nThe cache TTL of 30 minutes was chosen based on typical session length analysis. Most sessions are under 30 minutes. For longer sessions (e.g., an operator watching a support interaction), the next periodic full snapshot refreshes the cache.`,
      },
      {
        heading: 'Presence: Who\'s Online and Who\'s Watching',
        body: `Operators need to know which visitors are currently on the site and which operators are watching them. This is the presence system.\n\nWhen a visitor connects, the visitor WebSocket handler calls Hub.VisitorOnline(), which:\n\n1. Creates a SessionChan for the session\n2. Records the visitor client ID in visitorClients\n3. Broadcasts a presence.online message to the TenantRoom\n\nThe TenantRoom receives the message and fans it out to all connected operators in that tenant. Each operator's admin panel updates its UI to show the new visitor in the live feed.\n\nWhen the visitor disconnects (WebSocket close or session timeout), VisitorOffline() broadcasts presence.offline and cleans up the session mapping.\n\nOperators also broadcast their own presence when they connect/disconnect — so the visitor's co-browsing banner can show the operator's name, and other operators know who's assisting which session.`,
      },
      {
        heading: 'Backpressure at Every Layer',
        body: `In a system with 500 concurrent data streams, the only guarantee is that some consumers will be slower than others. Network congestion, background tab throttling, and browser rendering lag all cause operators to fall behind.\n\nPinConsole handles backpressure at three levels:\n\n1. Client writeCh (256 buffer): When an operator's WebSocket connection is slow, their Client's write buffer fills up. Excess messages are dropped with a warning log. The operator misses some events but the session continues for everyone else.\n\n2. SessionChan subscriber channels (64 buffer): When an operator's subscriber channel fills up, the router drops messages before they reach the Client. This is a secondary backpressure layer — the first dropped-message warning is at the SessionChan level, not the Client level.\n\n3. Redis Stream trimming (200 entries): The Redis Stream is trimmed after each flush to keep only 200 entries. If the flusher falls behind (MinIO is slow), the stream won't grow unbounded. The oldest entries are discarded.\n\nAll three levels use the same pattern: drop the oldest data when buffers are full. For session replay, this is acceptable — losing a few incremental events means the operator sees a minor jump in the replay rather than a gaping hole. The full snapshots ensure the page state is always correct, even if some increments are lost.\n\nWe validated this with a concurrent stress test (50 goroutines sending to the same session, -race enabled). The hub doesn't deadlock, doesn't panic, and doesn't leak goroutines under pressure.`,
      },
      {
        heading: 'Why Not WebRTC or P2P?',
        body: `A common question is: why route all traffic through a central server instead of using WebRTC or P2P for the visitor → operator data path?\n\nWebRTC would reduce server bandwidth — the visitor sends events directly to the operator's browser. But it introduces complexity: STUN/TURN server setup, NAT traversal handling, browser compatibility issues, and a separate signaling channel.\n\nMore importantly, WebRTC doesn't help with session recording. The events still need to reach the server for storage. With a hub-and-spoke model, the server receives events once and handles both storage and forwarding simultaneously. With P2P, the server would need to either receive a copy of every event (defeating the purpose of P2P) or request events from the operator after the session (adding complexity).\n\nFor a self-hosted tool, simplicity is a feature. The hub-and-spoke model is dead simple to reason about, debug, and deploy. The bottleneck isn't server bandwidth — it's storage I/O. And that's handled by the async flush pipeline, not the WebSocket hub.`,
      },
      {
        heading: 'From 500 to 5000: The Scaling Path',
        body: `PinConsole targets 500 concurrent connections per room as its baseline. But the architecture can scale further with straightforward changes:\n\n• Increase Redis Stream trim limit (TrimKeep 200 → 500): allows larger backpressure buffer\n• Increase Client writeCh buffer (256 → 1024): trades memory for fewer dropped messages\n• Increase Redis connection pool (PoolSize 50 → 200): handles more concurrent XADD operations\n• Increase PostgreSQL connection pool (MaxConns 25 → 100): handles more session metadata queries\n\nAll of these are configuration changes, not code changes. The hub itself has no tunable limits — it's bounded only by available memory and goroutine scheduler throughput.\n\nFor deployments that need beyond 5000 concurrent connections, the in-process hub becomes a bottleneck. At that scale, you'd want to split traffic across multiple server instances behind a WebSocket-aware load balancer, with Redis Pub/Sub bridging the hubs. That's a v2 problem. For a self-hosted tool serving a single tenant, 500 connections per room covers virtually all real-world use cases.`,
      },
    ],
  },
  relatedPosts: [
    { title: 'How We Built a Self-Hosted Session Replay Alternative to FullStory', url: '/en/blog/building-self-hosted-session-replay/', description: 'The full architecture behind PinConsole — Go, rrweb, MinIO, and single-binary deployment.' },
    { title: 'Defense in Depth for Session Replay: Building Anti-Bot Infrastructure in Go', url: '/en/blog/defense-in-depth/', description: 'Rate limiting, behavioral analysis, and SDK fingerprinting for the WebSocket layer.' },
  ],
  cta: {
    title: 'Try Real-Time Session Monitoring',
    subtitle: '500 concurrent connections per room. Self-hosted. Single binary.',
    primary: { label: 'Get started on GitHub', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: 'Talk to the maintainer', href: '#consult' },
  },
};
