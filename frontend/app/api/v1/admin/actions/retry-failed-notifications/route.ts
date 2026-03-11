import { NextResponse } from "next/server";
import { fetchWithTimeout } from "@/lib/api/fetchWithTimeout";

const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";
const ADMIN_API_TOKEN = process.env.ADMIN_API_TOKEN;

export async function POST() {
  try {
    const res = await fetchWithTimeout(new URL('/v1/admin/actions/retry-failed-notifications', BACKEND_BASE_URL), {
      method: 'POST',
      cache: 'no-store',
      headers: ADMIN_API_TOKEN ? { 'X-Admin-Token': ADMIN_API_TOKEN } : undefined,
    });

    const contentType = (res.headers.get('content-type') || '').toLowerCase();
    if (contentType.includes('application/json')) {
      return NextResponse.json(await res.json(), { status: res.status });
    }

    const raw = await res.text();
    return NextResponse.json(
      {
        error: 'backend_invalid_response',
        detail: raw ? raw.slice(0, 200) : 'empty response body',
      },
      { status: res.status }
    );
  } catch (err) {
    return NextResponse.json(
      { error: 'backend_unreachable', detail: err instanceof Error ? err.message : 'unknown' },
      { status: 502 }
    );
  }
}
