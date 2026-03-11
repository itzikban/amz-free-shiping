import { NextResponse } from "next/server";
import { fetchWithTimeout } from "@/lib/api/fetchWithTimeout";

const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";
const ADMIN_API_TOKEN = process.env.ADMIN_API_TOKEN;

function isAuthorized(req: Request): boolean {
  const sameOrigin = req.headers.get('sec-fetch-site') === 'same-origin';
  if (sameOrigin) return true;
  if (!ADMIN_API_TOKEN) return false;
  return req.headers.get('x-admin-token') === ADMIN_API_TOKEN;
}

export async function POST(req: Request) {
  if (!isAuthorized(req)) {
    return NextResponse.json({ error: 'forbidden' }, { status: 403 });
  }

  try {
    const res = await fetchWithTimeout(new URL('/v1/admin/actions/replay-failed-jobs', BACKEND_BASE_URL), {
      method: 'POST',
      cache: 'no-store',
      headers: { 'X-Admin-Token': ADMIN_API_TOKEN as string },
    });

    const contentType = (res.headers.get('content-type') || '').toLowerCase();
    if (contentType.includes('application/json')) {
      return NextResponse.json(await res.json(), { status: res.status });
    }

    const raw = await res.text();
    return NextResponse.json(
      { error: 'backend_error', detail: raw || res.statusText || 'unknown' },
      { status: res.status }
    );
  } catch (err) {
    return NextResponse.json(
      { error: 'backend_unreachable', detail: err instanceof Error ? err.message : 'unknown' },
      { status: 502 }
    );
  }
}
