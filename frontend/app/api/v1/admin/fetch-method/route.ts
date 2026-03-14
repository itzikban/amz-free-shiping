import { NextRequest, NextResponse } from "next/server";
import { fetchWithTimeout } from "@/lib/api/fetchWithTimeout";
const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";

export async function GET() {
  try {
    const res = await fetchWithTimeout(new URL('/v1/admin/fetch-method', BACKEND_BASE_URL), { cache: 'no-store' });
    return NextResponse.json(await res.json(), { status: res.status });
  } catch {
    return NextResponse.json({ method: 'auto' }, { status: 200 });
  }
}

export async function PUT(req: NextRequest) {
  try {
    const payload = await req.json();
    const res = await fetchWithTimeout(new URL('/v1/admin/fetch-method', BACKEND_BASE_URL), {
      method: 'PUT',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(payload),
      cache: 'no-store',
    });
    return NextResponse.json(await res.json(), { status: res.status });
  } catch {
    return NextResponse.json({ error: 'backend_unreachable' }, { status: 502 });
  }
}
