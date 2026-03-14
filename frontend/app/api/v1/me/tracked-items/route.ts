import { NextRequest, NextResponse } from "next/server";
import { fetchWithTimeout } from "@/lib/api/fetchWithTimeout";
const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";

function sanitize(body: Record<string, unknown>) {
  if (Array.isArray(body.items)) {
    body.items = body.items.map((it: Record<string, unknown>) => {
      delete it.method;
      if (typeof it.signal === 'string') it.signal = it.signal.replace(/decodo/gi, 'proxy');
      return it;
    });
  }
  return body;
}

export async function GET() {
  try {
    const res = await fetchWithTimeout(new URL('/v1/me/tracked-items', BACKEND_BASE_URL), { cache: 'no-store' });
    const body = await res.json();
    return NextResponse.json(sanitize(body), { status: res.status });
  } catch (err) {
    return NextResponse.json({ error: 'backend_unreachable', detail: err instanceof Error ? err.message : 'unknown' }, { status: 502 });
  }
}

/**
 * Create or update the current user's tracked items by forwarding the request body to the backend.
 *
 * Reads the request body as JSON and POSTs it to the backend endpoint /v1/me/tracked-items, returning the backend's JSON response and HTTP status.
 *
 * @param req - Incoming request whose JSON body will be sent to the backend
 * @returns The backend response parsed as JSON with the backend HTTP status; if the request body is invalid JSON, returns `{ error: 'invalid_json' }` with status 400; if the backend is unreachable, returns `{ error: 'backend_unreachable', detail }` with status 502
 */
export async function POST(req: NextRequest) {
  let payload: unknown;
  try { payload = await req.json(); } catch { return NextResponse.json({ error: 'invalid_json' }, { status: 400 }); }
  try {
    const res = await fetchWithTimeout(new URL('/v1/me/tracked-items', BACKEND_BASE_URL), {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(payload),
      cache: 'no-store',
    }, 45000);
    return NextResponse.json(await res.json(), { status: res.status });
  } catch (err) {
    return NextResponse.json({ error: 'backend_unreachable', detail: err instanceof Error ? err.message : 'unknown' }, { status: 502 });
  }
}
