# AGPL (OSS Core) / UNLICENSED (Marketing Layer) License Split

The `pinconsole` repository carries two distinct licenses by directory: the
**OSS Core** (`server/`, `admin/`, `visitor-sdk/`, `landing/`, `packages/`,
`e2e/`) is **AGPL-3.0-or-later** as the project's open-source product; the
**Marketing Layer** (`marketing/`) is **UNLICENSED / All Rights Reserved** as
the maintainer's proprietary consulting/lead-gen surface. The split is
physical (separate directory, separate `package.json`, separate deploy
target — Cloudflare Pages + D1, never embedded in the Go binary) so that an
OSS user deploying `pinconsole` cannot accidentally serve or fork the
maintainer's marketing content.

## Considered Options

- **Single AGPL for everything** — rejected: AGPL's share-alike clause would
  force public distribution of blog posts, customer case studies, and the
  lead-capture form, exposing marketing assets to forks by competitors.
- **Move `marketing/` to a separate repo** — deferred: would make the
  license boundary physical at the repo level, but the directory-level
  split already achieves the legal isolation, and keeping everything in one
  repo simplifies cross-cutting changes (shared design tokens, shared
  release notes, single CI surface).
- **BUSL → Apache 2.0 (4y)** — rejected: adds change-date tracking overhead
  for an asset class (marketing content) that has no near-term plan to open.
- **Mixed: code UNLICENSED + content CC BY-NC-ND** — rejected: doubles the
  contributor cognitive load and the difference is moot for a single
  maintainer.

## Consequences

- A future contributor must check the directory before reusing code: OSS
  Core is freely reusable under AGPL terms; `marketing/` requires the
  maintainer's explicit permission.
- The "AGPL may deter enterprise adoption" risk noted in `PLAN.md §10` is
  preserved by keeping the OSS Core under AGPL; the marketing layer's
  proprietary status does not bleed into the OSS distribution.
- `marketing/package.json` `private: true` and the `LICENSE` file in
  `marketing/` are the boundary markers; if either is removed, the split is
  silently broken.
