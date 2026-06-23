/**
 * Lead submission endpoint — Cloudflare Pages Function via Astro hybrid output.
 *
 * Storage: D1 (pinconsole-leads database, binding DB)
 * Anti-spam: honeypot field + in-memory rate limit + Cloudflare Turnstile verification
 * Notification: outbound email via Resend (RESEND_API_KEY + LEAD_NOTIFY_EMAIL secrets)
 *
 * POST /api/leads
 *   { name, company, contact, purpose, message?, locale, website?, turnstileToken? }
 *   → 200 { ok: true }
 *   → 400 { error: "invalid_input" | "turnstile_failed" }
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
  turnstileToken?: string | null;
}

interface D1Statement {
  bind(...values: unknown[]): D1Statement;
  run(): Promise<unknown>;
}
interface D1Database {
  prepare(sql: string): D1Statement;
}

interface CloudflareEnv {
  DB: D1Database;
  RESEND_API_KEY?: string;
  LEAD_NOTIFY_EMAIL?: string;
  TURNSTILE_SECRET?: string;
  TURNSTILE_SITE_KEY?: string;
  SITE_URL?: string;
  CONTACT_EMAIL?: string;
}

interface AstroLocals {
  runtime?: { env?: CloudflareEnv };
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const PHONE_RE = /^[+\d][\d\s\-()]{6,}$/;
const VALID_PURPOSES: readonly LeadPurpose[] = ['evaluate', 'self-host', 'custom', 'compliance', 'other'];

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

function isValidPayload(payload: Partial<LeadPayload>): payload is LeadPayload {
  if (!payload) return false;
  if (typeof payload.name !== 'string' || payload.name.trim().length < 1 || payload.name.length > 200) return false;
  if (typeof payload.company !== 'string' || payload.company.trim().length < 1 || payload.company.length > 200) return false;
  if (typeof payload.contact !== 'string' || (!EMAIL_RE.test(payload.contact) && !PHONE_RE.test(payload.contact))) return false;
  if (!VALID_PURPOSES.includes(payload.purpose as LeadPurpose)) return false;
  if (payload.message != null && (typeof payload.message !== 'string' || payload.message.length > 5000)) return false;
  if (payload.locale !== 'zh' && payload.locale !== 'en') return false;
  return true;
}

async function verifyTurnstile(token: string, secret: string, ip: string): Promise<boolean> {
  try {
    const res = await fetch('https://challenges.cloudflare.com/turnstile/v0/siteverify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({ response: token, secret, remoteip: ip }),
    });
    const data = (await res.json()) as { success: boolean };
    return data.success === true;
  } catch {
    return false;
  }
}

async function sendLeadEmailDebug(
  env: CloudflareEnv,
  payload: LeadPayload,
  ip: string,
): Promise<{ ok: boolean; status: number; body: string }> {
  if (!env.RESEND_API_KEY) {
    return { ok: false, status: 0, body: 'RESEND_API_KEY missing in env' };
  }
  if (!env.LEAD_NOTIFY_EMAIL) {
    return { ok: false, status: 0, body: 'LEAD_NOTIFY_EMAIL missing in env' };
  }

  const siteUrl = env.SITE_URL || 'https://pinconsole.com';
  const subject = `[PinConsole lead] ${payload.name} @ ${payload.company}`;
  const purposeLabels: Record<LeadPurpose, string> = {
    evaluate: '评估替代现有 SaaS',
    'self-host': '自托管部署咨询',
    custom: '定制开发',
    compliance: '合规咨询',
    other: '其他',
  };
  const safeName = escapeHtml(payload.name);
  const safeCompany = escapeHtml(payload.company);
  const safeContact = escapeHtml(payload.contact);
  const safeIp = escapeHtml(ip);
  const safePurpose = escapeHtml(purposeLabels[payload.purpose]);
  const safeLocale = escapeHtml(payload.locale);
  const dateStr = new Date().toISOString().replace('T', ' ').substring(0, 16) + ' UTC';
  const replySubject = encodeURIComponent(`Re: PinConsole consultation`);
  const messageBlock = payload.message
    ? `
        <tr>
          <td style="padding:24px 40px 8px;">
            <div style="font-family:ui-monospace,'SF Mono',Monaco,Consolas,monospace;font-size:10px;font-weight:600;letter-spacing:0.1em;color:#999;text-transform:uppercase;margin-bottom:8px;">MESSAGE</div>
            <div style="padding:16px 20px;background-color:#FAFAF7;border-left:2px solid #0F766E;font-size:14px;line-height:1.6;color:#333;white-space:pre-wrap;">${escapeHtml(payload.message)}</div>
          </td>
        </tr>
      `
    : '';

  const html = `<!DOCTYPE html>
<html lang="${safeLocale === 'zh' ? 'zh' : 'en'}">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="color-scheme" content="light only">
  <meta name="supported-color-schemes" content="light only">
  <title>PinConsole lead</title>
</head>
<body style="margin:0;padding:0;background-color:#FAFAF7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',system-ui,sans-serif;color:#1A1A1A;font-size:15px;line-height:1.5;-webkit-font-smoothing:antialiased;">

  <div style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;">${safeName} from ${safeCompany} · ${safePurpose}</div>

  <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="background-color:#FAFAF7;">
    <tr>
      <td align="center" style="padding:40px 16px;">

        <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="600" style="max-width:600px;width:100%;background-color:#FFFFFF;border:1px solid #E8E5DD;">

          <tr>
            <td style="height:3px;background-color:#0F766E;font-size:0;line-height:0;">&nbsp;</td>
          </tr>

          <tr>
            <td style="padding:32px 40px 24px;border-bottom:1px solid #E8E5DD;">
              <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%">
                <tr>
                  <td style="vertical-align:top;">
                    <div style="font-family:ui-monospace,'SF Mono',Monaco,Consolas,monospace;font-size:11px;font-weight:600;letter-spacing:0.12em;color:#0F766E;text-transform:uppercase;">NEW LEAD</div>
                    <div style="margin-top:8px;font-size:22px;font-weight:600;color:#1A1A1A;letter-spacing:-0.01em;line-height:1.3;">${safeName} <span style="color:#999;font-weight:400;margin:0 4px;">—</span> ${safeCompany}</div>
                  </td>
                  <td align="right" style="vertical-align:top;">
                    <div style="font-family:ui-monospace,'SF Mono',Monaco,Consolas,monospace;font-size:11px;color:#999;letter-spacing:0.04em;">${dateStr}</div>
                  </td>
                </tr>
              </table>
            </td>
          </tr>

          <tr>
            <td style="padding:8px 40px 0;">
              <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%">
                <tr>
                  <td style="padding:14px 0;border-bottom:1px solid #F0EEE8;width:96px;vertical-align:top;">
                    <div style="font-family:ui-monospace,'SF Mono',Monaco,Consolas,monospace;font-size:10px;font-weight:600;letter-spacing:0.1em;color:#999;text-transform:uppercase;">CONTACT</div>
                  </td>
                  <td style="padding:14px 16px;border-bottom:1px solid #F0EEE8;color:#1A1A1A;">
                    <a href="mailto:${safeContact}" style="color:#0F766E;text-decoration:none;font-weight:500;">${safeContact}</a>
                  </td>
                </tr>
                <tr>
                  <td style="padding:14px 0;border-bottom:1px solid #F0EEE8;width:96px;vertical-align:top;">
                    <div style="font-family:ui-monospace,'SF Mono',Monaco,Consolas,monospace;font-size:10px;font-weight:600;letter-spacing:0.1em;color:#999;text-transform:uppercase;">PURPOSE</div>
                  </td>
                  <td style="padding:14px 16px;border-bottom:1px solid #F0EEE8;color:#1A1A1A;">${safePurpose}</td>
                </tr>
                <tr>
                  <td style="padding:14px 0;border-bottom:1px solid #F0EEE8;width:96px;vertical-align:top;">
                    <div style="font-family:ui-monospace,'SF Mono',Monaco,Consolas,monospace;font-size:10px;font-weight:600;letter-spacing:0.1em;color:#999;text-transform:uppercase;">LOCALE</div>
                  </td>
                  <td style="padding:14px 16px;border-bottom:1px solid #F0EEE8;color:#1A1A1A;">${safeLocale}</td>
                </tr>
                <tr>
                  <td style="padding:14px 0;width:96px;vertical-align:top;">
                    <div style="font-family:ui-monospace,'SF Mono',Monaco,Consolas,monospace;font-size:10px;font-weight:600;letter-spacing:0.1em;color:#999;text-transform:uppercase;">IP</div>
                  </td>
                  <td style="padding:14px 16px;color:#666;font-family:ui-monospace,'SF Mono',Monaco,Consolas,monospace;font-size:12px;">${safeIp}</td>
                </tr>
              </table>
            </td>
          </tr>
          ${messageBlock}
          <tr>
            <td style="padding:32px 40px 8px;">
              <table role="presentation" cellpadding="0" cellspacing="0" border="0">
                <tr>
                  <td style="background-color:#0F766E;border-radius:4px;">
                    <a href="mailto:${safeContact}?subject=${replySubject}" style="display:inline-block;padding:12px 24px;color:#FFFFFF;text-decoration:none;font-size:14px;font-weight:600;letter-spacing:0.02em;">Reply to ${safeName} →</a>
                  </td>
                </tr>
              </table>
            </td>
          </tr>

          <tr>
            <td style="padding:28px 40px 32px;">
              <div style="border-top:1px solid #E8E5DD;padding-top:16px;font-size:12px;color:#999;line-height:1.6;">
                <div style="font-family:ui-monospace,'SF Mono',Monaco,Consolas,monospace;font-size:10px;letter-spacing:0.08em;text-transform:uppercase;color:#666;margin-bottom:4px;">REPLY WINDOW · 48H</div>
                Sent from <a href="${siteUrl}" style="color:#0F766E;text-decoration:none;">pinconsole.com</a> · Stored in your Cloudflare D1.
              </div>
            </td>
          </tr>

        </table>

        <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="600" style="max-width:600px;width:100%;">
          <tr>
            <td style="padding:16px 0;text-align:center;font-family:ui-monospace,'SF Mono',Monaco,Consolas,monospace;font-size:10px;color:#999;letter-spacing:0.08em;text-transform:uppercase;">
              PinConsole · Open source ToB customer ops · AGPL-3.0
            </td>
          </tr>
        </table>

      </td>
    </tr>
  </table>

</body>
</html>`;

  const res = await fetch('https://api.resend.com/emails', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${env.RESEND_API_KEY}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      from: 'PinConsole Marketing <noreply@pinconsole.com>',
      to: [env.LEAD_NOTIFY_EMAIL],
      reply_to: payload.contact,
      subject,
      html,
    }),
  });
  const body = await res.text();
  return { ok: res.ok, status: res.status, body };
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

export const POST: APIRoute = async ({ request, locals }) => {
  const env = (locals as AstroLocals).runtime?.env ?? ({} as CloudflareEnv);
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
    payload = (await request.json()) as Partial<LeadPayload>;
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

  if (!isValidPayload(payload)) {
    return new Response(JSON.stringify({ error: 'invalid_input' }), {
      status: 400,
      headers: { 'Content-Type': 'application/json' },
    });
  }

  // Turnstile verification — only enforced when secret is configured
  if (env.TURNSTILE_SECRET && env.TURNSTILE_SITE_KEY) {
    if (!payload.turnstileToken) {
      return new Response(JSON.stringify({ error: 'turnstile_failed' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' },
      });
    }
    const ok = await verifyTurnstile(payload.turnstileToken, env.TURNSTILE_SECRET, ipRaw);
    if (!ok) {
      return new Response(JSON.stringify({ error: 'turnstile_failed' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' },
      });
    }
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

    // Email notification — surface error to client for debugging
    let emailStatus: 'sent' | 'skipped' | { error: string } = 'skipped';
    try {
      const result = await sendLeadEmailDebug(env, payload, ip);
      emailStatus = result.ok ? 'sent' : { error: `${result.status}: ${result.body}` };
    } catch (err) {
      emailStatus = { error: String(err) };
    }

    return new Response(JSON.stringify({ ok: true, email: emailStatus }), {
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
