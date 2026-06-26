/**
 * Lightweight stubs for Node.js built-in / native modules that PostCSS
 * relies on but are externalized by Vite in browser bundles.
 *
 * PostCSS destructures these at module scope, triggering Vite's externalized
 * proxy warning on every accessed key.  These stubs are aliased in
 * `vite.config.ts` so the dep optimizer bundles them instead.
 *
 * Each stub is kept as small as possible — just enough to silence the
 * console noise and allow PostCSS's own feature-detection guards
 * (e.g. `sourceMapAvailable`) to work correctly.
 */

// ---- source-map-js ----
// PostCSS guards every call site with `sourceMapAvailable`, so undefined
// classes mean source-map features quietly no-op in the browser.
export const SourceMapConsumer = undefined
export const SourceMapGenerator = undefined

// ---- url ----
// Used with `if (fileURLToPath)` / `if (pathToFileURL)` guards.
export function fileURLToPath(url: string): string {
  return url
}

export function pathToFileURL(path: string): URL {
  return new URL(`file://${path}`)
}

// ---- fs ----
// `existsSync` returns false so `loadFile` skips the `readFileSync` branch.
export function existsSync(_path: string): boolean {
  return false
}

export function readFileSync(_path: string, _encoding: string): string {
  // Should never be called because existsSync returns false above.
  return ''
}
