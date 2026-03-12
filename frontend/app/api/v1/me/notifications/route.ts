import { NextRequest, NextResponse } from "next/server";
import { fetchWithTimeout } from "@/lib/api/fetchWithTimeout";
const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";

export async function GET(req: NextRequest) {
  try {
    const query = req.nextUrl.searchParams.toString();
    const path = query ? `/v1/me/notifications?${query}` : '/v1/me/notifications';
    const res = await fetchWithTimeout(new URL(path, BACKEND_BASE_URL), { cache: 'no-store' });
    return NextResponse.json(await res.json(), { status: res.status });
  } catch (err) {
    return NextResponse.json({ error: 'backend_unreachable', detail: err instanceof Error ? err.message : 'unknown' }, { status: 502 });
  }
}
