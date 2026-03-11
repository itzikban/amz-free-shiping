const ADMIN_API_TOKEN = process.env.ADMIN_API_TOKEN;

function hasNonEmptyToken(token: string | undefined): token is string {
  return typeof token === 'string' && token.trim().length > 0;
}

function getCookie(req: Request, name: string): string {
  const raw = req.headers.get('cookie') || '';
  const parts = raw.split(';').map((x) => x.trim());
  for (const part of parts) {
    if (!part) continue;
    const eq = part.indexOf('=');
    if (eq <= 0) continue;
    const k = part.slice(0, eq).trim();
    if (k !== name) continue;
    return decodeURIComponent(part.slice(eq + 1));
  }
  return '';
}

function hasValidAdminHeader(req: Request): boolean {
  if (!hasNonEmptyToken(ADMIN_API_TOKEN)) return false;
  return req.headers.get('x-admin-token') === ADMIN_API_TOKEN;
}

function hasValidAdminCookie(req: Request): boolean {
  if (!hasNonEmptyToken(ADMIN_API_TOKEN)) return false;
  return getCookie(req, 'admin_session') === ADMIN_API_TOKEN;
}

export function isAuthorized(req: Request): boolean {
  return hasValidAdminHeader(req) || hasValidAdminCookie(req);
}

export function getBackendAdminHeaders(): HeadersInit {
  if (!hasNonEmptyToken(ADMIN_API_TOKEN)) {
    throw new Error('ADMIN_API_TOKEN is not configured');
  }

  return { 'X-Admin-Token': ADMIN_API_TOKEN };
}
