import type { BlogContent } from './types';

export const privacyEn: BlogContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'Privacy by Design in Open-Source Session Replay: Building GDPR-First Surveillance',
    description: 'How PinConsole implements privacy-first session monitoring — opt-in consent, selective masking, right-to-be-forgotten, and data retention. A technical guide to building GDPR-compliant visitor surveillance.',
    ogTitle: 'Privacy by Design in Open-Source Session Replay',
    ogDescription: 'Opt-in consent, selective masking, right-to-be-forgotten, and 30-day retention — building GDPR-first visitor surveillance in PinConsole.',
  },
  blog: {
    author: 'Rong Zhu',
    publishedDate: '2026-06-28',
    readingTime: '9 min read',
    tags: ['Privacy', 'GDPR', 'Security', 'Engineering'],
  },
  hero: {
    h1: 'Privacy by Design in Open-Source Session Replay: Building GDPR-First Surveillance',
    subtitle: 'How to build visitor monitoring that respects privacy by default — consent, masking, erasure, and retention, implemented from day one.',
  },
  content: {
    sections: [
      {
        heading: 'The Privacy Paradox of Session Replay',
        body: `Session replay tools are surveillance tools. They record every click, scroll, input, and navigation your visitors make. In the hands of a product team, this data is invaluable for UX optimization, bug reproduction, and conversion analysis. In the wrong context, it's a privacy nightmare.\n\nEvery major session replay tool faces this tension. FullStory, Hotjar, LogRocket — they all collect by default and ask you to opt out. Data leaves your infrastructure, stored on third-party servers across jurisdictions. GDPR Article 44 (international transfers) becomes an immediate compliance risk.\n\nWe took a different approach with PinConsole. Because we're self-hosted, our starting point is: data never leaves your infrastructure. But that's table stakes. The real design challenge is building privacy into the surveillance pipeline itself — not as an afterthought, not as a compliance checkbox, but as a first-class system property.\n\nThis is how we did it.`,
      },
      {
        heading: 'Default to Masked: Privacy at the SDK Level',
        body: `The first line of defense isn't a database permission or an API guard — it's the visitor SDK itself. Before any data leaves the visitor's browser, we decide what should never be collected.\n\nBy default, every input field is masked. When UNMASK_INPUTS is false (the default), rrweb records <input> and <textarea> values as asterisks (***). The raw text never reaches the server — it's stripped at the SDK level before transmission.\n\nEven when UNMASK_INPUTS is true, password fields remain unconditionally masked. This is hardcoded — no configuration can remove it. It's a safety floor, not a setting.\n\nBeyond inputs, we support two CSS-based masking mechanisms:\n\n• blockClass ('mm-block'): Elements with this class are completely excluded from recording. No snapshot, no mutation tracking, no trace. Use it for elements that should never appear in any recording — credit card iframes, medical data displays, internal admin panels.\n\n• ignoreClass ('mm-ignore'): Elements with this class are recorded as placeholders. Their position, size, and layout are preserved for accurate replay, but their content is stripped. Use it for dynamic content panels where you need the visual context but not the actual data.\n\nBoth mechanisms operate before the data leaves the browser. There's no server-side processing that could accidentally expose masked content. The SDK is the privacy boundary.`,
        code: `// SDK config: privacy-first defaults
const config = {
  unmaskInputs: false,        // Default: mask all inputs
  maskInputOptions: {
    password: true,            // Always mask, not configurable
    email: true,               // Masked by default
    tel: true,                 // Masked by default
  },
  blockClass: 'mm-block',     // Excluded from recording entirely
  ignoreClass: 'mm-ignore',   // Visible as placeholder, content stripped
};`,
        codeLanguage: 'typescript',
      },
      {
        heading: 'Consent Modes: Four Models, One Framework',
        body: `Different sites have different consent requirements. A SaaS dashboard used by logged-in employees has different privacy expectations than a public e-commerce store that serves EU visitors. Rather than a single consent model, we built four modes into the SDK:\n\n• opt-in (default): No data is collected until the visitor explicitly accepts. The banner is displayed immediately on page load. This is the safest default for GDPR compliance.\n\n• opt-out: Data is collected immediately, but the visitor can decline. If they reject, collection stops. Suitable for sites where session replay is essential for operations but consent is still legally required.\n\n• always-on: Collection starts without any banner. For internal tools or authenticated sessions where consent is implied by the employment contract or terms of service.\n\n• always-off: Collection never starts. For pages or visitors that should never be recorded under any circumstances.\n\nThe consent state is persisted server-side. When the SDK initializes, it calls GET /api/privacy/consent?fingerprint=<hash> to check for an existing consent decision. If none exists and the mode is opt-in, the consent banner is shown.\n\nThe banner itself is a centered modal with a backdrop blur — visually unmistakable. It contains:\n\n• A brief explanation of what data is collected\n• Accept and Reject buttons\n• An optional link to your privacy policy\n\nThe banner text is customizable through the admin panel. You can set separate messages for each site, in any language. For EU deployments, we recommend explaining the data processing purpose ("We record your session to improve our website") as required by GDPR Article 13.`,
      },
      {
        heading: 'The SDK Won\'t Start Without Consent',
        body: `This sounds obvious, but it's not how most session replay tools work. Many SDKs initialize the recorder immediately and only check consent state before transmitting — meaning the DOM snapshot has already been captured in memory.\n\nPinConsole's SDK takes a stricter approach. The shouldCollectSurveillance() function is the gatekeeper:\n\n1. Check consent mode\n2. If opt-in: only proceed if an accepted consent record exists\n3. If opt-out: proceed unless a rejected consent record exists\n4. If always-on: proceed unconditionally\n5. If always-off: never proceed\n\nWhen the check fails, no rrweb recorder is created. No DOM snapshot is taken. No WebSocket connection is established. There is no data to "forget" later because there was never any data.\n\nThis has a real performance benefit too — visitors who haven't consented don't incur the ~15KB SDK download or the WebSocket connection overhead. Privacy and performance align.\n\nWhen a visitor withdraws consent (via setConsent(false)), the rrweb recorder is stopped immediately. Any buffered events that haven't been transmitted are discarded. The WebSocket connection is closed. The visitor's privacy choice is honored within the same event loop tick.`,
      },
      {
        heading: 'Right to Be Forgotten: Cascade Deletion Across All Storage Layers',
        body: `GDPR Article 17 gives visitors the right to have their data erased. In a self-hosted session replay tool, this means deleting data from three storage systems: PostgreSQL, MinIO, and Redis.\n\nWhen an operator submits a deletion request (via the /privacy admin page), the server executes this cascade:\n\n1. Find the visitor by fingerprint, collect all associated session IDs\n2. List all MinIO object keys from the event_blobs table (before deleting the rows — the keys are the reference)\n3. Delete PostgreSQL rows in dependency order: visitor_consents → chat_messages → co_browsing_commands → event_blobs → sessions → visitors\n4. Delete MinIO objects for each event key (best-effort, doesn't block the response)\n5. Delete Redis claim keys for each session (best-effort)\n\nA notable design choice: each step commits independently. There is no overarching transaction. The rationale is "err on the side of deletion" — if the database deletion succeeds but a MinIO object delete fails, the visitor's metadata is still gone. The orphaned event blob is harmless and will be cleaned up by the GC worker.\n\nIf the visitor has already been deleted (duplicate request), the endpoint returns 200 OK with a note — not a 404. This prevents information leakage about whether a specific fingerprint existed.\n\nThe operator who performs the deletion needs the admin role — a safety check we added after a code audit caught that the initial implementation allowed any operator to trigger erasure.`,
        code: `// Cascade deletion order (each step commits independently)
func (r *ErasureRepo) DeleteVisitorByFingerprint(ctx context.Context, fp string) error {
  // 1. Visitor consents
  r.db.Exec(ctx, "DELETE FROM visitor_consents WHERE fingerprint = $1", fp)
  // 2. Chat messages (via sessions)
  r.db.Exec(ctx, "DELETE FROM chat_messages WHERE session_id IN (SELECT id FROM sessions WHERE visitor_id IN (SELECT id FROM visitors WHERE fingerprint = $1))", fp)
  // 3. Co-browsing commands
  r.db.Exec(ctx, "DELETE FROM co_browsing_commands WHERE session_id IN (...)")
  // 4. Event blobs
  r.db.Exec(ctx, "DELETE FROM event_blobs WHERE session_id IN (...)")
  // 5. Sessions
  r.db.Exec(ctx, "DELETE FROM sessions WHERE visitor_id IN (...)")
  // 6. Visitors
  r.db.Exec(ctx, "DELETE FROM visitors WHERE fingerprint = $1", fp)
  return nil
}`,
        codeLanguage: 'go',
      },
      {
        heading: 'Data Retention: GC by Design',
        body: `Session replay data accumulates fast. A medium-traffic site can generate gigabytes of event data per week. Without a retention policy, storage costs grow indefinitely and privacy risk accumulates with every byte.\n\nPinConsole runs a garbage collection (GC) worker that runs hourly. Its job is to delete sessions that have ended and exceeded the retention period. By default, that's 30 days — matching common SaaS practice.\n\nThe GC is thorough. It doesn't just delete event_blobs. It cleans up across five tables in reverse dependency order, mirroring the erasure cascade:\n\n1. event_blobs (MinIO objects + PG rows)\n2. chat_messages\n3. co_browsing_commands\n4. sessions\n5. visitors (last_seen_at < threshold)\n\nEach batch processes up to 1000 records, preventing long-running transactions. If a batch fails partway through, the next GC cycle picks up where it left off. The worker runs immediately on startup and then on an hourly ticker.\n\nThe retention period is configurable via the RETENTION_DAYS environment variable. Operators can set different retention for different sites — 7 days for high-traffic marketing pages, 90 days for critical onboarding flows, forever for compliance-required audit trails.\n\nWe deliberately did NOT use MinIO bucket lifecycle policies for deletion. Deletion is driven by application logic, not storage-layer rules. This ensures the GC respects session boundaries (partial session cleanup would break replay), coordinates with PG metadata deletion, and logs what it removes.`,
        code: `// GC worker: runs hourly, processes in batches
func (w *GCWorker) Run(ctx context.Context) {
  ticker := time.NewTicker(1 * time.Hour)
  defer ticker.Stop()

  // Run immediately on startup
  w.gcOnce(ctx)

  for {
    select {
    case <-ticker.C:
      w.gcOnce(ctx)
    case <-ctx.Done():
      return
    }
  }
}

func (w *GCWorker) gcOnce(ctx context.Context) {
  threshold := time.Now().Add(-w.cfg.RetentionPeriod)
  // event_blobs → chat_messages → co_browsing_commands
  // → sessions → visitors
  w.repo.DeleteSessionsEndedBefore(ctx, threshold, 1000)
}`,
        codeLanguage: 'go',
      },
      {
        heading: 'Co-Browsing Privacy: Consent Before Control',
        body: `Co-browsing raises an entirely different privacy concern. It's not just recording what happened — it's giving a human operator real-time access to a visitor's screen. The privacy implications are immediate and visceral.\n\nWhen an operator initiates co-browsing, the visitor's browser displays a banner before any operator input reaches the page:\n\n• An eye icon (visual indicator of observation)\n• The operator's name (transparency about who is watching)\n• An "Exit" button (instant termination)\n\nThis banner persists for the entire co-browsing session. It's not a one-time acceptance — it's a continuous awareness indicator. If the visitor feels uncomfortable at any point, they can terminate with a single click, or by pressing Escape three times within one second.\n\nFor the most sensitive deployments (financial services, healthcare), we support a passive monitoring mode. In this mode, the operator can see the visitor's screen but cannot interact until the visitor explicitly clicks "Allow assistance." This ensures the operator never performs an action the visitor didn't explicitly authorize.\n\nEvery co-browsing command is logged to the co_browsing_commands table with the operator_id. If a privacy complaint arises, there's a complete audit trail of exactly what the operator did, when, and on which session.`,
      },
      {
        heading: 'IP Handling and Data Minimization',
        body: `Data minimization (GDPR Article 5(1)(c)) means collecting only what you need. For session replay, you need DOM events — you generally don't need the visitor's IP address beyond basic rate limiting.\n\nPinConsole does not store visitor IP addresses permanently. The server may use the IP for:\n\n• Rate limiting (in-memory counter, not persisted)\n• Geolocation for analytics (country-level only, stored as ISO code, not IP)\n\nThe raw IP is never written to the sessions table or any log. If you need stricter controls, the server can be configured behind a reverse proxy that strips the X-Forwarded-For header before it reaches the application.\n\nWe also minimize the session identifier. Each session is identified by a server-generated UUID — not by a visitor ID cookie, not by a fingerprint hash. There's no cross-session tracking built into the core SDK. If you need to identify returning visitors, that's an opt-in feature you configure explicitly.`,
      },
      {
        heading: 'Self-Hosted Privacy: The Structural Advantage',
        body: `All of these privacy features exist in SaaS session replay tools too — but with a fundamental difference: your data lives on their servers.\n\nWith PinConsole, every privacy guarantee is structural. When data never leaves your infrastructure:\n\n• There are no data processing agreements (DPAs) to sign\n• There are no international transfer impact assessments (TIA) to conduct\n• There is no third-party vendor risk assessment for the data processor\n• Your data retention policy is enforced by your storage costs, not by a vendor's pricing tier\n\nFor EU companies, this is the difference between "we use a US-based SaaS that stores data on AWS US-East" and "we run the software on our own EU-hosted servers." The second requires significantly less legal overhead to comply with GDPR Chapter V (international transfers).\n\nAnd because we're open source (AGPL-3.0), the privacy guarantees are auditable. You can verify the masking logic, the consent flow, and the erasure cascade by reading the code. You don't need to trust a vendor's SOC 2 report — you can see exactly how the data flows from browser to storage.`,
      },
      {
        heading: 'Privacy as a Feature, Not a Compliance Exercise',
        body: `Building privacy into session replay isn't just about avoiding fines. It's about building trust with your visitors.\n\nWhen a visitor lands on your site and sees a clear consent dialog that explains what will be recorded and gives them real control, their trust increases. When they know their password inputs are never captured, their form submissions are masked by default, and they can request deletion with a simple process, the surveillance aspect of session replay becomes less concerning.\n\nFor us, privacy is not a separate workstream. It's baked into every layer — from the SDK's default-masked inputs to the GC worker's hourly cleanup to the admin panel's erasure UI. Privacy is a first-class feature, tracked as its own vertical slice (1l) alongside auth, replay, and co-browsing.\n\nIf you're evaluating session replay tools and privacy is a concern (it should be), PinConsole offers a fundamentally different approach: self-hosted, auditable, privacy-first by default. Free under AGPL-3.0.`,
      },
    ],
  },
  relatedPosts: [
    { title: 'Defense in Depth for Session Replay: Building Anti-Bot Infrastructure in Go', url: '/en/blog/defense-in-depth/', description: 'Four-layer anti-bot defense: rate limiting, behavioral analysis, SDK fingerprinting, and WebSocket auth.' },
    { title: 'How We Built a Self-Hosted Session Replay Alternative to FullStory', url: '/en/blog/building-self-hosted-session-replay/', description: 'Architecture decisions behind PinConsole — Go, rrweb, MinIO, and single-binary deployment.' },
    { title: 'Building Open-Source Co-Browsing: How We Made Two Browsers Share One Session', url: '/en/blog/building-co-browsing/', description: 'How PinConsole implements privacy-first co-browsing with consent and selective masking.' },
  ],
  cta: {
    title: 'Try Privacy-First Session Replay',
    subtitle: 'Self-hosted. Auditable. GDPR-ready from day one.',
    primary: { label: 'Get started on GitHub', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: 'Talk to the maintainer', href: '#consult' },
  },
};
