/// <reference path="../.astro/types.d.ts" />
/// <reference types="@cloudflare/workers-types" />

interface CloudflareEnv {
  DB: D1Database;
  LEAD_NOTIFY_EMAIL?: string;
  LEAD_NOTIFY_WEBHOOK?: string;
  TURNSTILE_SECRET?: string;
}

interface Runtime {
  env: CloudflareEnv;
}
