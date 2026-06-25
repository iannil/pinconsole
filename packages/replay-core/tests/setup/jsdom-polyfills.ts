// jsdom polyfills for rrweb Replayer
//
// jsdom v25 does not implement window.scrollTo / window.scroll.
// rrweb's Replayer calls these during replay, so we stub them.
if (typeof window !== 'undefined') {
  const noop = () => {};
  // Force override — jsdom's built-in throws "Not implemented"
  try {
    delete (window as any).scrollTo;
  } catch {
    // ignore if non-deletable
  }
  try {
    (window as any).scrollTo = noop;
  } catch {
    Object.defineProperty(window, 'scrollTo', { value: noop, configurable: true });
  }
  try {
    delete (window as any).scroll;
  } catch {
    // ignore
  }
  try {
    (window as any).scroll = noop;
  } catch {
    Object.defineProperty(window, 'scroll', { value: noop, configurable: true });
  }
}
