import { NextRequest, NextResponse } from "next/server";
import { fetchWithTimeout } from "@/lib/api/fetchWithTimeout";
const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";

export async function GET() {
  try {
    const res = await fetchWithTimeout(new URL('/v1/me/tracked-items', BACKEND_BASE_URL), { cache: 'no-store' });
    return NextResponse.json(await res.json(), { status: res.status });
  } catch (err) {
    return NextResponse.json({ error: 'backend_unreachable', detail: err instanceof Error ? err.message : 'unknown' }, { status: 502 });
  }
}

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
