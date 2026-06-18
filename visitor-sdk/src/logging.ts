// 1m-observability:SDK 端结构化 logger
// dev 模式 console.warn(JSON.stringify({...}));prod 模式仅 error 级别输出
// 替代散落各处的 console.* 调用(CLAUDE.md 要求 "所有日志输出必须是 JSON 格式")

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export interface LogPayload {
  event: string;
  [key: string]: unknown;
}

const LEVEL_ORDER: Record<LogLevel, number> = {
  debug: 10,
  info: 20,
  warn: 30,
  error: 40,
};

// 当前 log level:dev 默认 debug,prod 默认 error
// 通过 URL 参数 ?mm_log=info 或 localStorage.mm_log 调
let currentLevel: LogLevel = (() => {
  if (typeof window === 'undefined') return 'error';
  try {
    const url = new URL(window.location.href);
    const fromUrl = url.searchParams.get('mm_log');
    if (fromUrl && ['debug', 'info', 'warn', 'error'].includes(fromUrl)) {
      return fromUrl as LogLevel;
    }
    const fromStorage = window.localStorage?.getItem('mm_log');
    if (fromStorage && ['debug', 'info', 'warn', 'error'].includes(fromStorage)) {
      return fromStorage as LogLevel;
    }
    // 默认:dev 模式 debug,prod error
    return url.hostname === 'localhost' || url.hostname === '127.0.0.1' ? 'debug' : 'error';
  } catch {
    return 'error';
  }
})();

export function setLogLevel(level: LogLevel): void {
  currentLevel = level;
  try {
    window.localStorage?.setItem('mm_log', level);
  } catch {
    // ignore
  }
}

function shouldLog(level: LogLevel): boolean {
  return LEVEL_ORDER[level] >= LEVEL_ORDER[currentLevel];
}

function emit(level: LogLevel, payload: LogPayload): void {
  if (!shouldLog(level)) return;

  const enriched: LogPayload & { level: string; ts: string; source: string } = {
    ...payload,
    level,
    ts: new Date().toISOString(),
    source: 'visitor-sdk',
  };

  const json = JSON.stringify(enriched);

  // eslint-disable-next-line no-console
  switch (level) {
    case 'debug':
      // eslint-disable-next-line no-console
      console.debug(json);
      break;
    case 'info':
      // eslint-disable-next-line no-console
      console.info(json);
      break;
    case 'warn':
      // eslint-disable-next-line no-console
      console.warn(json);
      break;
    case 'error':
      // eslint-disable-next-line no-console
      console.error(json);
      break;
  }
}

export const sdkLogger = {
  debug: (event: string, extras: Record<string, unknown> = {}) => emit('debug', { event, ...extras }),
  info: (event: string, extras: Record<string, unknown> = {}) => emit('info', { event, ...extras }),
  warn: (event: string, extras: Record<string, unknown> = {}) => emit('warn', { event, ...extras }),
  error: (event: string, extras: Record<string, unknown> = {}) => emit('error', { event, ...extras }),
};

// 1m:trace_id 生成(crypto getRandomValues 16字节 hex,与服务端格式一致)
export function generateTraceId(): string {
  const bytes = new Uint8Array(16);
  if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
    crypto.getRandomValues(bytes);
  } else {
    // fallback(Math.random,弱但 MSYS 兼容)
    for (let i = 0; i < 16; i++) bytes[i] = Math.floor(Math.random() * 256);
  }
  return Array.from(bytes).map((b) => b.toString(16).padStart(2, '0')).join('');
}
