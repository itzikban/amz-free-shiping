import { NextRequest, NextResponse } from "next/server";

const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || "http://127.0.0.1:8085";

export async function POST(req: NextRequest) {
  const payload = await req.json();
  try {
    const res = await fetch(new URL("/monitor/start", BACKEND_BASE_URL), {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(payload),
      cache: "no-store",
    });
    return NextResponse.json(await res.json(), { status: res.status });
  } catch (err) {
    return NextResponse.json({ error: "backend_unreachable", detail: err instanceof Error ? err.message : "unknown" }, { status: 502 });
  }
}
