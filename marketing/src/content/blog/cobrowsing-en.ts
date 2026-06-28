import type { BlogContent } from './types';

export const cobrowsingEn: BlogContent = {
  locale: 'en',
  htmlLang: 'en-US',
  meta: {
    title: 'Building Open-Source Co-Browsing: How We Made Two Browsers Share One Session',
    description: 'A technical deep-dive into PinConsole\'s co-browsing implementation — rrweb node ID selectors, 1:1 operator locking, 300ms form-fill debounce, and the hub-and-spoke WebSocket architecture that makes it all work.',
    ogTitle: 'Building Open-Source Co-Browsing: Two Browsers, One Session',
    ogDescription: 'rrweb node IDs, 1:1 claim locking, and WebSocket hub routing — the architecture behind PinConsole\'s open-source co-browsing.',
  },
  blog: {
    author: 'Rong Zhu',
    publishedDate: '2026-06-28',
    readingTime: '10 min read',
    tags: ['Engineering', 'Co-Browsing', 'WebSocket', 'Open Source'],
  },
  hero: {
    h1: 'Building Open-Source Co-Browsing: How We Made Two Browsers Share One Session',
    subtitle: 'The architecture, trade-offs, and safety mechanisms behind PinConsole\'s bidirectional co-browsing — built entirely on rrweb and a hub-and-spoke WebSocket model.',
  },
  content: {
    sections: [
      {
        heading: 'Why Co-Browsing Is Harder Than Session Replay',
        body: `Session replay is a solved problem. You record DOM mutations as the visitor interacts, serialize them into an event stream, and play them back later. It's fire-and-forget — the operator watches a recording of what happened.\n\nCo-browsing is the inverse. The operator needs to reach into the visitor's live browser and make things happen — click a button, scroll to a section, fill in a form field. And it has to feel natural. No half-second delays, no visual glitches, no security holes.\n\nThis is why most co-browsing tools are expensive SaaS products. Cobrowse.io charges $149/month per operator. Upscope starts at $99/month. And none of them are open source.\n\nWe built PinConsole's co-browsing to change that. It's free, self-hosted, and built entirely on the same rrweb foundation that powers our session replay. This is how it works under the hood.`,
      },
      {
        heading: 'Why rrweb Node IDs Instead of CSS Selectors',
        body: `When an operator clicks on a replayed page, the server needs to tell the visitor's browser: "click on this element." The obvious approach is a CSS selector — something like div.container > button.btn-primary:nth-child(3).\n\nWe don't do that. Here's why:\n\nCSS selectors are fragile. A dynamic class change, a re-render that shifts DOM order, or an A/B test that swaps element positions breaks the selector. The operator clicked "Buy Now" but the visitor's browser highlights the footer link instead. That's not just a bug — it's a support disaster.\n\nrrweb solves this elegantly. Every DOM node that rrweb captures is assigned a stable data-rr-node-id attribute. This ID is deterministic for the same DOM tree — two snapshots of the same page produce the same node IDs. And rrweb tracks node additions and removals, so IDs remain valid even as the DOM mutates.\n\nWhen the operator clicks on the rrweb replay canvas, we use elementFromPoint() to find the clicked element, walk up the DOM tree until we find a node with data-rr-node-id, and send that ID to the visitor's browser. The visitor's SDK then calls document.querySelector([data-rr-node-id="<id>"]) to find the target and executes the action.\n\nThis approach survives full-page re-renders, CSS class flips, and dynamic content swaps. As long as rrweb's DOM snapshot logic is consistent (and it is — it's been battle-tested across millions of sessions), the co-browsing target is always correct.`,
        code: `// Admin overlay: find the clicked element's rrweb node ID
function findRRWebNode(target: Element): number | null {
  let el: Element | null = target;
  while (el) {
    const id = el.getAttribute('data-rr-node-id');
    if (id !== null) return parseInt(id, 10);
    el = el.parentElement;
  }
  return null;
}`,
        codeLanguage: 'typescript',
      },
      {
        heading: 'Architecture: Hub-and-Spoke WebSocket Routing',
        body: `Co-browsing traffic flows through a central Go server. There is no P2P, no WebRTC, no third-party relay. Every message — visitor DOM mutation, operator cursor move, form fill instruction — passes through the hub.\n\nWe chose coder/websocket for the WebSocket layer. It's minimal, idiomatic Go with no global state, no implicit goroutine spawning, and a clean API. The hub pattern:\n\n• Each site is a "room" identified by its site ID\n• Visitor SDKs connect to their site's room\n• Operators connect to the same room\n• Messages are routed by type: visitor events fan out to subscribed operators, operator commands are sent directly to the target visitor\n\nThe critical design decision: we do NOT broadcast all events to all operators. Each session is 1:1 locked. This avoids fan-out overhead and prevents the nightmare scenario of two operators fighting for control of the same session.\n\nFor the reverse path (operator → visitor), the server maintains a visitorClients map: session ID → WebSocket connection ID. When an operator submits a command, the server looks up the visitor's connection and pushes the message directly.`,
        code: `// Simplified hub routing
type Hub struct {
  sessions      map[uuid.UUID]*SessionChan  // session → subscriber channels
  visitorClients map[uuid.UUID]uuid.UUID    // session → visitor client ID
  clients       map[uuid.UUID]*Client       // client ID → WebSocket conn
}

// Visitor sends event → all subscribed operators
func (h *Hub) PublishEvent(sessionID uuid.UUID, msg []byte) {
  if ch, ok := h.sessions[sessionID]; ok {
    ch.publish(msg)  // non-blocking fan-out to all subs
  }
}

// Operator sends command → specific visitor
func (h *Hub) SendCommandToVisitor(sessionID uuid.UUID, msg []byte) {
  if visitorID, ok := h.visitorClients[sessionID]; ok {
    if client, ok := h.clients[visitorID]; ok {
      client.Send(msg)
    }
  }
}`,
        codeLanguage: 'go',
      },
      {
        heading: '1:1 Locking: The Atomic Claim Protocol',
        body: `Co-browsing with multiple operators is chaos without ownership discipline. Imagine two support agents both trying to fill in the same form field. We prevent this with an atomic claim/release protocol backed by Redis.\n\nThe flow:\n\n1. Operator clicks "Start co-browsing" → POST /api/sessions/:id/claim\n2. Server executes Redis SET NX EX 300 on the key claim:session:<uuid>\n3. If the key doesn't exist, the claim succeeds and the operator is recorded as the owner (value = user_id)\n4. If the key exists, the operator sees "Session is being assisted by Alice"\n5. The admin SPA sends a keep-alive every 60 seconds to refresh the TTL\n6. When the operator clicks "Release" or closes the session, POST /api/sessions/:id/release executes a Lua script that atomically checks ownership and deletes the key\n\nThe claim is enforced at three levels:\n\n• HTTP API — all command endpoints check claim ownership via Redis before accepting the command\n• Hub routing — only commands from the claimed operator are forwarded to the visitor\n• Audit log — every command is logged to the co_browsing_commands table with the operator_id, so you can trace who did what\n\nThis "claim as capability" pattern means the system is safe by default. An operator who hasn't claimed a session can watch events but cannot send commands.`,
        code: `-- Lua: atomic claim release with ownership check
local key = KEYS[1]
local expectedOwner = ARGV[1]
local current = redis.call('GET', key)
if current == expectedOwner then
  return redis.call('DEL', key)
end
return 0`,
        codeLanguage: 'lua',
      },
      {
        heading: 'The Overlay: Capturing Operator Actions on a Replay Canvas',
        body: `The operator's co-browsing UI is a transparent overlay on top of the rrweb player. When the operator enables co-browsing, the player enters a "live interaction" mode:\n\n• An invisible div (z-index: 10, cursor: crosshair) sits over the replay iframe\n• Mouse movement is captured at 30fps (rAF-throttled) and sent as cursor_highlight commands\n• Clicks are debounced at 200ms to prevent double-fires\n• The clicked coordinates are translated through the rrweb iframe to find the correct DOM node\n\nCursor highlighting deserves special attention. The operator's cursor position is sent to the visitor's browser, which renders a fixed-position SVG circle with the operator's name. The circle follows the operator's movements with a smooth CSS transition (transform: translate() 50ms linear), creating the illusion that someone is pointing at things on the visitor's actual screen.\n\nFor scrolling, we send window.scrollTo(x, y) commands. This is intentionally simple — complex scroll animations on the operator's side don't need to be replicated frame-by-frame. The visitor's browser jumps to the correct position, which is good enough for "look at this section below."`,
      },
      {
        heading: 'Form Filling: The Hardest Co-Browsing Problem',
        body: `Filling in a form field on someone else's browser sounds trivial: set the value and dispatch an event. But modern frontend frameworks (React, Vue, Angular) don't listen to value property changes — they listen to synthetic input events with specific properties.\n\nIf you just do element.value = 'text' and dispatch a plain Event('input'), React won't see it. The framework uses a property descriptor on HTMLInputElement.prototype that intercepts the native setter. You need to invoke that setter directly.\n\nOur solution:\n\n1. Save the native value property descriptor from HTMLInputElement.prototype\n2. Call the native setter to update the value\n3. Dispatch both 'input' and 'change' events with bubbles: true and cancelable: true\n4. The framework picks up the change and updates its virtual DOM\n\nWe also added a 300ms debounce per field. When the operator types in a form field, each keystroke is sent as a separate fill_input command. The 300ms debounce prevents overlapping writes — if the visitor's network is slow, commands queue up and execute sequentially.\n\nDuring form filling, the visitor sees a subtle blue border on the active field and a toast notification: "Operator is filling in [field label]." This transparency is critical for trust — the visitor always knows when an operator is interacting with their page.\n\nAfter 5 seconds of inactivity on a field, the visual lock is released automatically. The operator can also explicitly release control of all fields.`,
        code: `// Invoke native value setter for framework compatibility
function setNativeValue(element: HTMLInputElement, value: string) {
  const nativeSetter = Object.getOwnPropertyDescriptor(
    window.HTMLInputElement.prototype, 'value'
  )?.set;
  nativeSetter?.call(element, value);
  element.dispatchEvent(new Event('input', { bubbles: true }));
  element.dispatchEvent(new Event('change', { bubbles: true }));
}`,
        codeLanguage: 'typescript',
      },
      {
        heading: 'Safety by Design: URL Validation, Command Whitelist, and Emergency Exits',
        body: `Giving an operator the ability to control a visitor's browser is a security-sensitive feature. We built multiple safety layers:\n\nCommand whitelist. Only 8 command types are accepted: cursor_highlight, click, scroll, fill_input, navigate, release_control, show_popup, and chat_message. Any unknown type is silently dropped.\n\nURL validation for navigation. When an operator sends a navigate command, the server validates the target URL against the session's origin. Same-origin navigations are allowed freely. Cross-origin navigations are blocked unless the site operator has explicitly whitelisted the domain in the admin panel. This prevents an operator from redirecting a visitor to a malicious site.\n\nPopup URL sanitization. The show_popup command's action_url is validated against a strict scheme whitelist: only https: and mailto: are allowed. javascript: and data: URLs are rejected at the API layer.\n\nVisitor-side emergency exits. The visitor can immediately end a co-browsing session by:\n\n• Pressing Escape three times within 1 second\n• Pressing Ctrl+Shift+X\n• Clicking the "Exit" button on the co-browsing banner\n\nAll three triggers call the onReleased() callback, which removes the operator cursor, clears visual locks, and closes the co-browsing channel. The operator sees "Session released by visitor."\n\nThese safety measures aren't theoretical — they're validated by our e2e test suite (Playwright fork-5 tests that simulate full co-browsing sessions including emergency exits).`,
      },
      {
        heading: 'GDPR Consent: Transparent by Default',
        body: `Co-browsing involves a human operator seeing a visitor's screen in real time. This has obvious privacy implications under GDPR, CCPA, and similar regulations.\n\nWhen an operator initiates co-browsing, the visitor's SDK triggers a consent banner before any operator input is applied. The banner shows:\n\n• An SVG eye icon (visual indicator that someone is watching)\n• The operator's name (so the visitor knows who)\n• An "Exit" button to immediately terminate\n\nThe banner is fixed at the top of the viewport with z-index: 999999, ensuring it's always visible. It stays visible for the entire co-browsing session. If the visitor dismisses it, co-browsing continues but the banner can be re-shown for critical interactions (like form filling).\n\nThis isn't just good UX — it's legally required in most jurisdictions. And because PinConsole is self-hosted, you can customize the consent text, styling, and behavior to match your specific compliance requirements.\n\nFor sites that need stricter controls, we also support a passive monitoring mode. In this mode, the operator can see the visitor's screen but cannot interact until the visitor explicitly clicks "Allow assistance." This is the default for financial services and healthcare deployments.`,
      },
      {
        heading: 'Observability: Tracing Operator Actions End-to-End',
        body: `When something goes wrong during co-browsing — a command that didn't execute, a visitor who disconnected — you need to trace what happened. We instrument the entire command lifecycle with trace_id propagation.\n\nWhen an operator sends a command, the server generates a trace_id and embeds it in the MsgCommand envelope. The visitor's SDK receives the command, executes it, and caches the trace_id for up to 5 seconds (or 10 events). Any rrweb events generated as a result of the command carry this same trace_id back through the server to the admin panel.\n\nThis creates an end-to-end trace loop:\n\noperator click → [trace_id: abc123] → server → [trace_id: abc123] → SDK executes click → [trace_id: abc123] → rrweb records resulting DOM mutation → [trace_id: abc123] → server stores event → [trace_id: abc123] → admin panel displays event with trace\n\nIn practice, this means an operator who sends a "Click" command and sees no visual feedback can look up the trace: was the command received? Was it executed? Did the DOM mutation fire back? The answer is in the logs, all linked by the same trace_id.\n\nEvery command is also logged to the co_browsing_commands PostgreSQL table with the operator_id, session_id, command_type, target_node_id, and payload. This audit trail is invaluable for debugging and compliance.`,
      },
      {
        heading: 'Comparison: How We Stack Against Dedicated Co-Browsing Tools',
        body: `Here's how PinConsole's co-browsing compares to the dedicated tools:\n\n| Feature | PinConsole | Cobrowse.io | Upscope |\n|---|---|---|---|\n| **Pricing** | Free (AGPL) | $149/op/month | $99/op/month |\n| **Self-hosted** | ✓ | ✗ (Enterprise only) | ✗ |\n| **Open source** | ✓ AGPL-3.0 | ✗ | ✗ |\n| **Session replay + co-browsing** | ✓ Unified | Separate products | Separate products |\n| **Form fill** | ✓ (300ms debounce) | ✓ | ✓ |\n| **Navigation** | ✓ (same-origin) | ✓ | ✓ |\n| **Visitor consent banner** | ✓ (GDPR-ready) | ✓ | ✓ |\n| **Operator locking** | ✓ Atomic claim | ✓ | ✓ |\n| **Emergency exit** | ✓ Triple-Escape | ✓ | ✓ |\n| **Co-browsing audit log** | ✓ PostgreSQL | Limited | Limited |\n| **End-to-end tracing** | ✓ trace_id loop | ✗ | ✗ |\n\nThe key differentiator is integration. Because our co-browsing runs on the same platform as session replay and live monitoring, an operator can watch a visitor's session, identify the pain point, and jump into co-browsing without switching tools. The visitor's context is preserved across the transition — there's no "new session" handoff, no lost history.\n\nAt $0/month for unlimited operators (self-hosted), the cost argument is compelling. But the real value is in the unified workflow: session replay → live monitoring → co-browsing, all in one self-hosted binary.`,
      },
      {
        heading: 'What\'s Next for Co-Browsing',
        body: `Our co-browsing implementation covers the core use cases — click, scroll, form fill, navigation, and chat. But there's room to grow:\n\n• Mobile co-browsing — the rrweb node ID approach works on mobile browsers, but the overlay UX needs to be rethought for touch interfaces. We're exploring a "mirror mode" where the operator sees a phone-sized viewport.\n\n• Multi-operator handoff — currently 1:1 locked. A handoff protocol would let one operator transfer control to another without disconnecting the visitor.\n\n• Screen sharing fallback — for complex interactions that our command protocol can't handle (drag-and-drop, canvas drawing), a WebRTC-based screen share fallback would cover the edge cases.\n\n• Co-browsing recording — recording the operator's interactions alongside the visitor's session for training and quality assurance.\n\nThese are on the post-v1 backlog. The current implementation already handles the 90% use case: a support agent helping a visitor navigate a web application in real time.\n\nIf you're building a product that needs co-browsing — or you're tired of paying $149/operator for a feature that should be part of your session replay tool — PinConsole is free, self-hosted, and open source. Try it in 5 minutes.`,
        code: `git clone https://github.com/iannil/pinconsole\ncd pinconsole\ndocker compose up -d`,
        codeLanguage: 'bash',
      },
    ],
  },
  relatedPosts: [
    { title: 'How We Built a Self-Hosted Session Replay Alternative to FullStory', url: '/en/blog/building-self-hosted-session-replay/', description: 'The full architecture behind PinConsole — Go, rrweb, MinIO, and single-binary deployment.' },
    { title: 'Privacy by Design in Open-Source Session Replay', url: '/en/blog/privacy-by-design/', description: 'How PinConsole implements GDPR-first session monitoring with opt-in consent and selective masking.' },
  ],
  cta: {
    title: 'Try Co-Browsing on PinConsole',
    subtitle: 'Self-hosted session replay + co-browsing in one binary. Free. AGPL-3.0.',
    primary: { label: 'Get started on GitHub', href: 'https://github.com/iannil/pinconsole' },
    secondary: { label: 'Talk to the maintainer', href: '#consult' },
  },
};
