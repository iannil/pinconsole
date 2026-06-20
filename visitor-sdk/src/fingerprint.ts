// Canvas + WebGL + screen fingerprint 采集（1i）
// SDK 启动时采集一次，附加到 hello envelope

export interface Fingerprint {
  canvas_hash: string;
  webgl_vendor: string;
  webgl_renderer: string;
  screen: string;
  timezone: string;
  combined_hash: string;
}

export function collectFingerprint(): Fingerprint {
  const canvasHash = getCanvasHash();
  const { vendor, renderer } = getWebGLInfo();
  const screen = `${window.screen.width}x${window.screen.height}x${window.screen.colorDepth}`;
  const tz = Intl.DateTimeFormat().resolvedOptions().timeZone || 'unknown';
  const combined = simpleHash(canvasHash + vendor + renderer + screen + tz);

  return {
    canvas_hash: canvasHash,
    webgl_vendor: vendor,
    webgl_renderer: renderer,
    screen,
    timezone: tz,
    combined_hash: combined,
  };
}

function getCanvasHash(): string {
  try {
    const canvas = document.createElement('canvas');
    canvas.width = 200;
    canvas.height = 50;
    const ctx = canvas.getContext('2d');
    if (!ctx) return 'no-canvas';
    ctx.textBaseline = 'top';
    ctx.font = '14px Arial';
    ctx.fillText('pinconsole fingerprint 🎯', 2, 2);
    ctx.fillStyle = 'rgba(102,204,0,0.7)';
    ctx.fillRect(125, 1, 62, 20);
    return simpleHash(canvas.toDataURL());
  } catch {
    return 'canvas-blocked';
  }
}

function getWebGLInfo(): { vendor: string; renderer: string } {
  try {
    const canvas = document.createElement('canvas');
    const gl = (canvas.getContext('webgl') || canvas.getContext('experimental-webgl')) as WebGLRenderingContext | null;
    if (!gl) return { vendor: 'no-webgl', renderer: 'no-webgl' };
    const ext = gl.getExtension('WEBGL_debug_renderer_info');
    if (!ext) return { vendor: 'hidden', renderer: 'hidden' };
    return {
      vendor: String(gl.getParameter(ext.UNMASKED_VENDOR_WEBGL)),
      renderer: String(gl.getParameter(ext.UNMASKED_RENDERER_WEBGL)),
    };
  } catch {
    return { vendor: 'webgl-blocked', renderer: 'webgl-blocked' };
  }
}

function simpleHash(s: string): string {
  let hash = 0;
  for (let i = 0; i < s.length; i++) {
    const char = s.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash |= 0;
  }
  return Math.abs(hash).toString(36);
}
