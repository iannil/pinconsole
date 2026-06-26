// DB/Redis fixtures for e2e tests.
//
// 用于需要预先 seed 状态的测试(1w flagged session、1x throttle counter)。
// 暴露 paired seed/cleanup 函数 + Playwright auto-cleanup fixture wrapper。
//
// 连接参数从根目录 .env 读(PG_*/REDIS_*),与 server 共用同一套。
//
// Schema 说明(读 server/migrations/*.sql 校验过):
// - visitors 表:PK 是 id(UUID),fingerprint 文本列,无 fingerprint_hash
// - sessions 表:PK 是 id(UUID),visitor_id 引用 visitors(id),
//   status 列 CHECK IN ('active','ended','timed_out'),
//   **没有 flagged/flag_reason 列**
// - flag 存在 Redis:key=flagged:session:<id>,value=reason,TTL=10min
//   (见 server/internal/antiscrape/ratelimit.go:134 FlagSession)

import pg from 'pg';
import { createClient, type RedisClientType } from 'redis';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { randomUUID } from 'node:crypto';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const PROJECT_ROOT = resolve(__dirname, '..', '..');

interface DBConfig {
  pgUser: string;
  pgPassword: string;
  pgHost: string;
  pgPort: number;
  pgDatabase: string;
  redisHost: string;
  redisPort: number;
}

function loadEnv(): DBConfig {
  const envPath = resolve(PROJECT_ROOT, '.env');
  const cfg: DBConfig = {
    pgUser: 'mm',
    pgPassword: 'mm_dev',
    pgHost: 'localhost',
    pgPort: 7032,
    pgDatabase: 'pinconsole',
    redisHost: 'localhost',
    redisPort: 7079,
  };
  try {
    const raw = readFileSync(envPath, 'utf-8');
    for (const line of raw.split('\n')) {
      const m = line.match(/^\s*(PG_USER|PG_PASSWORD|PG_HOST|PG_PORT|PG_DB|REDIS_HOST|REDIS_PORT)\s*=\s*(.+?)\s*$/);
      if (!m) continue;
      const val = m[2].replace(/^["']|["']$/g, '');
      switch (m[1]) {
        case 'PG_USER': cfg.pgUser = val; break;
        case 'PG_PASSWORD': cfg.pgPassword = val; break;
        case 'PG_HOST': cfg.pgHost = val; break;
        case 'PG_PORT': cfg.pgPort = parseInt(val, 10); break;
        case 'PG_DB': cfg.pgDatabase = val; break;
        case 'REDIS_HOST': cfg.redisHost = val; break;
        case 'REDIS_PORT': cfg.redisPort = parseInt(val, 10); break;
      }
    }
  } catch {
    // 默认值
  }
  return cfg;
}

const CFG = loadEnv();

let _pool: pg.Pool | null = null;
let _redis: RedisClientType | null = null;

function pool(): pg.Pool {
  if (!_pool) {
    _pool = new pg.Pool({
      user: CFG.pgUser,
      password: CFG.pgPassword,
      host: CFG.pgHost,
      port: CFG.pgPort,
      database: CFG.pgDatabase,
    });
  }
  return _pool;
}

async function redis(): Promise<RedisClientType> {
  if (!_redis) {
    _redis = createClient({ url: `redis://${CFG.redisHost}:${CFG.redisPort}` }) as RedisClientType;
    await _redis.connect();
  }
  return _redis;
}

export interface SeededFlaggedSession {
  sessionId: string;
  visitorId: string;
  reason: string;
  cleanup: () => Promise<void>;
}

// 创建一条 ended 状态的 session(visitor + session row),并把 Redis flag 设上。
// 用于 1w 测试:验证 admin 列表能显示 flagged 标记。
// cleanup 同时删 PG row + Redis flag key。
export async function seedFlaggedSession(reason = 'e2e-test-flag'): Promise<SeededFlaggedSession> {
  const p = pool();
  const r = await redis();
  const visitorId = randomUUID();
  const sessionId = randomUUID();

  await p.query(
    `INSERT INTO visitors (id, fingerprint, first_seen_at, last_seen_at)
     VALUES ($1, $2, NOW(), NOW())`,
    [visitorId, `e2e-${visitorId.slice(0, 8)}`],
  );

  await p.query(
    `INSERT INTO sessions (id, visitor_id, started_at, ended_at, status)
     VALUES ($1, $2, NOW(), NOW(), 'ended')`,
    [sessionId, visitorId],
  );

  // 与 antiscrape.FlagSession 一致:key=flagged:session:<id>,value=reason,TTL=10min。
  // 但测试场景下我们希望 TTL 长一些不要测试中途消失 — 用 1h。
  await r.set(`flagged:session:${sessionId}`, reason, { EX: 3600 });

  return {
    sessionId,
    visitorId,
    reason,
    cleanup: async () => {
      await r.del(`flagged:session:${sessionId}`);
      await p.query('DELETE FROM sessions WHERE id = $1', [sessionId]);
      await p.query('DELETE FROM visitors WHERE id = $1', [visitorId]);
    },
  };
}

// 把 login throttle counter 设到指定值(用于测 1x 边界)。
// key 形如 auth:throttle:<email>:<ip>(见 server/internal/api/auth.go:179)。
export async function setLoginThrottleCounter(email: string, ip: string, count: number): Promise<void> {
  const r = await redis();
  const key = `auth:throttle:${email}:${ip}`;
  // 与 recordLoginFailure 一致:15min TTL(loginLockoutWindow)
  await r.set(key, String(count), { EX: 900 });
}

export async function clearLoginThrottle(email: string, ip: string): Promise<void> {
  const r = await redis();
  const key = `auth:throttle:${email}:${ip}`;
  await r.del(key);
}

// 清所有 throttle key(全局)。
export async function clearAllLoginThrottles(): Promise<void> {
  const r = await redis();
  const keys = await r.keys('auth:throttle:*');
  if (keys.length > 0) {
    await r.del(keys);
  }
}

// 关闭所有连接。
export async function closeDBFixtures(): Promise<void> {
  if (_pool) {
    await _pool.end();
    _pool = null;
  }
  if (_redis) {
    await _redis.quit();
    _redis = null;
  }
}
