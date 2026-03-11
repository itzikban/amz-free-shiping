import { NextResponse } from "next/server";
import { fetchWithTimeout } from "@/lib/api/fetchWithTimeout";
import { getBackendAdminHeaders, isAuthorized } from "@/lib/api/adminAuth";

const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";

export function createAdminActionProxy(backendPath: string) {
  return async function POST(req: Request) {
    if (!isAuthorized(req)) {
      return NextResponse.json({ error: 'forbidden' }, { status: 403 });
    }

    let headers: HeadersInit;
    try {
      headers = getBackendAdminHeaders();
    } catch (err) {
      return NextResponse.json(
        { error: 'misconfigured_admin_token', detail: err instanceof Error ? err.message : 'unknown' },
        { status: 422 }
      );
    }

    try {
      const res = await fetchWithTimeout(new URL(backendPath, BACKEND_BASE_URL), {
        method: 'POST',
        cache: 'no-store',
        headers,
      });

      if (res.status === 204) {
        return new NextResponse(null, { status: 204 });
      }

      const contentType = (res.headers.get('content-type') || '').toLowerCase();
      if (contentType.includes('application/json')) {
        return NextResponse.json(await res.json(), { status: res.status });
      }

      const raw = await res.text();
      return NextResponse.json(
        {
          error: 'backend_unexpected_response',
          detail: {
            contentType: contentType || 'unknown',
            body: raw || 'empty response body',
          },
        },
        { status: 502 }
      );
    } catch (err) {
      return NextResponse.json(
        { error: 'backend_unreachable', detail: err instanceof Error ? err.message : 'unknown' },
        { status: 502 }
      );
    }
  };
}
