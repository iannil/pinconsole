-- pinconsole marketing — D1 schema for leads storage
-- D1 is SQLite-based; types follow SQLite conventions.

DROP TABLE IF EXISTS leads;

CREATE TABLE leads (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  company TEXT NOT NULL,
  contact TEXT NOT NULL,           -- phone or email
  purpose TEXT NOT NULL CHECK (purpose IN ('evaluate', 'self-host', 'custom', 'compliance', 'other')),
  message TEXT,
  locale TEXT NOT NULL CHECK (locale IN ('zh', 'en')),
  ip TEXT NOT NULL,                -- already truncated to /24 (IPv4) or /64 (IPv6)
  ua TEXT,
  created_at INTEGER NOT NULL,     -- unix ms
  handled_at INTEGER,
  status TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'contacted', 'qualified', 'closed', 'spam'))
);

CREATE INDEX idx_leads_status ON leads(status);
CREATE INDEX idx_leads_created_at ON leads(created_at DESC);

-- Read access patterns:
--   SELECT * FROM leads WHERE status = 'new' ORDER BY created_at DESC;
--   UPDATE leads SET status = ?, handled_at = ? WHERE id = ?;
