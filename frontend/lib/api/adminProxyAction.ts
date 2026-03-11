import { NextResponse } from "next/server";
import { fetchWithTimeout } from "@/lib/api/fetchWithTimeout";
import { getBackendAdminHeaders, isAuthorized } from "@/lib/api/adminAuth";

const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";
const ADMIN_ACTION_TIMEOUT_MS = 30_000;

export function createUnexpectedResponse(
  res: Response,
  backendPath: string,
  contentType: string,
  raw: string,
  reason: 'invalid JSON' | 'non-JSON response'
) {
  if (raw) {
    console.error(`adminProxyAction: upstream returned ${reason}`, {
      backendPath,
      status: res.status,
      contentType,
      body: '<REDACTED_BODY>',
    });
  }

  return NextResponse.json(
    {
      error: 'backend_unexpected_response',
      detail: {
        contentType: contentType || 'unknown',
        body: raw
          ? '<redacted upstream response>'
          : reason === 'invalid JSON'
            ? 'invalid or empty JSON body'
            : 'empty response body',
      },
    },
    { status: res.ok ? 502 : res.status }
  );
}

export function createAdminActionProxy(backendPath: string) {
  return async function POST(req: Request) {
    let headers: HeadersInit;
    try {
      headers = getBackendAdminHeaders();
    } catch (err) {
      return NextResponse.json(
        { error: 'misconfigured_admin_token', detail: err instanceof Error ? err.message : 'unknown' },
        { status: 500 }
      );
    }

    if (!isAuthorized(req)) {
      return NextResponse.json({ error: 'forbidden' }, { status: 403 });
    }

    try {
      const res = await fetchWithTimeout(
        new URL(backendPath, BACKEND_BASE_URL),
        {
          method: 'POST',
          cache: 'no-store',
          headers,
        },
        ADMIN_ACTION_TIMEOUT_MS
      );

      if (res.status === 204) {
        return new NextResponse(null, { status: 204 });
      }

      const contentType = (res.headers.get('content-type') || '').toLowerCase();
      if (contentType.includes('application/json')) {
        try {
          return NextResponse.json(await res.clone().json(), { status: res.status });
        } catch {
          const raw = await res.text().catch(() => '');
          return createUnexpectedResponse(res, backendPath, contentType, raw, 'invalid JSON');
        }
      }

      const raw = await res.text().catch(() => '');
      return createUnexpectedResponse(res, backendPath, contentType, raw, 'non-JSON response');
    } catch (err) {
      return NextResponse.json(
        { error: 'backend_unreachable', detail: err instanceof Error ? err.message : 'unknown' },
        { status: 502 }
      );
    }
  };
}
