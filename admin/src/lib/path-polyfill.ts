/**
 * Lightweight path polyfill for browser compatibility.
 *
 * PostCSS (a transitive dependency via Vite/Vue plugin) destructures
 * `require('path')` at module scope in three files:
 *   - `input.js`:       isAbsolute, resolve
 *   - `previous-map.js`: dirname, join
 *   - `map-generator.js`: dirname, relative, resolve, sep
 *
 * Vite externalizes Node built-ins in browser bundles and logs a proxy
 * warning for every accessed key.  This polyfill is aliased in
 * `vite.config.ts` so the dep optimizer bundles it instead, silencing
 * the noisy console warning and providing working no-op implementations.
 */

export function isAbsolute(p: string): boolean {
  return typeof p === 'string' && p.startsWith('/')
}

export function resolve(...paths: string[]): string {
  return paths[paths.length - 1] || '/'
}

export function dirname(p: string): string {
  // PostCSS uses dirname to extract the root directory of a source map
  // file.  In the browser this is never meaningfully called on real
  // filesystem paths; return '/' as a safe fallback.
  if (!p || typeof p !== 'string') return '/'
  const i = p.lastIndexOf('/')
  return i > 0 ? p.slice(0, i) : '/'
}

export function join(...paths: string[]): string {
  // PostCSS uses join to concatenate source map paths.  Minimal
  // join that handles the common case for URL-like paths.
  return paths
    .filter(Boolean)
    .join('/')
    .replace(/\/+/g, '/')
}

export function relative(_from: string, to: string): string {
  // PostCSS's map-generator uses relative to compute source map
  // source paths.  Return the target as-is — correct enough for
  // the browser dev context.
  return to
}

export const sep = '/'

