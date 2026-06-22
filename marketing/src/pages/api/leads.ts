/**
 * Lead submission endpoint — Cloudflare Worker via Astro hybrid output.
 *
 * Storage: D1 (pinconsole-leads database, binding DB)
 * Anti-spam: honeypot field + Cloudflare native rate limit + format validation
 *
 * POST /api/leads
 *   { name, company, contact, purpose, message?, locale, website? }
 *   → 200 { ok: true }
 *   → 400 { error: "invalid_input" }
 *   → 429 { error: "rate_limited" }
 *   → 500 { error: "server_error" }
 */

import type { APIRoute } from 'astro';

export const prerender = false;

type LeadPurpose = 'evaluate' | 'self-host' | 'custom' | 'compliance' | 'other';

interface LeadPayload {
  name: string;
  company: string;
  contact: string;
  purpose: LeadPurpose;
  message: string | null;
  locale: 'zh' | 'en';
  website?: string; // honeypot — must be empty
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const PHONE_RE = /^[+\d][\d\s\-()]{6,}$/;
const VALID_PURPOSES: LeadPurpose[] = ['evaluate', 'self-host', 'custom', 'compliance', 'other'];

// In-memory per-IP rate limit (per worker instance).
// Cloudflare's native rate limiting rules are authoritative; this is a defensive layer.
const RATE_LIMIT_WINDOW_MS = 60_000;
const RATE_LIMIT_MAX = 5;
const rateMap = new Map<string, { count: number; resetAt: number }>();

function isRateLimited(ip: string): boolean {
  const now = Date.now();
  const entry = rateMap.get(ip);
  if (!entry || entry.resetAt < now) {
    rateMap.set(ip, { count: 1, resetAt: now + RATE_LIMIT_WINDOW_MS });
    return false;
  }
  entry.count += 1;
  return entry.count > RATE_LIMIT_MAX;
}

function truncateIp(ip: string): string {
  // IPv4: /24 truncation (privacy). IPv6: /64.
  if (ip.includes('.')) {
    const parts = ip.split('.');
    return `${parts[0]}.${parts[1]}.${parts[2]}.0`;
  }
  if (ip.includes(':')) {
    return ip.split(':').slice(0, 4).join(':') + '::';
  }
  return 'unknown';
}

function isValid(payload: Partial<LeadPayload>): payload is LeadPayload {
  if (!payload) return false;
  if (typeof payload.name !== 'string' || payload.name.trim().length < 1 || payload.name.length > 200) return false;
  if (typeof payload.company !== 'string' || payload.company.trim().length < 1 || payload.company.length > 200) return false;
  if (typeof payload.contact !== 'string' || !EMAIL_RE.test(payload.contact) && !PHONE_RE.test(payload.contact)) return false;
  if (!VALID_PURPOSES.includes(payload.purpose)) return false;
  if (payload.message != null && (typeof payload.message !== 'string' || payload.message.length > 5000)) return false;
  if (payload.locale !== 'zh' && payload.locale !== 'en') return false;
  return true;
}

export const POST: APIRoute = async ({ request, locals }) => {
  const env = (locals.runtime?.env ?? {}) as CloudflareEnv;
  const db = env.DB;

  if (!db) {
    console.error('D1 binding missing — check wrangler.toml [[d1_databases]]');
    return new Response(JSON.stringify({ error: 'server_error' }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' },
    });
  }

  const ipRaw = request.headers.get('CF-Connecting-IP') || request.headers.get('X-Forwarded-For') || 'unknown';
  const ip = truncateIp(ipRaw);

  if (isRateLimited(ip)) {
    return new Response(JSON.stringify({ error: 'rate_limited' }), {
      status: 429,
      headers: { 'Content-Type': 'application/json', 'Retry-After': '60' },
    });
  }

  let payload: Partial<LeadPayload>;
  try {
    payload = await request.json();
  } catch {
    return new Response(JSON.stringify({ error: 'invalid_input' }), {
      status: 400,
      headers: { 'Content-Type': 'application/json' },
    });
  }

  // Honeypot — silently accept so bots don't retry.
  if (payload.website) {
    return new Response(JSON.stringify({ ok: true }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    });
  }

  if (!isValid(payload)) {
    return new Response(JSON.stringify({ error: 'invalid_input' }), {
      status: 400,
      headers: { 'Content-Type': 'application/json' },
    });
  }

  const ua = request.headers.get('User-Agent')?.slice(0, 500) ?? null;
  const now = Date.now();

  try {
    await db
      .prepare(
        `INSERT INTO leads (name, company, contact, purpose, message, locale, ip, ua, created_at, status)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'new')`
      )
      .bind(
        payload.name,
        payload.company,
        payload.contact,
        payload.purpose,
        payload.message ?? null,
        payload.locale,
        ip,
        ua,
        now,
      )
      .run();

    // Optional notification hooks (configured via wrangler secrets)
    if (env.LEAD_NOTIFY_WEBHOOK) {
      try {
        await fetch(env.LEAD_NOTIFY_WEBHOOK, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            text: `New pinconsole lead: ${payload.name} @ ${payload.company} (${payload.purpose})`,
          }),
        });
      } catch (err) {
        console.error('webhook notify failed', err);
      }
    }

    return new Response(JSON.stringify({ ok: true }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    });
  } catch (err) {
    console.error('lead insert failed', err);
    return new Response(JSON.stringify({ error: 'server_error' }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' },
    });
  }
};

// CORS / preflight — same-origin only.
export const OPTIONS: APIRoute = () =>
  new Response(null, {
    status: 204,
    headers: {
      'Access-Control-Allow-Origin': 'null',
      'Access-Control-Allow-Methods': 'POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type',
    },
  });
