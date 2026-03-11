import { NextResponse } from "next/server";
import { fetchWithTimeout } from "@/lib/api/fetchWithTimeout";
import { getBackendAdminHeaders, isAuthorized } from "@/lib/api/adminAuth";
import { createUnexpectedResponse } from "@/lib/api/adminProxyAction";

const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";

export async function GET(req: Request) {
  if (!isAuthorized(req)) {
    return NextResponse.json({ error: 'forbidden' }, { status: 403 });
  }

  let headers: HeadersInit;
  try {
    headers = getBackendAdminHeaders();
  } catch (err) {
    return NextResponse.json(
      { error: 'misconfigured_admin_token', detail: err instanceof Error ? err.message : 'unknown' },
      { status: 500 }
    );
  }

  try {
    const res = await fetchWithTimeout(new URL('/v1/admin/metrics', BACKEND_BASE_URL), {
      cache: 'no-store',
      headers,
    });

    if (res.status === 204) {
      return new NextResponse(null, { status: 204 });
    }

    const contentType = (res.headers.get('content-type') || '').toLowerCase();
    if (contentType.includes('application/json')) {
      try {
        return NextResponse.json(await res.clone().json(), { status: res.status });
      } catch (err) {
        return NextResponse.json(
          { error: 'backend_unreachable', detail: err instanceof Error ? err.message : 'invalid_json' },
          { status: 502 }
        );
      }
    }

    const raw = await res.text().catch(() => '');
    return createUnexpectedResponse(res, '/v1/admin/metrics', contentType, raw, 'non-JSON response');
  } catch (err) {
    return NextResponse.json(
      { error: 'backend_unreachable', detail: err instanceof Error ? err.message : 'unknown' },
      { status: 502 }
    );
  }
}
