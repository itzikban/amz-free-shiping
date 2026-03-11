import { NextResponse } from "next/server";
import { fetchWithTimeout } from "@/lib/api/fetchWithTimeout";
import { getBackendAdminHeaders, isAuthorized } from "@/lib/api/adminAuth";

const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";

export async function POST(req: Request) {
  if (!isAuthorized(req)) {
    return NextResponse.json({ error: 'forbidden' }, { status: 403 });
  }

  try {
    const res = await fetchWithTimeout(new URL('/v1/admin/actions/replay-failed-jobs', BACKEND_BASE_URL), {
      method: 'POST',
      cache: 'no-store',
      headers: getBackendAdminHeaders(),
    });

    if (res.status === 204) {
      return NextResponse.json({}, { status: 204 });
    }

    const contentType = (res.headers.get('content-type') || '').toLowerCase();
    if (contentType.includes('application/json')) {
      return NextResponse.json(await res.json(), { status: res.status });
    }

    const raw = await res.text();
    return NextResponse.json(
      {
        error: 'backend_unexpected_response',
        detail: { contentType: contentType || 'unknown', body: raw ? raw.slice(0, 200) : 'empty response body' },
      },
      { status: 502 }
    );
  } catch (err) {
    return NextResponse.json(
      { error: 'backend_unreachable', detail: err instanceof Error ? err.message : 'unknown' },
      { status: 502 }
    );
  }
}
