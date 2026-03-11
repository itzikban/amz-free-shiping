const ADMIN_API_TOKEN = process.env.ADMIN_API_TOKEN;

function hasNonEmptyToken(token: string | undefined): token is string {
  return typeof token === 'string' && token.trim().length > 0;
}

function hasValidAdminHeader(req: Request): boolean {
  if (!hasNonEmptyToken(ADMIN_API_TOKEN)) return false;
  return req.headers.get('x-admin-token') === ADMIN_API_TOKEN;
}

export function isAuthorized(req: Request): boolean {
  return hasValidAdminHeader(req);
}

export function getBackendAdminHeaders(): HeadersInit {
  if (!hasNonEmptyToken(ADMIN_API_TOKEN)) {
    throw new Error('ADMIN_API_TOKEN is not configured');
  }

  return { 'X-Admin-Token': ADMIN_API_TOKEN };
}
