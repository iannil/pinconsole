import type { BlogContent } from './types';

export const securityEn: BlogContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'Defense in Depth for Session Replay: Building Anti-Bot Infrastructure in Go',
    description: 'How PinConsole implements defense-in-depth against bots and scrapers — rate limiting, behavioral analysis, SDK fingerprinting, WebSocket auth, and URL validation. A technical guide to securing a self-hosted session replay platform.',
    ogTitle: 'Defense in Depth for Session Replay: Anti-Bot Infrastructure',
    ogDescription: 'Rate limiting, behavioral analysis, SDK fingerprinting, and WebSocket auth — the four-layer anti-bot defense behind PinConsole.',
  },
  blog: {
    author: 'Rong Zhu',
    publishedDate: '2026-06-28',
    readingTime: '10 min read',
    tags: ['Security', 'Anti-Bot', 'Engineering', 'Self-Hosted'],
  },
  hero: {
    h1: 'Defense in Depth for Session Replay: Building Anti-Bot Infrastructure in Go',
    subtitle: 'Rate limiting, behavioral analysis, SDK fingerprinting, and WebSocket auth — how PinConsole protects self-hosted session replay from bots and scrapers.',
  },
  content: {
    sections: [
      {
        heading: 'Why Session Replay Tools Are Prime Bot Targets',
        body: `Session replay tools sit in a unique security position. They serve a JavaScript SDK to every visitor, maintain persistent WebSocket connections, and stream DOM data back to the server. For a bot operator, this is a goldmine — the SDK is public, the WebSocket protocol can be reverse-engineered, and the server accepts event data from anyone.\n\nA naive session replay deployment can be abused in multiple ways:\n\n• Event injection: a bot sends fake events to pollute analytics data\n• Data exfiltration: a scraper uses the SDK to capture page content at scale\n• Credential stuffing: if rate limits are weak, the login endpoint gets hammered\n• Session hijacking: if WebSocket auth is weak, an attacker can listen to live visitor streams\n\nBecause PinConsole is self-hosted, the security posture is different from a SaaS tool. There's no vendor-managed WAF, no Cloudflare in front, no security team monitoring traffic patterns. The security must be baked into the application itself.\n\nThis is how we built defense-in-depth — four layers, each catching what the previous one misses.`,
      },
      {
        heading: 'Layer 1: HTTP Rate Limiting (Fixed Window per IP)',
        body: `The outermost defense is HTTP-level rate limiting. Every request to the PinConsole API passes through a Gin middleware that enforces a per-IP rate cap.\n\nThe implementation is straightforward: for each request, we INCR a Redis key (ratelimit:{ip}:{window}) and set EXPIRE to the window duration (60 seconds). If the count exceeds the limit (60 requests per minute by default), we return 429 Too Many Requests with standard rate limit headers.\n\nA few design decisions worth noting:\n\n• Redis failure is handled gracefully. If Redis is unreachable, the middleware allows the request through. We'd rather have a brief period of unlimited requests than knock the entire admin panel offline due to a Redis blip.\n\n• The rate limiter is production-only. In development mode, it's disabled so e2e tests don't trip over rate caps.\n\n• Health check endpoints (GET /healthz, GET /readyz) bypass rate limiting entirely.\n\n• The response includes Retry-After and X-RateLimit-Remaining headers, so well-behaved clients can back off without waiting for errors.\n\nThis layer stops naive scrapers and misconfigured bots. But it won't stop a distributed attack with rotating IPs — that requires deeper analysis.`,
        code: `// Simplified HTTP rate limiter middleware
func RateLimitMiddleware(rdb *redis.Client) gin.HandlerFunc {
  return func(c *gin.Context) {
    ip := c.ClientIP()
    now := time.Now().Unix() / 60  // 1-minute window
    key := fmt.Sprintf("ratelimit:%s:%d", ip, now)

    count, err := rdb.Incr(ctx, key).Result()
    if err != nil {
      c.Next()  // Redis down: fail open
      return
    }
    if count == 1 {
      rdb.Expire(ctx, key, 60*time.Second)
    }
    if count > 60 {
      c.AbortWithStatusJSON(429, gin.H{"error": "rate_limit_exceeded"})
      return
    }
    c.Next()
  }
}`,
        codeLanguage: 'go',
      },
      {
        heading: 'Layer 2: WebSocket Rate Limiting (Sliding Window per Session)',
        body: `HTTP rate limiting doesn't apply to WebSocket connections — once the upgrade succeeds, the connection is persistent and messages flow freely. So we need a separate rate limiter for the WebSocket message stream.\n\nPinConsole applies two sliding-window limits per session:\n\n• 500 messages per 10-second window\n• 50 MiB of data per 10-second window\n\nBoth limits are enforced via a single Redis Lua script that atomically INCR (for message count) and INCRBY (for byte count) with EXPIRE. The key is scoped to the session ID, not the IP — this prevents a bot from rotating IPs to bypass the limit.\n\nWhen either limit is exceeded, the server:\n\n1. Flags the session via antiscrape.FlagSession() (persisted to Redis with a 10-minute TTL)\n2. Immediately closes the WebSocket connection with a 1011 (Internal Error) status code\n3. Logs the violation with the session ID for operator review\n\nThe flag propagates to the admin panel — any operator monitoring the live feed sees a warning that the session has been flagged for suspicious activity.\n\nCrucially, the rate check happens before the event is written to the event stream. A flagged message is never persisted to MinIO or Redis Stream — it's rejected before it enters the pipeline. This prevents pollution of the event data even if the limiter is triggered mid-stream.`,
        code: `-- Lua: atomic WebSocket rate check
local countKey = KEYS[1]       -- ws:rate:count:{session}
local bytesKey = KEYS[2]       -- ws:rate:bytes:{session}
local window = ARGV[1]         -- 10 (seconds)
local maxMsg = ARGV[2]         -- 500
local maxBytes = ARGV[3]       -- 52428800 (50 MiB)
local msgSize = ARGV[4]

local msgCount = redis.call('INCR', countKey)
if msgCount == 1 then
  redis.call('EXPIRE', countKey, window)
end
if msgCount > tonumber(maxMsg) then
  return {0, "rate_exceeded"}
end

local byteCount = redis.call('INCRBY', bytesKey, msgSize)
if byteCount == msgSize then
  redis.call('EXPIRE', bytesKey, window)
end
if byteCount > tonumber(maxBytes) then
  return {0, "rate_exceeded"}
end

return {1, "ok"}`,
        codeLanguage: 'lua',
      },
      {
        heading: 'Layer 3: SDK Fingerprinting and Behavioral Analysis',
        body: `HTTP rate limiting catches simple scrapers. WebSocket rate limiting catches aggressive event injection. But a sophisticated bot — one that sends human-like event patterns at human-like speeds — will pass both.\n\nFor that, we need behavioral analysis.\n\nWhen the visitor SDK initializes, it collects a browser fingerprint via canvas hashing, WebGL vendor/rendering information, screen resolution, and timezone. This fingerprint isn't used for cross-site tracking — it's used for session identification and as an input to behavior analysis.\n\nThe server-side BehaviorTracker maintains per-session statistics:\n\n• Mouse event count\n• Click position distribution\n• Event type distribution (mouse, scroll, input, resize, etc.)\n• Inter-event timing (min, max, first, last)\n\nEvery 100 events, the tracker runs three heuristic checks:\n\n1. Zero mouse events with >50 total events → pure script generation. A human always generates at least some mouse movement.\n\n2. More than 20 clicks at the same (x, y) coordinate → machine pattern. Humans don't click the same pixel 20 times.\n\n3. Max interval / min interval < 2.0 with >100 events → machine-generated timing. Human event timing is irregular; bots are consistent.\n\nIf any heuristic triggers, the session is flagged in Redis (flagged:session:{id}, 10-minute TTL). The admin panel surfaces flagged sessions with a visible indicator and the flag reason.\n\nThese heuristics are simple by design. They have near-zero false positives for real visitors (humans always move their mouse and vary their timing) while catching the most common bot patterns. We don't need 99.9% bot detection accuracy — we need to make bot operation clearly more expensive than operating on a competitor's platform.`,
        code: `// Behavioral analysis: three heuristic checks
func (bt *BehaviorTracker) CheckAndFlag(sessionID string) (flagged bool, reason string) {
  bt.mu.Lock()
  defer bt.mu.Unlock()

  // Heuristic 1: no mouse movement
  if bt.totalEvents > 50 && bt.mouseEvents == 0 {
    return true, "no_mouse_movement"
  }

  // Heuristic 2: repeated clicks at same position
  for _, count := range bt.clickPositions {
    if count > 20 {
      return true, "repeated_clicks"
    }
  }

  // Heuristic 3: uniform event timing
  if bt.totalEvents > 100 && bt.minInterval > 0 {
    ratio := float64(bt.maxInterval) / float64(bt.minInterval)
    if ratio < 2.0 {
      return true, "uniform_timing"
    }
  }

  return false, ""
}`,
        codeLanguage: 'go',
      },
      {
        heading: 'Layer 4: WebSocket Authentication — No Tokens in URLs',
        body: `WebSocket connections are particularly vulnerable to auth leaks. The most common pattern — passing a JWT token as a query parameter — leaks the token to server logs, referrer headers, and browser history.\n\nPinConsole takes a different approach for operator WebSocket connections. The operator logs in via a standard form POST, receiving an HttpOnly session cookie (mm_session). This cookie:\n\n• Has SameSite=Lax — prevents CSRF from external origins\n• Is HttpOnly — inaccessible to JavaScript, preventing XSS theft\n• Is Secure in production — only sent over HTTPS\n• Has a 24-hour MaxAge — automatic session expiry\n\nWhen the admin SPA opens a WebSocket connection, the mm_session cookie is automatically included by the browser (same-origin). The server validates the cookie against Redis before accepting the WebSocket upgrade. If invalid, the connection is rejected with a 401 before any data exchange.\n\nVisitor WebSocket connections use a different mechanism. After the SDK calls POST /api/session/init, the server returns a session_id. This ID is sent in the hello message after the WebSocket connection is established. The server validates session existence and visitor-session binding before accepting events.\n\nFor both paths, the key principle is: authentication happens before the WebSocket connection is established. There's no window where an unauthenticated connection can observe traffic.`,
      },
      {
        heading: 'User-Agent Blacklist: Honest About Its Limitations',
        body: `We include a User-Agent blocklist as a lightweight first filter. It catches the lowest-effort scrapers:\n\n• curl/, wget/, python-requests/, Go-http-client/\n• HeadlessChrome, PhantomJS, jsdom\n• scrapy, bot, crawler, spider (substring match)\n• Empty User-Agent\n\nThe check is case-insensitive substring matching — deliberately simple to avoid false negatives from creative UA formatting. Blocked requests get a 403 with a descriptive error code (blocked_user_agent or empty_user_agent).\n\nBut the code comment is honest: "UA blacklist only stops low-effort scrapers; modern Puppeteer/Playwright headless=new UAs are nearly identical to real browsers." This layer catches the dumb bots. The smart ones require the deeper layers.\n\nNotably, the UA middleware runs even in development mode. It's the only security layer that does — because if you're developing with curl scripts against the API, you should know it won't work without setting a real browser UA.`,
      },
      {
        heading: 'URL Safety: Preventing Operator-Initiated Attacks',
        body: `Co-browsing gives operators the ability to navigate the visitor's browser and show popups. This is powerful — and dangerous if not constrained.\n\nNavigation commands go through isURLAllowed():\n\n• Same-origin URLs are always allowed\n• localhost URLs are allowed (for development and internal tools)\n• Cross-origin URLs are blocked unless the operator has explicitly whitelisted them\n\nPopup URLs go through isURLSchemeAllowed():\n\n• Only https: and relative URLs are permitted\n• javascript:, data:, vbscript:, file: are explicitly rejected\n\nThese checks happen server-side, in the command handler. Even if a compromised admin client sends a malicious command, the server rejects it before it reaches the visitor's SDK.\n\nThe command type itself is also whitelisted — only 8 types are accepted. Any unknown type is silently dropped at the API layer.`,
      },
      {
        heading: 'Login Brute-Force Protection and Password Policies',
        body: `The authentication system has its own dedicated protection layer.\n\nLogin attempts are rate-limited per (email, IP) pair: 5 failed attempts within 15 minutes triggers a lockout. The counter is maintained in Redis via an atomic Lua script, so concurrent requests from the same credentials can't race past the limit.\n\nRedis failure is handled gracefully here too — if Redis is down, login attempts bypass the rate check. We'd rather have a brief window of unlimited login attempts than permanently lock all operators out of the admin panel.\n\nPasswords are hashed with bcrypt at cost 12. The minimum cost is enforced at startup — if someone sets BCRYPT_COST below 12, the server refuses to start. This prevents accidental weakening of the hash cost in production.\n\nThe admin account creation flow requires the operator to change the default password on first login. The SYSTEM_EMAIL and SYSTEM_PASSWORD environment variables are validated at startup to ensure they're not default values in production mode.`,
      },
      {
        heading: 'Configuration Fail-Secure: Preventing Deployment Mistakes',
        body: `Security misconfigurations are the most common cause of breaches. We built a startup validation system that checks critical configuration before the server binds to any port:\n\n• SERVER_ENV must be one of: development, staging, production. Typos like "produciton" are caught.\n• In production mode, SYSTEM_PASSWORD must not be the default value.\n• In production mode, MinIO credentials must not be the default minioadmin/minioadmin.\n• If PostgreSQL is not on localhost, SSL mode must be enabled.\n• If MinIO endpoint is not localhost, TLS must be enabled (https://).\n\nEach check produces a specific error message: "SERVER_ENV=produciton is invalid; did you mean production?" — not a generic "configuration error". This reduces debugging time when deploying.\n\nSome checks are environment-conditional. For example, local development can use plain PostgreSQL without SSL, but production deployments to a remote database must have SSL enabled. The server adapts its strictness to the deployment context.\n\nWe also enforce the release build flag at the code level. Bypass mechanisms (like dev-mode authentication skip) are gated by //go:build !release build tags. Even if someone sets SERVER_ENV=development in production, the release binary's bypass functions always return false. The compiler enforces this — it's not a runtime check that could be misconfigured.`,
      },
      {
        heading: 'What We Don\'t Do (And Why)',
        body: `Honesty about security is important. Here's what we deliberately don't implement:\n\nNo TLS fingerprinting (JA3). Passive JA3 fingerprinting of WebSocket connections is a powerful anti-bot technique, but it requires maintaining a fingerprint database and handling false positives from browser version churn. We chose behavioral analysis instead — simpler, more transparent, and harder to evade without fundamentally changing bot behavior.\n\nNo Content Security Policy. Because the admin panel, visitor SDK, and API all share the same origin (single-binary deployment), CSP adds complexity without proportional benefit. The main attack vector CSP would prevent — XSS — is already mitigated by HttpOnly cookies and input validation.\n\nNo WAF. Self-hosted deployments have their own reverse proxy (Nginx, Caddy), and adding a WAF would create deployment friction. Security should be easy to get right.\n\nThese gaps are filled by the deployment environment: TLS termination at the reverse proxy, network-level IP blocking at the firewall, and monitoring at the infrastructure level.\n\nFor a self-hosted tool, "defense in cooperation with the deployment environment" is more realistic than "defense in a single application." The four layers we built handle application-level threats. Infrastructure threats are handled by the operator's existing security stack — because they know their network better than we do.`,
      },
      {
        heading: 'Security by Audit: How We Found and Fixed a Vulnerability',
        body: `No security system is perfect on day one. Our erasure API (DELETE /api/privacy/visitor/:fingerprint) initially had no role check — any authenticated operator could trigger a GDPR erasure. This was caught in a routine code audit (slice 1ac).\n\nThe fix was a single guard: require the admin role before processing erasure requests. The commit message notes: "role check: only admin can delete visitors."\n\nThis is a pattern we follow deliberately. Rather than trying to build a perfect security model upfront and freezing it, we ship early with reasonable guards and audit continuously. The audit log of our own fixes is transparent — there's no security-through-obscurity.\n\nIf you're deploying PinConsole, you can see every security fix in the git log. You can run the test suite to verify the fix works. And if you find a vulnerability, you can submit a fix — it's open source.`,
      },
    ],
  },
  relatedPosts: [
    { title: 'Building a Production WebSocket Hub for 500 Concurrent Connections', url: '/en/blog/websocket-hub-500-concurrent/', description: 'The real-time WebSocket architecture that powers PinConsole\'s event pipeline.' },
    { title: 'How We Built a Self-Hosted Session Replay Alternative to FullStory', url: '/en/blog/building-self-hosted-session-replay/', description: 'Architecture decisions behind PinConsole — Go, rrweb, MinIO, and single-binary deployment.' },
  ],
  cta: {
    title: 'Try Self-Hosted Session Replay',
    subtitle: 'Defense-in-depth baked in. AGPL-3.0. Your data, your servers.',
    primary: { label: 'Get started on GitHub', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: 'Talk to the maintainer', href: '#consult' },
  },
};
